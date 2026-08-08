package api

import (
	"fmt"
	"io/fs"
	"net/http"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"

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
	loginLimiter := apimw.NewLoginLimiter()

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
			r.With(loginLimiter.Limit).Post("/login", authH.Login)
			r.With(loginLimiter.Limit).Post("/refresh", authH.RefreshToken)
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
		r.Post("/domains/{id}/ssl/custom", domainH.UploadCustomSSL)
			r.Get("/domains/{id}/subdomains", domainH.ListSubdomains)
			r.Post("/domains/{id}/subdomains", domainH.CreateSubdomain)
			r.Get("/domains/{id}/aliases", domainH.ListDomainAliases)
			r.Post("/domains/{id}/aliases", domainH.CreateDomainAlias)
			r.Delete("/domains/{id}/aliases/{aliasId}", domainH.DeleteDomainAlias)
			r.Post("/domains/{id}/secure", domainH.SecureSite)
		r.Post("/domains/{id}/install-cms", domainH.InstallCMS)

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

			// OLS Status (bilgi amaçlı, admin-only değil)
			r.Get("/ols/status", olsH.Status)
		r.Get("/ols/php-extensions", domainH.GetPHPExtensions)

			// Monitor (bilgi amaçlı)
			r.Get("/monitor/stats", monitorH.Stats)
			r.Get("/monitor/ws", monitorH.LiveStats)

			// Cache Status (bilgi amaçlı)
			r.Get("/cache/status", cacheH.Status)
			r.Get("/cache/info", cacheH.Info)

			// Hostname (GET - bilgi amaçlı)
			r.Get("/server/hostname", adminH.GetHostname)

			// === ADMIN + RESELLER ROUTES ===
			r.Group(func(r chi.Router) {
				r.Use(apimw.RequireAdminOrReseller())

				// Kullanici yonetimi (reseller kendi kullanicilarini yonetebilir)
				r.Get("/admin/users", adminH.ListUsers)
				r.Post("/admin/users", adminH.CreateUser)
				r.Put("/admin/users/{id}", adminH.UpdateUser)
				r.Delete("/admin/users/{id}", adminH.DeleteUser)
			r.Post("/admin/users/{id}/jail", adminH.ToggleJail)
			r.Get("/admin/packages", adminH.ListPackages)
			r.Post("/admin/users/{id}/package", adminH.AssignPackage)

				// Audit log (reseller sadece kendi loglarini gorebilir)
				r.Get("/admin/audit-logs", adminH.AuditLogs)
			})

			// === ADMIN ONLY ROUTES ===
			r.Group(func(r chi.Router) {
				r.Use(apimw.RequireAdmin())

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

				// Cache (Redis) - flush
				r.Post("/cache/flush", cacheH.FlushCache)

				// OLS WebAdmin (şifre görüntüleme/değiştirme/proxy)
				r.Get("/ols/info", olsH.LoginInfo)
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
			r.Post("/ssl/dns-challenge", sslH.StartManualDNSChallenge)
			r.Post("/ssl/dns-challenge/verify", sslH.CompleteDNSChallenge)
			r.Get("/ssl/dns-challenge/{id}", sslH.GetDNSChallengeStatus)

				// Hostname (set)
				r.Put("/server/hostname", adminH.SetHostname)

				// Settings (sadece admin)
				r.Get("/admin/settings", adminH.GetSettings)
				r.Put("/admin/settings", adminH.UpdateSettings)
			r.Post("/admin/packages", adminH.CreatePackage)
			r.Put("/admin/packages/{id}", adminH.UpdatePackage)
			r.Delete("/admin/packages/{id}", adminH.DeletePackage)
			})
		})
	})

	// Health check — DB, disk, OLS
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := "ok"
		code := http.StatusOK

		// Disk check: <90% dolu olmalı
		var stat syscall.Statfs_t
		if err := syscall.Statfs("/", &stat); err == nil {
			total := stat.Blocks * uint64(stat.Bsize)
			avail := stat.Bavail * uint64(stat.Bsize)
			usedPct := 0.0
			if total > 0 {
				usedPct = float64(total-avail) / float64(total) * 100
			}
			if usedPct > 90 {
				status = "degraded"
				code = http.StatusServiceUnavailable
			}
		}

		// DB check (store nil değil ve ListUsers deneyebilsin)
		if cfg.Store != nil {
			// 2s timeout içinde basit query
			done := make(chan error, 1)
			go func() {
				_, err := cfg.Store.ListUsers(r.Context())
				done <- err
			}()
			select {
			case err := <-done:
				if err != nil {
					status = "degraded"
					code = http.StatusServiceUnavailable
				}
			case <-time.After(2 * time.Second):
				status = "degraded"
				code = http.StatusServiceUnavailable
			}
		}

		w.WriteHeader(code)
		w.Write([]byte(fmt.Sprintf(`{"status":"%s","service":"OpenSpeed Panel","go":"%s","time":"%s"}`, status, runtime.Version(), time.Now().UTC().Format(time.RFC3339))))
	})

	// Metrics — admin JWT auth required (spoofable header kontrolü YOK)
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Sadece geçerli admin JWT token ile erişilebilir
		tokenString := ""
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tokenString = auth[7:]
		}
		if tokenString == "" {
			http.Error(w, `{"error":"Yetkilendirme gerekli"}`, http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("beklenmeyen imza yöntemi")
			}
			return []byte(cfg.JWTSecret), nil
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("ospanel"))
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"Geçersiz token"}`, http.StatusForbidden)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":"Geçersiz claims"}`, http.StatusForbidden)
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" {
			http.Error(w, `{"error":"Admin yetkisi gerekli"}`, http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Fprintf(w, "# HELP go_goroutines Number of goroutines\n")
		fmt.Fprintf(w, "# TYPE go_goroutines gauge\n")
		fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())
		fmt.Fprintf(w, "# HELP go_mem_alloc_bytes Allocated memory\n")
		fmt.Fprintf(w, "go_mem_alloc_bytes %d\n", m.Alloc)
		fmt.Fprintf(w, "# HELP go_mem_sys_bytes System memory\n")
		fmt.Fprintf(w, "go_mem_sys_bytes %d\n", m.Sys)
	})

	// SPA fallback (son middleware olarak)
	if cfg.WebFS != nil {
		r.NotFound(apimw.ServeSPA(cfg.WebFS))
	}

	return r
}
