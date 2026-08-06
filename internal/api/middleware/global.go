package middleware

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/openspeed-panel/ospanel/internal/pkg/logger"
)

// WithGlobalMiddleware global middleware'leri uygular
func WithGlobalMiddleware(handler http.Handler, log *logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

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
