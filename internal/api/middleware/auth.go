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
			tokenString := extractToken(r)
			if tokenString == "" {
				http.Error(w, `{"error":"Yetkilendirme gerekli"}`, http.StatusUnauthorized)
				return
			}

			claims, err := parseAndValidateToken(tokenString, jwtSecret, "access")
			if err != nil {
				http.Error(w, `{"error":"Geçersiz veya süresi dolmuş token"}`, http.StatusUnauthorized)
				return
			}

			ctx := injectClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthWS WebSocket için token'ı query (?token=) veya header'dan alır
func AuthWS(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			tokenString = r.URL.Query().Get("token")
		}
		if tokenString == "" {
			if c, err := r.Cookie("access_token"); err == nil {
				tokenString = c.Value
			}
		}
		if tokenString == "" {
			http.Error(w, `{"error":"Yetkilendirme gerekli"}`, http.StatusUnauthorized)
			return
		}
		claims, err := parseAndValidateToken(tokenString, jwtSecret, "access")
		if err != nil {
			http.Error(w, `{"error":"Geçersiz veya süresi dolmuş token"}`, http.StatusUnauthorized)
			return
		}
		ctx := injectClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

func parseAndValidateToken(tokenString, jwtSecret, expectedType string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("ospanel"), jwt.WithAudience("ospanel"))
	if err != nil || !token.Valid {
		if err != nil {
			return nil, err
		}
		return nil, jwt.ErrTokenExpired
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	if expectedType != "" {
		if typ, ok := claims["type"].(string); !ok || typ != expectedType {
			return nil, jwt.ErrTokenInvalidClaims
		}
	}
	return claims, nil
}

func injectClaims(ctx context.Context, claims jwt.MapClaims) context.Context {
	if userID, ok := claims["user_id"]; ok {
		switch v := userID.(type) {
		case float64:
			ctx = context.WithValue(ctx, UserIDKey, int64(v))
		case int64:
			ctx = context.WithValue(ctx, UserIDKey, v)
		case int:
			ctx = context.WithValue(ctx, UserIDKey, int64(v))
		}
	}
	if role, ok := claims["role"]; ok {
		if s, ok := role.(string); ok {
			ctx = context.WithValue(ctx, UserRoleKey, s)
		}
	}
	if username, ok := claims["username"]; ok {
		if s, ok := username.(string); ok {
			ctx = context.WithValue(ctx, UsernameKey, s)
		}
	}
	return ctx
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
