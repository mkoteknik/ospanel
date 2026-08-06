package middleware

import (
	"net/http"
	"time"

	"github.com/openspeed-panel/ospanel/internal/model"
	"github.com/openspeed-panel/ospanel/internal/store"
)

// AuditLogger audit log middleware'i
type AuditLogger struct {
	store store.Store
}

// NewAuditLogger yeni bir audit logger oluşturur
func NewAuditLogger(s store.Store) *AuditLogger {
	return &AuditLogger{store: s}
}

// Log bir audit kaydı oluşturur
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

		// Context'i request'ten bağımsız olarak oluştur
		_ = al.store.CreateAuditLog(r.Context(), log)
	}()
}
