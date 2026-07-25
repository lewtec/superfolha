package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

type UserContext struct {
	UserID string
	Email  string
}

// Middleware checks for JWT token in a cookie first, then in the Authorization header.
// It populates the context with UserContext if a valid token is found.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// 1. Try to get the token from a cookie
		cookie, err := r.Cookie(AuthCookieName)
		if err == nil {
			tokenString = cookie.Value
		}

		// 2. If no token from cookie, try Authorization header (case-sensitive "Bearer ").
		if tokenString == "" {
			const bearerPrefix = "Bearer "
			authHeader := r.Header.Get("Authorization")
			if after, ok := strings.CutPrefix(authHeader, bearerPrefix); ok {
				tokenString = strings.TrimSpace(after)
			}
		}

		// If a token string was found from either source, validate it
		if tokenString != "" {
			claims, err := ValidateToken(tokenString)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
					UserID: claims.UserID,
					Email:  claims.Email,
				})
				r = r.WithContext(ctx)
			} else if cookie != nil {
				// Invalid cookie token: clear with the same flags as Set/logout.
				ClearAuthCookie(w)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extracts user info from context
func GetUserFromContext(ctx context.Context) (*UserContext, bool) {
	user, ok := ctx.Value(UserContextKey).(*UserContext)
	return user, ok
}
