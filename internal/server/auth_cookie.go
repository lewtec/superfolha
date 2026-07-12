package server

import (
	"net/http"
	"time"

	"github.com/lewtec/superfolha/internal/appenv"
	"github.com/lewtec/superfolha/internal/auth"
)

func setAuthCookie(w http.ResponseWriter, token string) {
	// Secure only when not in development so local HTTP can set the cookie.
	secure := !appenv.IsDevelopment()
	http.SetCookie(w, &http.Cookie{
		Name:     "authToken",
		Value:    token,
		Expires:  time.Now().Add(auth.TokenTTL),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
}
