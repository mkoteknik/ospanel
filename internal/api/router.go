package api

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/mkoteknik/ospanel/internal/adapter/ols"
	"github.com/mkoteknik/ospanel/internal/api/handler"
	apimw "github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// RouterConfig router konfigürasyonu
type RouterConfig struct {
	Store     store.Store
	Logger    *logger.Logger
	JWTSecret string
	WebFS     fs.FS
	OLS       *ols.Client
}

// NewRouter ana API router'ı oluşturur
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Chi middleware'leri
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Compress(5))

	// Rate limiter
	rateLimiter := apimw.NewRateLimiter(100, 200) // saniyede 100 istek, burst 200

	// Handler'ları oluştur
	authH := handler.NewAuthHandler(cfg.Store, cfg.JWTSecret, cfg.Logger)
	domainH := handler.NewDomainHandler(cfg.Store, cfg.Logger, cfg.OLS)
	databaseH := handler.NewDatabaseHandler(cfg.Store, cfg.Logger)
	fileH := handler.NewFileHandler(cfg.Store, cfg.Logger)
	monitorH := handler.NewMonitorHandler(cfg.Logger)
	adminH := handler.NewAdminHandler(cfg.Store, cfg.Logger)

	// Auth middleware
	authMW := apimw.AuthMiddleware(cfg.JWTSecret)

	// API v1 rotaları
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(rateLimiter.Limit)

		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.RefreshToken)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMW)

			// Auth
			r.Get("/auth/me", authH.Me)
			r.Post("/auth/logout", authH.Logout)
			r.Put("/auth/password", authH.ChangePassword)
			r.Put("/auth/2fa", authH.Setup2FA)

			// Domains
			r.Get("/domains", domainH.List)
			r.Post("/domains", domainH.Create)
			r.Get("/domains/{id}", domainH.Get)
			r.Put("/domains/{id}", domainH.Update)
			r.Delete("/domains/{id}", domainH.Delete)
			r.Post("/domains/{id}/ssl", domainH.InstallSSL)

			// Databases
			r.Get("/databases", databaseH.List)
			r.Post("/databases", databaseH.Create)
			r.Delete("/databases/{id}", databaseH.Delete)

			// Files
			r.Get("/files", fileH.List)
			r.Post("/files/upload", fileH.Upload)
			r.Post("/files/read", fileH.ReadFile)
			r.Post("/files/write", fileH.WriteFile)
			r.Delete("/files", fileH.DeleteFile)
			r.Post("/files/mkdir", fileH.CreateDir)
			r.Post("/files/archive", fileH.CreateArchive)
			r.Post("/files/extract", fileH.ExtractArchive)

			// Monitor
			r.Get("/monitor/stats", monitorH.Stats)
			r.Get("/monitor/ws", monitorH.LiveStats)

			// Admin only
			r.Group(func(r chi.Router) {
				r.Use(apimw.RequireAdmin())
				r.Get("/admin/users", adminH.ListUsers)
				r.Post("/admin/users", adminH.CreateUser)
				r.Put("/admin/users/{id}", adminH.UpdateUser)
				r.Delete("/admin/users/{id}", adminH.DeleteUser)
				r.Get("/admin/settings", adminH.GetSettings)
				r.Put("/admin/settings", adminH.UpdateSettings)
				r.Get("/admin/audit-logs", adminH.AuditLogs)
			})
		})
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"OpenSpeed Panel"}`))
	})

	// SPA fallback (son middleware olarak)
	if cfg.WebFS != nil {
		r.NotFound(apimw.ServeSPA(cfg.WebFS))
	}

	return r
}
