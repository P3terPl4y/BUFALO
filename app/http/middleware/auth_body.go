package middleware

import (
	"net/url"
	"strings"

	"github.com/valyala/fasthttp"
)

// Apply auth size limits as soon as headers arrive, before buffering a body.
// The normal 6 MiB limit still applies to uploads and other routes.
func AuthBodyConfig(header *fasthttp.RequestHeader) fasthttp.RequestConfig {
	if !header.IsPost() {
		return fasthttp.RequestConfig{}
	}
	uri, err := url.ParseRequestURI(string(header.RequestURI()))
	if err == nil && (CanonicalPath(uri.Path) == "/login" || CanonicalPath(uri.Path) == "/register" || CanonicalPath(uri.Path) == "/register/confirm") {
		return fasthttp.RequestConfig{MaxRequestBodySize: 16 << 10}
	}
	return fasthttp.RequestConfig{}
}

// CanonicalPath matches Fiber's default case-insensitive, non-strict routing.
func CanonicalPath(path string) string {
	path = strings.ToLower(strings.TrimRight(path, "/"))
	if path == "" {
		return "/"
	}
	return path
}
