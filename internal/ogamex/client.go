package ogamex

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/user/ogame-bot/internal/ogamed"
)

var _ ogamed.ClientInterface = (*Client)(nil)

type Client struct {
	baseURL    string
	httpClient *http.Client
	csrfToken  string
	email      string
	password   string
	log        *slog.Logger
	mu         sync.Mutex
}

func NewClient(baseURL, email, password string, log *slog.Logger) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		email:    email,
		password: password,
		log:      log.With("component", "ogamex-client"),
	}
}

func (c *Client) getCSRFToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.csrfToken
}

func (c *Client) setCSRFToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.csrfToken = token
}

func (c *Client) GetCaptchaChallenge(ctx context.Context) (ogamed.CaptchaChallenge, error) {
	return ogamed.CaptchaChallenge{}, fmt.Errorf("ogamex: GetCaptchaChallenge not implemented")
}

func (c *Client) SolveCaptchaChallenge(ctx context.Context, challengeID string, answer int) error {
	return fmt.Errorf("ogamex: SolveCaptchaChallenge not implemented")
}
