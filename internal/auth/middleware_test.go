package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddleware_AuthSources(t *testing.T) {
	withJWTEnv(t, "test-secret-for-middleware", "development")

	token, err := GenerateToken("user-42", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// Handler that records whether a user was injected into the request context.
	var gotUser *UserContext
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _ = GetUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(next)

	tests := []struct {
		name       string
		cookie     string
		authHeader string
		wantUserID string
		wantEmail  string
		wantUser   bool
		// When true, expect Set-Cookie clearing authToken on the response.
		wantClearCookie bool
	}{
		{
			name:       "valid bearer",
			authHeader: "Bearer " + token,
			wantUser:   true,
			wantUserID: "user-42",
			wantEmail:  "user@example.com",
		},
		{
			name:       "bearer with surrounding token whitespace",
			authHeader: "Bearer  " + token + "  ",
			wantUser:   true,
			wantUserID: "user-42",
			wantEmail:  "user@example.com",
		},
		{
			name:       "valid cookie preferred over bearer",
			cookie:     token,
			authHeader: "Bearer invalid-token",
			wantUser:   true,
			wantUserID: "user-42",
			wantEmail:  "user@example.com",
		},
		{
			name:       "valid cookie alone",
			cookie:     token,
			wantUser:   true,
			wantUserID: "user-42",
			wantEmail:  "user@example.com",
		},
		{
			name:            "invalid cookie cleared",
			cookie:          "not-a-valid-jwt",
			wantUser:        false,
			wantClearCookie: true,
		},
		{
			name:       "invalid bearer does not clear cookie path",
			authHeader: "Bearer not-a-valid-jwt",
			wantUser:   false,
		},
		{
			name:       "missing scheme rejected",
			authHeader: token,
			wantUser:   false,
		},
		{
			name:       "wrong scheme case rejected",
			authHeader: "bearer " + token,
			wantUser:   false,
		},
		{
			name:       "basic auth rejected",
			authHeader: "Basic dXNlcjpwYXNz",
			wantUser:   false,
		},
		{
			name:       "empty bearer value rejected",
			authHeader: "Bearer ",
			wantUser:   false,
		},
		{
			name:       "scheme only rejected",
			authHeader: "Bearer",
			wantUser:   false,
		},
		{
			name:     "no credentials",
			wantUser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser = nil
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: tt.cookie})
			}
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if tt.wantUser {
				if gotUser == nil {
					t.Fatal("expected user in context, got nil")
				}
				if gotUser.UserID != tt.wantUserID {
					t.Errorf("UserID = %q, want %q", gotUser.UserID, tt.wantUserID)
				}
				if gotUser.Email != tt.wantEmail {
					t.Errorf("Email = %q, want %q", gotUser.Email, tt.wantEmail)
				}
			} else if gotUser != nil {
				t.Errorf("expected no user, got %+v", gotUser)
			}

			setCookie := rec.Header().Get("Set-Cookie")
			cleared := strings.Contains(setCookie, AuthCookieName+"=") &&
				(strings.Contains(setCookie, "Max-Age=0") || strings.Contains(setCookie, "Max-Age=-1"))
			if tt.wantClearCookie && !cleared {
				t.Errorf("expected authToken clear cookie, Set-Cookie=%q", setCookie)
			}
			if !tt.wantClearCookie && cleared {
				t.Errorf("unexpected authToken clear cookie: %q", setCookie)
			}
		})
	}
}

func TestGetUserFromContext_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if user, ok := GetUserFromContext(req.Context()); ok || user != nil {
		t.Fatalf("expected empty context, got user=%v ok=%v", user, ok)
	}
}
