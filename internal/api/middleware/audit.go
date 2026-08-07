package middleware

import (
	"context"
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

// Log bir audit kaydi olusturur (async, goroutine ile)
func (al *AuditLogger) Log(r *http.Request, action, resource, details string) {
	go func() {
		userID, _ := GetUserID(r.Context())
		var uid *int64
		if userID != 0 {
			uid = &userID
		}

		log := &model.AuditLog{
			UserID:    uid,
			Action:    action,
			Resource:  resource,
			Details:   details,
			IP:        getClientIP(r),
			CreatedAt: time.Now(),
		}

		// request context'i yerine background context kullan (goroutine icin)
		_ = al.store.CreateAuditLog(context.Background(), log)
	}()
}
