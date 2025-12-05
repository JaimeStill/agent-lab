package middleware

import (
	"net/http"
	"strings"
)

// TrimSlash returns middleware that redirects requests with trailing slashes
// to their canonical form without the slash. The root path "/" is preserved.
func TrimSlash() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.URL.Path) > 1 && strings.HasSuffix(r.URL.Path, "/") {
				target := strings.TrimSuffix(r.URL.Path, "/")
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
