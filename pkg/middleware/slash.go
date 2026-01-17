package middleware

import (
	"net/http"
	"strings"
)

// AddSlash returns middleware that redirects requests without trailing slashes
// to their canonical form with a slash, unless the path has a file extension.
func AddSlash() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/") && !hasFileExtension(r.URL.Path) {
				target := r.URL.Path + "/"
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

func hasFileExtension(path string) bool {
	lastSlash := strings.LastIndex(path, "/")
	lastDot := strings.LastIndex(path, ".")
	return lastDot > lastSlash
}
