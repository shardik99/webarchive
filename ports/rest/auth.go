package rest

import (
	"context"
	"net/http"

	"github.com/derfenix/webarchive/config"
)

type contextKey string

const ownerContextKey contextKey = "owner"

// AuthMiddleware creates a middleware that checks basic auth or proxy header,
// and injects the owner into the request context.
func AuthMiddleware(cfg config.Auth, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		if !cfg.Enabled {
			// If not enabled, proceed with empty owner (anonymous)
			ctx := context.WithValue(r.Context(), ownerContextKey, "")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Check proxy header first
		if cfg.ProxyHeader != "" {
			if owner := r.Header.Get(cfg.ProxyHeader); owner != "" {
				ctx := context.WithValue(r.Context(), ownerContextKey, owner)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Check basic auth
		if cfg.BasicUsername != "" && cfg.BasicPassword != "" {
			user, pass, ok := r.BasicAuth()
			if ok && user == cfg.BasicUsername && pass == cfg.BasicPassword {
				ctx := context.WithValue(r.Context(), ownerContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// If we get here, authentication failed
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// OwnerFromContext retrieves the owner from the context. Returns empty string if not found.
func OwnerFromContext(ctx context.Context) string {
	if owner, ok := ctx.Value(ownerContextKey).(string); ok {
		return owner
	}
	return ""
}
