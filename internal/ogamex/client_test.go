package ogamex

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractCSRFToken(t *testing.T) {
	t.Parallel()

	html := `<html><head><meta name="csrf-token" content="test-token-123"></head><body></body></html>`
	token, err := extractCSRFToken(strings.NewReader(html))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-token-123" {
		t.Errorf("expected token 'test-token-123', got '%s'", token)
	}
}

func TestExtractCSRFToken_Missing(t *testing.T) {
	t.Parallel()

	html := `<html><head></head><body></body></html>`
	_, err := extractCSRFToken(strings.NewReader(html))
	if err == nil {
		t.Fatal("expected error when csrf token missing")
	}
}

func TestLoginFlow(t *testing.T) {
	t.Parallel()

	var loginPOSTCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/login":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><meta name="csrf-token" content="srv-csrf-token"></head><body><form></form></body></html>`))
		case r.Method == http.MethodPost && r.URL.Path == "/login":
			loginPOSTCalled = true
			r.ParseForm()
			if r.Form.Get("email") != "test@test.com" || r.Form.Get("password") != "pass123" {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`<html><head><meta name="csrf-token" content="srv-csrf-token"></head><body>login failed</body></html>`))
				return
			}
			if r.Form.Get("_token") != "srv-csrf-token" {
				w.WriteHeader(http.StatusLocked)
				return
			}
			w.Header().Set("Location", "/overview")
			w.WriteHeader(http.StatusFound)
		case r.URL.Path == "/overview":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("overview page"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@test.com", "pass123", slog.Default())
	err := client.Login(context.Background())
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if !loginPOSTCalled {
		t.Fatal("login POST was not called")
	}
	if tok := client.getCSRFToken(); tok != "srv-csrf-token" {
		t.Errorf("expected csrf token 'srv-csrf-token', got '%s'", tok)
	}
}

func TestLoginFailure_WrongCredentials(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><meta name="csrf-token" content="tok"></head><body></body></html>`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><meta name="csrf-token" content="tok"></head><body>bad credentials</body></html>`))
			return
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad@bad.com", "wrong", slog.Default())
	err := client.Login(context.Background())
	if err == nil {
		t.Fatal("expected login to fail with wrong credentials")
	}
}

func TestCSRFTokenRefresh(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><meta name="csrf-token" content="initial-tok"></head><body></body></html>`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/ajax/test" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"newAjaxToken":"refreshed-token-xyz"}`))
			return
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test@test.com", "pass", slog.Default())
	client.setCSRFToken("initial-tok")

	_, err := client.doAJAXPost(context.Background(), "/ajax/test", nil)
	if err != nil {
		t.Fatalf("doAJAXPost failed: %v", err)
	}

	if tok := client.getCSRFToken(); tok != "refreshed-token-xyz" {
		t.Errorf("expected token 'refreshed-token-xyz', got '%s'", tok)
	}
}

func TestReauthOn419(t *testing.T) {
	t.Parallel()

	var postAttempts int
	loginSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><meta name="csrf-token" content="fresh-token"></head><body></body></html>`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/login" {
			w.Header().Set("Location", "/overview")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/ajax/action" {
			postAttempts++
			if postAttempts == 1 {
				w.WriteHeader(http.StatusLocked)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"newAjaxToken":"after-reauth"}`))
			return
		}
	}))
	defer loginSrv.Close()

	client := NewClient(loginSrv.URL, "t@t.com", "p", slog.Default())
	client.setCSRFToken("stale-token")

	_, err := client.doPost(context.Background(), "/ajax/action", nil)
	if err != nil {
		t.Fatalf("doPost after reauth failed: %v", err)
	}
	if postAttempts != 2 {
		t.Errorf("expected 2 post attempts, got %d", postAttempts)
	}
}
