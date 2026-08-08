package middleware

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// GlobalConfig global middleware konfigürasyonu
type GlobalConfig struct {
	AllowedOrigins []string
	TrustedProxies []string
	IsTLS          bool
}

// WithGlobalMiddleware global middleware'leri uygular
func WithGlobalMiddleware(handler http.Handler, log *logger.Logger) http.Handler {
	return WithGlobalMiddlewareConfig(handler, log, GlobalConfig{})
}

// WithGlobalMiddlewareConfig konfigürasyonlu global middleware
func WithGlobalMiddlewareConfig(handler http.Handler, log *logger.Logger, cfg GlobalConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers - allowlist tabanlı, wildcard sadece AllowedOrigins boşsa development'ta
		origin := r.Header.Get("Origin")
		if len(cfg.AllowedOrigins) > 0 {
			for _, ao := range cfg.AllowedOrigins {
				if ao == origin || ao == "*" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					break
				}
			}
		} else if origin != "" {
			// Development: Origin varsa ama allowlist boşsa — header koyma (güvenli default)
			// Health bile wildcard döndürmez; sadece explicit AllowedOrigins ile izin
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With, X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if len(cfg.AllowedOrigins) > 0 && origin != "" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'")
		if cfg.IsTLS {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")

		// OPTIONS preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Request logging
		log.Infow("request",
			"method", r.Method,
			"path", r.URL.Path,
			"ip", getClientIP(r),
		)

		handler.ServeHTTP(w, r)
	})
}

// ServeSPA Vue 3 SPA'yı serve eder (API rotaları hariç)
func ServeSPA(webFS fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(webFS))

	return func(w http.ResponseWriter, r *http.Request) {
		// API isteklerini atla
		if strings.HasPrefix(r.URL.Path, "/api/") {
			return
		}

		// Dosya var mı kontrol et
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := webFS.Open(path)
		if err != nil {
			// Dosya bulunamadıysa SPA index.html döndür
			r.URL.Path = "/"
		} else {
			f.Close()
		}

		fileServer.ServeHTTP(w, r)
	}
}
