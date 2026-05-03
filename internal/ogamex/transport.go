package ogamex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
	retryMaxDelay  = 10 * time.Second
)

type ajaxResponse struct {
	NewAjaxToken string `json:"newAjaxToken"`
}

type permanentError struct {
	err error
}

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

func isRetryableHTTPStatus(code int) bool {
	return code == http.StatusInternalServerError ||
		code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable
}

func (c *Client) retryWithBackoff(ctx context.Context, path string, fn func() error) error {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		var pe *permanentError
		if errors.As(err, &pe) {
			return pe.err
		}
		if attempt == maxRetries {
			return err
		}
		delay := retryBaseDelay * time.Duration(1<<uint(attempt))
		if delay > retryMaxDelay {
			delay = retryMaxDelay
		}
		c.log.Warn("Retrying request", "path", path, "attempt", attempt, "delay", delay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			continue
		}
	}
	return nil
}

func (c *Client) doGet(ctx context.Context, path string) ([]byte, error) {
	var result []byte
	err := c.retryWithBackoff(ctx, path, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return &permanentError{fmt.Errorf("creating GET request: %w", err)}
		}

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("executing GET %s: %w", path, err)
		}
		defer resp.Body.Close()

		c.log.Debug("GET request completed",
			"path", path,
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		if isRetryableHTTPStatus(resp.StatusCode) {
			return fmt.Errorf("server error: HTTP %d on GET %s", resp.StatusCode, path)
		}

		if isRedirectToLogin(resp) {
			c.log.Info("Session expired, re-authenticating")
			if err := c.Login(ctx); err != nil {
				return &permanentError{fmt.Errorf("re-login failed: %w", err)}
			}
			req2, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
			if err != nil {
				return &permanentError{fmt.Errorf("creating retry GET request: %w", err)}
			}
			resp2, err := c.httpClient.Do(req2)
			if err != nil {
				return &permanentError{fmt.Errorf("retry GET %s failed: %w", path, err)}
			}
			defer resp2.Body.Close()
			if isRedirectToLogin(resp2) {
				return &permanentError{fmt.Errorf("still redirected to login after re-auth")}
			}
			body, err := readBody(resp2)
			if err != nil {
				return &permanentError{err}
			}
			result = body
			return nil
		}

		body, err := readBody(resp)
		if err != nil {
			return &permanentError{err}
		}
		result = body
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) doPost(ctx context.Context, path string, data url.Values) ([]byte, error) {
	if data == nil {
		data = url.Values{}
	}
	data.Set("_token", c.getCSRFToken())

	body, retry, err := c.executePost(ctx, path, data)
	if err != nil {
		return nil, err
	}
	if retry {
		data.Set("_token", c.getCSRFToken())
		body, _, err = c.executePost(ctx, path, data)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

func (c *Client) executePost(ctx context.Context, path string, data url.Values) ([]byte, bool, error) {
	var result struct {
		statusCode int
		body       []byte
		loginURL   bool
	}

	err := c.retryWithBackoff(ctx, path, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(data.Encode()))
		if err != nil {
			return &permanentError{fmt.Errorf("creating POST request: %w", err)}
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("executing POST %s: %w", path, err)
		}
		defer resp.Body.Close()

		c.log.Debug("POST request completed",
			"path", path,
			"status", resp.StatusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		if isRetryableHTTPStatus(resp.StatusCode) {
			return fmt.Errorf("server error: HTTP %d on POST %s", resp.StatusCode, path)
		}

		body, err := readBody(resp)
		if err != nil {
			return &permanentError{err}
		}
		result.statusCode = resp.StatusCode
		result.body = body
		result.loginURL = strings.Contains(resp.Request.URL.Path, "/login")
		return nil
	})

	if err != nil {
		return nil, false, err
	}

	if result.statusCode == http.StatusLocked || result.statusCode == 419 {
		c.log.Info("CSRF token mismatch, re-authenticating")
		if lerr := c.Login(ctx); lerr != nil {
			return nil, false, fmt.Errorf("re-login after 419 failed: %w", lerr)
		}
		return nil, true, nil
	}

	if result.statusCode == http.StatusUnauthorized || result.loginURL {
		c.log.Info("Session expired on POST, re-authenticating")
		if lerr := c.Login(ctx); lerr != nil {
			return nil, false, fmt.Errorf("re-login after 401 failed: %w", lerr)
		}
		return nil, true, nil
	}

	c.tryRefreshToken(result.body)
	return result.body, false, nil
}

func (c *Client) doAJAX(ctx context.Context, method, path string, data url.Values) ([]byte, error) {
	var body []byte
	var err error
	switch method {
	case http.MethodGet:
		body, err = c.doGet(ctx, path)
	case http.MethodPost:
		body, err = c.doPost(ctx, path, data)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
	if err != nil {
		return nil, err
	}
	c.tryRefreshToken(body)
	return body, nil
}

func (c *Client) doAJAXGet(ctx context.Context, path string) ([]byte, error) {
	return c.doAJAX(ctx, http.MethodGet, path, nil)
}

func (c *Client) doAJAXPost(ctx context.Context, path string, data url.Values) ([]byte, error) {
	return c.doAJAX(ctx, http.MethodPost, path, data)
}

func (c *Client) tryRefreshToken(body []byte) {
	var ajax ajaxResponse
	if err := json.Unmarshal(body, &ajax); err == nil && ajax.NewAjaxToken != "" {
		c.setCSRFToken(ajax.NewAjaxToken)
		c.log.Debug("CSRF token refreshed from response")
	}
}

func isRedirectToLogin(resp *http.Response) bool {
	return strings.Contains(resp.Request.URL.Path, "/login")
}

func readBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return body, nil
}

func toReader(body []byte) io.Reader {
	return bytes.NewReader(body)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
