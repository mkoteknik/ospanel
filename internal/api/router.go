package api

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/mkoteknik/ospanel/internal/adapter/cache"
	"github.com/mkoteknik/ospanel/internal/adapter/container"
	"github.com/mkoteknik/ospanel/internal/adapter/dns"
	"github.com/mkoteknik/ospanel/internal/adapter/email"
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
	pdnsClient := dns.NewClient()
	mailServer := email.NewMailServer()
	domainH := handler.NewDomainHandler(cfg.Store, cfg.Logger, cfg.OLS, pdnsClient, mailServer, "127.0.0.1")
	databaseH := handler.NewDatabaseHandler(cfg.Store, cfg.Logger)
	fileH := handler.NewFileHandler(cfg.Store, cfg.Logger)
	monitorH := handler.NewMonitorHandler(cfg.Logger)
	adminH := handler.NewAdminHandler(cfg.Store, cfg.Logger)
	cacheH := handler.NewCacheHandler(cache.NewRedisClient(), cfg.Logger)
	containerH := handler.NewContainerHandler(container.NewDockerClient(), cfg.Logger)
	deployH := handler.NewDeployHandler(cfg.Logger)
	cfH := handler.NewCFHandler(cfg.Logger)
	olsH := handler.NewOLSHandler("http://localhost:7080")
	cronH := handler.NewCronHandler(cfg.Logger)
	totpH := handler.NewTOTPHandler(cfg.Store, cfg.Logger)
	termH := handler.NewTerminalHandler(cfg.Logger)
	svcH := handler.NewServicesHandler(cfg.Logger)
	backupH := handler.NewBackupHandler(cfg.Store, cfg.Logger)
	dnsH := handler.NewDNSHandler(cfg.Store, cfg.Logger, pdnsClient)
	sslH := handler.NewSSLHandler(cfg.Store, cfg.Logger)

	// Audit logger
	auditLogger := apimw.NewAuditLogger(cfg.Store)
	_ = auditLogger

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
			r.Use(auditLogger.Middleware)

			// Auth
			r.Get("/auth/me", authH.Me)
			r.Post("/auth/logout", authH.Logout)
			r.Put("/auth/password", authH.ChangePassword)
			r.Put("/auth/2fa", authH.Setup2FA)

			// 2FA (TOTP)
			r.Get("/2fa/status", totpH.Status)
			r.Post("/2fa/setup", totpH.Setup)
			r.Post("/2fa/verify", totpH.Verify)
			r.Delete("/2fa/disable", totpH.Disable)

			// Email
			r.Get("/emails", domainH.ListEmails)
			r.Post("/emails", domainH.CreateEmailAccount)
			r.Delete("/emails/{id}", domainH.DeleteEmailAccount)
			r.Get("/emails/aliases", domainH.ListAliases)
			r.Post("/emails/aliases", domainH.CreateAlias)
			r.Delete("/emails/aliases/{id}", domainH.DeleteAlias)
			r.Put("/emails/{id}", domainH.UpdateEmail)

			// Domains
			r.Get("/domains", domainH.List)
			r.Post("/domains", domainH.Create)
			r.Get("/domains/{id}", domainH.Get)
			r.Put("/domains/{id}", domainH.Update)
			r.Delete("/domains/{id}", domainH.Delete)
			r.Post("/domains/{id}/ssl", domainH.InstallSSL)
			r.Get("/domains/{id}/subdomains", domainH.ListSubdomains)
			r.Post("/domains/{id}/subdomains", domainH.CreateSubdomain)

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
			r.Get("/files/download", fileH.Download)
			r.Post("/files/chmod", fileH.Chmod)
			r.Post("/files/rename", fileH.Rename)
			r.Post("/files/create", fileH.CreateFile)
			r.Post("/files/archive", fileH.CreateArchive)
			r.Post("/files/extract", fileH.ExtractArchive)

			// CloudFlare
			r.Get("/cf/status", cfH.Status)
			r.Get("/cf/zones", cfH.ListZones)
			r.Post("/cf/configure", cfH.Configure)
			r.Get("/cf/dns", cfH.ListDNS)
			r.Post("/cf/dns", cfH.CreateDNS)
			r.Delete("/cf/dns", cfH.DeleteDNS)
			r.Post("/cf/purge", cfH.PurgeCache)
			r.Get("/cf/analytics", cfH.Analytics)
			r.Post("/cf/ssl", cfH.SSLMode)

			// Cache (Redis)
			r.Get("/cache/status", cacheH.Status)
			r.Get("/cache/info", cacheH.Info)
			r.Post("/cache/flush", cacheH.FlushCache)

			// OLS WebAdmin
			r.Get("/ols/info", olsH.LoginInfo)
			r.Get("/ols/status", olsH.Status)
			r.Put("/ols/password", olsH.ChangePassword)
			r.Get("/ols/proxy", olsH.Proxy)
			r.Handle("/ols/*", http.StripPrefix("/ols", http.HandlerFunc(olsH.Proxy)))

			// Cron Jobs
			r.Get("/cron", cronH.List)
			r.Post("/cron", cronH.Add)
			r.Delete("/cron", cronH.Delete)

			// Deploy (One-click)
			r.Get("/deploy/templates", deployH.ListTemplates)
			r.Get("/deploy/template", deployH.GetTemplate)
			r.Post("/deploy", deployH.Deploy)

			// Containers (Docker/Podman)
			r.Get("/containers", containerH.List)
			r.Get("/containers/stats", containerH.Stats)
			r.Post("/containers/{id}/start", containerH.Start)
			r.Post("/containers/{id}/stop", containerH.Stop)
			r.Post("/containers/{id}/restart", containerH.Restart)

			// Terminal + Logs
			r.Get("/terminal/ws", termH.Connect)
			r.Get("/logs", termH.LogList)
			r.Get("/logs/view", termH.LogStream)

			// Monitor
			r.Get("/monitor/stats", monitorH.Stats)
			r.Get("/monitor/ws", monitorH.LiveStats)

			// System Services
			r.Get("/services", svcH.List)
			r.Post("/services/action", svcH.Action)

			// Backup
			r.Get("/backups", backupH.List)
			r.Post("/backups", backupH.Create)
			r.Put("/backups/{id}", backupH.Update)
			r.Delete("/backups/{id}", backupH.Delete)
			r.Post("/backups/{id}/run", backupH.Run)
			r.Get("/backups/results", backupH.ListBackups)

			// DNS Records
			r.Get("/dns", dnsH.List)
			r.Post("/dns", dnsH.Create)
			r.Put("/dns/{id}", dnsH.Update)
			r.Delete("/dns/{id}", dnsH.Delete)

			// SSL Certificates
			r.Get("/ssl", sslH.List)
			r.Get("/ssl/count", sslH.Get)
			r.Get("/ssl/{id}", sslH.Get)
			r.Post("/ssl/{id}/renew", sslH.Renew)
			r.Delete("/ssl/{id}", sslH.Delete)
			r.Post("/ssl/auto-renew", sslH.SetupAutoRenew)
			r.Post("/ssl/wildcard", sslH.IssueWildcard)

			// Hostname (public - bilgi amaçlı)
			r.Get("/server/hostname", adminH.GetHostname)
			r.Put("/server/hostname", adminH.SetHostname)

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
