package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/store"
)

// AuditLogger audit log middleware'i — asenkron, buffered channel ile
type AuditLogger struct {
	store store.Store
	queue chan *model.AuditLog
}

// NewAuditLogger yeni bir audit logger oluşturur
func NewAuditLogger(s store.Store) *AuditLogger {
	al := &AuditLogger{
		store: s,
		queue: make(chan *model.AuditLog, 1000),
	}
	go al.worker()
	return al
}

func (al *AuditLogger) worker() {
	for log := range al.queue {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = al.store.CreateAuditLog(ctx, log)
		cancel()
	}
}

// Middleware tum API isteklerini audit log'a kaydeder
func (al *AuditLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sadece yazma islemlerini logla (POST, PUT, DELETE, PATCH)
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
			resource := r.URL.Path
			action := r.Method
			// Detay: query + redacted body info (password'leri maskele)
			details := r.URL.RawQuery
			if details == "" {
				details = "{}"
			}
			// Basit redaction: URL'de password varsa maskele
			details = redact(details)
			al.Log(r, action, resource, details)
		}
		next.ServeHTTP(w, r)
	})
}

// Log bir audit kaydi olusturur (async)
func (al *AuditLogger) Log(r *http.Request, action, resource, details string) {
	userID, _ := GetUserID(r.Context())
	clientIP := getClientIP(r)

	var uidPtr *int64
	if userID != 0 {
		uidPtr = &userID
	}

	log := &model.AuditLog{
		UserID:    uidPtr,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IP:        clientIP,
		CreatedAt: time.Now(),
	}

	// Async enqueue, full ise drop (request'i bloke etme)
	select {
	case al.queue <- log:
	default:
		// Queue dolu, sync fallback best-effort
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = al.store.CreateAuditLog(ctx, log)
		}()
	}
}

func redact(s string) string {
	// Basit: password, passwd, secret, token parametrelerini maskele
	lower := strings.ToLower(s)
	for _, kw := range []string{"password", "passwd", "secret", "token"} {
		if strings.Contains(lower, kw) {
			// Tüm detay maskele
			return `{"redacted":true}`
		}
	}
	return s
}
