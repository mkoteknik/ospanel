package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// AuthHandler kimlik doğrulama işlemleri
type AuthHandler struct {
	store     store.Store
	jwtSecret string
	log       *logger.Logger
}

// NewAuthHandler yeni AuthHandler oluşturur
func NewAuthHandler(s store.Store, jwtSecret string, log *logger.Logger) *AuthHandler {
	return &AuthHandler{store: s, jwtSecret: jwtSecret, log: log}
}

// Login kullanıcı girişi
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Kullanıcı adı ve şifre gerekli"})
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		h.log.Errorw("login başarısız - kullanıcı bulunamadı", "username", req.Username)
		// Timing attack mitigasyonu: dummy verify
		_ = verifyPassword(req.Password, "ospanel$v1$00000000000000000000000000000000$0000000000000000000000000000000000000000000000000000000000000000")
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz kullanıcı adı veya şifre"})
		return
	}

	if user.Status != model.StatusActive {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Hesap aktif değil"})
		return
	}

	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Hesap geçici olarak kilitlendi, lütfen daha sonra deneyin"})
		return
	}
	// Kilit süresi dolduysa sıfırla
	if user.LockedUntil != nil && time.Now().After(*user.LockedUntil) {
		user.LoginAttempts = 0
		user.LockedUntil = nil
		_ = h.store.UpdateUser(r.Context(), user)
	}

	if !verifyPassword(req.Password, user.PasswordHash) {
		attempts := user.LoginAttempts + 1
		_ = h.store.UpdateLoginAttempts(r.Context(), user.ID, attempts)
		// Eşik aşıldıysa kilitle
		if attempts >= 5 {
			until := time.Now().Add(30 * time.Minute)
			user.LockedUntil = &until
			user.Status = model.StatusLocked
			_ = h.store.UpdateUser(r.Context(), user)
			h.log.Warnw("hesap kilitlendi - brute force", "username", req.Username, "attempt", attempts, "until", until)
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Çok fazla başarısız deneme, hesap 30 dakika kilitlendi"})
			return
		}
		h.log.Warnw("login başarısız - şifre hatalı", "username", req.Username, "attempt", attempts)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz kullanıcı adı veya şifre"})
		return
	}

	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "2FA kodu gerekli", "require_2fa": "true"})
			return
		}
		if !CheckTotpCode(user.TOTPSecret, req.TOTPCode) {
			h.log.Warnw("login başarısız - geçersiz 2FA kodu", "username", req.Username)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz 2FA kodu"})
			return
		}
	}

	accessToken, refreshToken, expiresIn, err := generateTokens(user, h.jwtSecret)
	if err != nil {
		h.log.Errorw("token oluşturma hatası", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Token oluşturulamadı"})
		return
	}

	now := time.Now()
	_ = h.store.UpdateLoginAttempts(r.Context(), user.ID, 0)
	user.LastLoginAt = &now
	user.LastLoginIP = getClientIP(r)
	// Başarılı girişte kilidi aç
	user.LockedUntil = nil
	if user.Status == model.StatusLocked {
		user.Status = model.StatusActive
	}
	_ = h.store.UpdateUser(r.Context(), user)

	h.log.Infow("login başarılı", "username", req.Username, "ip", getClientIP(r))

	// httpOnly cookie de set et (FE memory + cookie dual)
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   expiresIn,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 3600,
	})

	writeJSON(w, http.StatusOK, model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         sanitizeUser(user),
	})
}

// Logout çıkış işlemi
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Cookie temizle
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	// TODO: refresh token revoke (jti blacklist) - store'a eklenince aktif olacak
	writeJSON(w, http.StatusOK, map[string]string{"message": "Başarıyla çıkış yapıldı"})
}

// Me mevcut kullanıcı bilgilerini döndürür
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Yetkilendirme gerekli"})
		return
	}

	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	writeJSON(w, http.StatusOK, sanitizeUser(user))
}

// RefreshToken token yenileme
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if req.RefreshToken == "" {
		// Cookie fallback
		if c, err := r.Cookie("refresh_token"); err == nil {
			req.RefreshToken = c.Value
		}
	}
	if req.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Refresh token gerekli"})
		return
	}

	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("ospanel"), jwt.WithAudience("ospanel"))
	if err != nil || !token.Valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz refresh token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz token"})
		return
	}
	if typ, ok := claims["type"].(string); !ok || typ != "refresh" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz token tipi"})
		return
	}

	uidFloat, ok := claims["user_id"].(float64)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz token"})
		return
	}
	userID := int64(uidFloat)
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}
	if user.Status != model.StatusActive {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Hesap aktif değil"})
		return
	}

	accessToken, refreshToken, expiresIn, _ := generateTokens(user, h.jwtSecret)

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   expiresIn,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth/refresh",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 3600,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    expiresIn,
	})
}

// ChangePassword şifre değiştirme
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	var req model.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if err := validatePasswordStrength(req.NewPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	if !verifyPassword(req.CurrentPassword, user.PasswordHash) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Mevcut şifre yanlış"})
		return
	}

	hashed, _ := hashPassword(req.NewPassword)
	user.PasswordHash = hashed
	_ = h.store.UpdateUser(r.Context(), user)

	// Şifre değişiminde tüm oturumları geçersiz kılmak için cookie temizle
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})

	writeJSON(w, http.StatusOK, map[string]string{"message": "Şifre başarıyla değiştirildi, lütfen tekrar giriş yapın"})
}

// Setup2FA 2FA kurulumu (deprecated, TOTPHandler kullanın)
func (h *AuthHandler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, map[string]string{"error": "Bu endpoint kaldırıldı, /api/v1/2fa/* kullanın"})
}

func validatePasswordStrength(pw string) error {
	if len(pw) < 12 {
		return &passErr{"Şifre en az 12 karakter olmalı"}
	}
	if len(pw) > 128 {
		return &passErr{"Şifre en fazla 128 karakter olabilir"}
	}
	var hasUpper, hasLower, hasDigit bool
	for _, c := range pw {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return &passErr{"Şifre en az bir büyük harf, bir küçük harf ve bir rakam içermeli"}
	}
	return nil
}

type passErr struct{ msg string }

func (e *passErr) Error() string { return e.msg }

// hashPassword Argon2id ile şifre hashler, format: ospanel$v1$<hex salt>$<hex hash>
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	// OWASP önerisi: time=3, memory=64MB, threads=4
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return "ospanel$v1$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(hash), nil
}

// verifyPassword hashlenmiş şifreyi doğrular — v1 (time=3) ve legacy time=1 uyumlu
func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "ospanel" || parts[1] != "v1" {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expectedHash, err := hex.DecodeString(parts[3])
	if err != nil {
		return false
	}
	for _, iter := range []uint32{3, 1} {
		computed := argon2.IDKey([]byte(password), salt, iter, 64*1024, 4, 32)
		if subtle.ConstantTimeCompare(computed, expectedHash) == 1 {
			return true
		}
	}
	return false
}

// generateTokens JWT token çifti oluşturur
func generateTokens(user *model.User, secret string) (string, string, int, error) {
	accessExpiry := 15 * time.Minute
	refreshExpiry := 7 * 24 * time.Hour
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     string(user.Role),
		"exp":      now.Add(accessExpiry).Unix(),
		"iat":      now.Unix(),
		"iss":      "ospanel",
		"aud":      "ospanel",
		"jti":      uuid.NewString(),
		"type":     "access",
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secret))
	if err != nil {
		return "", "", 0, err
	}

	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     now.Add(refreshExpiry).Unix(),
		"iat":     now.Unix(),
		"iss":     "ospanel",
		"aud":      "ospanel",
		"jti":      uuid.NewString(),
		"type":    "refresh",
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(secret))
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, int(accessExpiry.Seconds()), nil
}

// sanitizeUser hassas bilgileri temizler
func sanitizeUser(user *model.User) *model.User {
	u := *user
	u.PasswordHash = ""
	u.TOTPSecret = ""
	u.LoginAttempts = 0
	u.LockedUntil = nil
	return &u
}
