package middleware

import (
	"net/http"

	"portfolio-site/internal/module/stats"
)

// StatsTracking returns a middleware that records page views.
func StatsTracking(svc *stats.Service) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Record view after the request is handled.
			next.ServeHTTP(w, r)

			// Don't record admin paths or static files.
			path := r.URL.Path
			if isAdminPath(path) || isStaticPath(path) {
				return
			}

			// Record asynchronously so it doesn't block the response.
			go svc.RecordView(r.Context(), r, path)
		})
	}
}

func isAdminPath(path string) bool {
	return len(path) >= 6 && path[:6] == "/admin"
}

func isStaticPath(path string) bool {
	return len(path) >= 8 && path[:8] == "/static/"
}
