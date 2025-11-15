//go:build !release

package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func GetWebApp() http.Handler {
	remote, _ := url.Parse("http://localhost:5174")
	return httputil.NewSingleHostReverseProxy(remote)
}
