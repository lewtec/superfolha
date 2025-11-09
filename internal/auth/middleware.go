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

// Middleware checks for JWT token in Authorization header
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]

				// Validate token
				claims, err := ValidateToken(token)
				if err == nil {
					// Add user context to request
					ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
						UserID: claims.UserID,
						Email:  claims.Email,
					})
					r = r.WithContext(ctx)
				}
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
