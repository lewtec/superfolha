package auth

import (
	"net/http"
	"time"

	"github.com/lewtec/superfolha/internal/appenv"
)

// AuthCookieName is the HttpOnly session cookie holding the JWT.
const AuthCookieName = "authToken"

// authCookieBase returns cookie fields shared by set and clear (flags must match
// or browsers keep stale cookies in local HTTP dev / mixed Secure contexts).
func authCookieBase() http.Cookie {
	return http.Cookie{
		Name:     AuthCookieName,
		Path:     "/",
		HttpOnly: true,
		// Secure only outside development so local HTTP can set and clear the cookie.
		Secure:   !appenv.IsDevelopment(),
		SameSite: http.SameSiteLaxMode,
	}
}

// SetAuthCookie writes the session JWT as an HttpOnly cookie with TokenTTL.
func SetAuthCookie(w http.ResponseWriter, token string) {
	c := authCookieBase()
	c.Value = token
	c.Expires = time.Now().Add(TokenTTL)
	http.SetCookie(w, &c)
}

// ClearAuthCookie expires the session cookie (logout or invalid token).
func ClearAuthCookie(w http.ResponseWriter) {
	c := authCookieBase()
	c.Value = ""
	c.MaxAge = -1
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, &c)
}
