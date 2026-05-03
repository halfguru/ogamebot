package ogamex

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func extractCSRFToken(body io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}
	token, exists := doc.Find("meta[name='csrf-token']").Attr("content")
	if !exists {
		return "", fmt.Errorf("csrf token not found in login page")
	}
	return token, nil
}

func (c *Client) Login(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/login", nil)
	if err != nil {
		return fmt.Errorf("creating login page request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching login page: %w", err)
	}
	defer resp.Body.Close()

	token, err := extractCSRFToken(resp.Body)
	if err != nil {
		return fmt.Errorf("extracting csrf token: %w", err)
	}
	c.setCSRFToken(token)
	c.log.Debug("CSRF token extracted from login page", "token", token[:min(8, len(token))])

	data := url.Values{}
	data.Set("email", c.email)
	data.Set("password", c.password)
	data.Set("_token", c.getCSRFToken())

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/login", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK && strings.Contains(resp.Request.URL.Path, "/login") {
		return fmt.Errorf("login failed: credentials rejected (status 200, still on login page)")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return fmt.Errorf("login failed: status %d", resp.StatusCode)
	}

	if newToken, err := extractCSRFTokenFromResponse(resp); err == nil && newToken != "" {
		c.setCSRFToken(newToken)
	}

	c.log.Info("Logged in to OGameX", "url", c.baseURL)
	return nil
}

func (c *Client) Logout(ctx context.Context) error {
	_, err := c.doGet(ctx, "/logout")
	if err != nil {
		c.log.Warn("Logout request failed", "error", err)
	}
	c.setCSRFToken("")
	c.log.Info("Logged out from OGameX")
	return nil
}

func extractCSRFTokenFromResponse(resp *http.Response) (string, error) {
	if resp.Body == nil {
		return "", fmt.Errorf("no body")
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return "", fmt.Errorf("not HTML")
	}
	return extractCSRFToken(resp.Body)
}
