package middleware

import (
	"net/http"

	"github.com/openspeed-panel/ospanel/internal/model"
)

// RequireRole belirli roller için yetki kontrolü middleware'i
func RequireRole(roles ...model.UserRole) func(http.Handler) http.Handler {
	allowedRoles := make(map[model.UserRole]bool, len(roles))
	for _, r := range roles {
		allowedRoles[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetUserRole(r.Context())
			if !ok {
				http.Error(w, `{"error":"Yetkilendirme gerekli"}`, http.StatusUnauthorized)
				return
			}

			if !allowedRoles[model.UserRole(role)] {
				http.Error(w, `{"error":"Bu işlem için yetkiniz yok"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin sadece admin rolü için
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole(model.RoleAdmin)
}

// RequireAdminOrReseller admin veya reseller rolü için
func RequireAdminOrReseller() func(http.Handler) http.Handler {
	return RequireRole(model.RoleAdmin, model.RoleReseller)
}
