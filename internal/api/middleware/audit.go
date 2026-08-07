package middleware

import (
	"net/http"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/store"
)

// AuditLogger audit log middleware'i
type AuditLogger struct {
	store store.Store
}

// NewAuditLogger yeni bir audit logger oluşturur
func NewAuditLogger(s store.Store) *AuditLogger {
	return &AuditLogger{store: s}
}

// Middleware tum API isteklerini audit log'a kaydeder
func (al *AuditLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sadece yazma islemlerini logla (POST, PUT, DELETE)
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
			resource := r.URL.Path
			action := r.Method
			details := r.URL.RawQuery
			if details == "" {
				details = "{}"
			}
			al.Log(r, action, resource, details)
		}
		next.ServeHTTP(w, r)
	})
}

// Log bir audit kaydi olusturur
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

	// Sync yaz
	_ = al.store.CreateAuditLog(r.Context(), log)
}
