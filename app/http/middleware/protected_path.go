package middleware

import "strings"

// isProtectedRequestPath keeps the catch-all session middleware from turning
// unrelated/unknown URLs into login redirects. Individual role middleware is
// still applied by the registered protected routes.
func isProtectedRequestPath(path string) bool {
	path = strings.TrimSuffix(path, "/")
	for _, prefix := range []string{
		"/home", "/logout", "/presence", "/profile", "/red-choferes",
		"/loads", "/direcciones", "/empresas", "/choferes", "/publicadores",
		"/facturas", "/admin",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
