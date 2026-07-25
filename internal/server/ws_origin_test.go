package server

import (
	"net/http"
	"testing"
)

func newWSReq(t *testing.T, rawURL, host, origin string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}

func TestWsCheckOrigin(t *testing.T) {
	// No t.Parallel: subtests mutate GO_ENV via t.Setenv.

	t.Run("development allows cross origin", func(t *testing.T) {
		t.Setenv("GO_ENV", "development")
		req := newWSReq(t, "http://127.0.0.1:8080/ws", "127.0.0.1:8080", "http://127.0.0.1:5174")
		if !wsCheckOrigin(req) {
			t.Fatal("expected development to allow Vite proxy origin")
		}
	})

	t.Run("production same host", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		req := newWSReq(t, "https://app.example.com/ws", "app.example.com", "https://app.example.com")
		if !wsCheckOrigin(req) {
			t.Fatal("expected same-origin production handshake")
		}
	})

	t.Run("production same host different scheme", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		req := newWSReq(t, "https://app.example.com/ws", "app.example.com", "http://app.example.com")
		if !wsCheckOrigin(req) {
			t.Fatal("host match should not require scheme match")
		}
	})

	t.Run("production rejects other host", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		req := newWSReq(t, "https://app.example.com/ws", "app.example.com", "https://evil.example")
		if wsCheckOrigin(req) {
			t.Fatal("expected cross-site origin to be rejected")
		}
	})

	t.Run("production rejects host port mismatch", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		req := newWSReq(t, "http://localhost:8080/ws", "localhost:8080", "http://localhost:5174")
		if wsCheckOrigin(req) {
			t.Fatal("expected different port to be rejected outside development")
		}
	})

	t.Run("production allows empty origin", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		req := newWSReq(t, "https://app.example.com/ws", "app.example.com", "")
		if !wsCheckOrigin(req) {
			t.Fatal("empty Origin should be allowed for non-browser clients")
		}
	})

	t.Run("production rejects garbage origin", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		req := newWSReq(t, "https://app.example.com/ws", "app.example.com", "://not-a-url")
		if wsCheckOrigin(req) {
			t.Fatal("malformed Origin must be rejected")
		}
	})

	t.Run("host compare is case insensitive", func(t *testing.T) {
		t.Setenv("GO_ENV", "production")
		req := newWSReq(t, "https://App.Example.Com/ws", "App.Example.Com", "https://app.example.com")
		if !wsCheckOrigin(req) {
			t.Fatal("host comparison should be case-insensitive")
		}
	})
}

func TestWsMaxMessageBytes(t *testing.T) {
	t.Parallel()
	// Must cover max collab text plus framing headroom; still far below unbounded.
	if wsMaxMessageBytes <= 5<<20 {
		t.Fatalf("wsMaxMessageBytes = %d, want > 5 MiB", wsMaxMessageBytes)
	}
	if wsMaxMessageBytes > 16<<20 {
		t.Fatalf("wsMaxMessageBytes = %d looks too large", wsMaxMessageBytes)
	}
}
