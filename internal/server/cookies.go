package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/lewtec/superfolha/internal/telemetry"
)

const (
	AuthCookieName     = "authToken"
	AuthCookieDuration = 7 * 24 * time.Hour
)

func SetAuthCookie(ctx context.Context, token string) error {
	w, ok := ctx.Value(ResponseWriterContextKey).(http.ResponseWriter)
	if !ok {
		err := errors.New("internal server error: response writer not available")
		telemetry.ReportError(ctx, err)
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Expires:  time.Now().Add(AuthCookieDuration),
		HttpOnly: true,
		Secure:   true, // Set to true in production for HTTPS
		Path:     "/",  // Make cookie available to all paths
	})

	return nil
}
