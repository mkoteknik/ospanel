package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	// UserIDKey context içinde kullanıcı ID'si için key
	UserIDKey contextKey = "user_id"
	// UserRoleKey context içinde kullanıcı rolü için key
	UserRoleKey contextKey = "user_role"
	// UsernameKey context içinde kullanıcı adı için key
	UsernameKey contextKey = "username"
)

// AuthMiddleware JWT tabanlı kimlik doğrulama middleware'i
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Authorization header'ı al
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Yetkilendirme gerekli"}`, http.StatusUnauthorized)
				return
			}

			// Bearer token formatı
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, `{"error":"Geçersiz yetkilendirme formatı"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Token'ı doğrula
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"Geçersiz veya süresi dolmuş token"}`, http.StatusUnauthorized)
				return
			}

			// Claims'leri al
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"Geçersiz token claims"}`, http.StatusUnauthorized)
				return
			}

			// Context'e kullanıcı bilgilerini ekle
			ctx := r.Context()
			if userID, ok := claims["user_id"]; ok {
				ctx = context.WithValue(ctx, UserIDKey, int64(userID.(float64)))
			}
			if role, ok := claims["role"]; ok {
				ctx = context.WithValue(ctx, UserRoleKey, role.(string))
			}
			if username, ok := claims["username"]; ok {
				ctx = context.WithValue(ctx, UsernameKey, username.(string))
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID context'ten kullanıcı ID'sini alır
func GetUserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(UserIDKey).(int64)
	return id, ok
}

// GetUserRole context'ten kullanıcı rolünü alır
func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok
}

// GetUsername context'ten kullanıcı adını alır
func GetUsername(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(UsernameKey).(string)
	return name, ok
}
