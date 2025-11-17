package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser" // Changed import path
)

type contextKey string

const UserContextKey contextKey = "user"

type UserContext struct {
	UserID string
	Email  string
}

// GraphQLRequest represents a basic GraphQL request structure
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// Middleware checks for JWT token in a cookie first, then in the Authorization header.
// It skips authentication for Login and Register GraphQL mutations.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// --- NEW: Check for public GraphQL mutations ---
		// Read the request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Auth Middleware: Failed to read request body: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// Restore the body for subsequent handlers
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var gqlReq GraphQLRequest
		if err := json.Unmarshal(bodyBytes, &gqlReq); err == nil {
			// Parse the GraphQL query to find the operation name
			// schema := &ast.Schema{} // Removed unused schema declaration
			query, parseErr := parser.ParseQuery(&ast.Source{Input: gqlReq.Query}) // Convert string to *ast.Source
			if parseErr == nil && query != nil && len(query.Operations) > 0 {
				op := query.Operations[0] // Assuming a single operation per request
				if op.Operation == ast.Mutation {
					// Check if it's a Login or Register mutation
					for _, sel := range op.SelectionSet {
						if field, ok := sel.(*ast.Field); ok {
							if field.Name == "login" || field.Name == "register" {
								log.Printf("Auth Middleware: Skipping authentication for public mutation: %s", field.Name)
								next.ServeHTTP(w, r) // Skip authentication
								return
							}
						}
					}
				}
			}
		}
		// --- END NEW ---

		var tokenString string
		var authAttempted bool

		// 1. Try to get the token from a cookie
		cookie, err := r.Cookie("authToken")
		if err == nil {
			tokenString = cookie.Value
			authAttempted = true
			log.Printf("Auth Middleware: Extracted token from cookie.")
		} else {
			log.Printf("Auth Middleware: No 'authToken' cookie found: %v", err)
		}

		// 2. If no token from cookie, try Authorization header
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				authAttempted = true
				log.Printf("Auth Middleware: Received Authorization header.")
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
					log.Printf("Auth Middleware: Extracted token from Authorization header.")
				} else {
					log.Printf("Auth Middleware: Authorization header malformed: %s", authHeader)
				}
			} else {
				log.Println("Auth Middleware: No Authorization header found.")
			}
		}

		// If a token string was found from either source, validate it
		if tokenString != "" {
			claims, err := ValidateToken(tokenString)
			if err == nil {
				log.Printf("Auth Middleware: Token validated successfully for UserID: %s, Email: %s", claims.UserID, claims.Email)
				// Add user context to request
				ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
					UserID: claims.UserID,
					Email:  claims.Email,
				})
				r = r.WithContext(ctx)
			} else {
				log.Printf("Auth Middleware: Token validation failed: %v", err)
				// If token validation fails and it came from a cookie, clear the cookie
				if cookie != nil {
					http.SetCookie(w, &http.Cookie{
						Name:    "authToken",
						Value:   "",
						Expires: time.Unix(0, 0),
						HttpOnly: true,
						Secure:   true,
						Path:     "/",
					})
					log.Println("Auth Middleware: Cleared expired/invalid 'authToken' cookie.")
				}
			}
		} else if authAttempted {
			log.Println("Auth Middleware: No valid token found after checking cookie and Authorization header.")
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extracts user info from context
func GetUserFromContext(ctx context.Context) (*UserContext, bool) {
	user, ok := ctx.Value(UserContextKey).(*UserContext)
	return user, ok
}
