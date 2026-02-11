package server

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestSetAuthCookie(t *testing.T) {
	// 1. Test success
	w := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), ResponseWriterContextKey, w)
	token := "test-token"

	err := SetAuthCookie(ctx, token)
	if err != nil {
		t.Fatalf("SetAuthCookie failed: %v", err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != AuthCookieName {
		t.Errorf("Expected cookie name %s, got %s", AuthCookieName, cookie.Name)
	}
	if cookie.Value != token {
		t.Errorf("Expected cookie value %s, got %s", token, cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Errorf("Expected HttpOnly true")
	}
	if !cookie.Secure {
		t.Errorf("Expected Secure true")
	}
	if cookie.Path != "/" {
		t.Errorf("Expected Path /, got %s", cookie.Path)
	}

	// 2. Test failure (missing ResponseWriter)
	ctx = context.Background()
	err = SetAuthCookie(ctx, token)
	if err == nil {
		t.Fatal("Expected error when ResponseWriter is missing, got nil")
	}
}
