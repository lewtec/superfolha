//go:build !release

package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Without -tags release the SPA is reverse-proxied to Vite (127.0.0.1:5174).
// Prefer: mise start (build:frontend + go run -tags release, embedded UI).
// HMR: mise run start:vite  +  go run ./cmd/superfolha --state-dir=./data
func GetWebApp() http.Handler {
	remote, err := url.Parse("http://127.0.0.1:5174")
	if err != nil {
		// Constant URL; parse failure is a programming error.
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(remote)
}
