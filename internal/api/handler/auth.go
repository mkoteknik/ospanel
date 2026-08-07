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

	// Kullanıcıyı bul
	user, err := h.store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		h.log.Errorw("login başarısız - kullanıcı bulunamadı", "username", req.Username)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz kullanıcı adı veya şifre"})
		return
	}

	// Hesap durumu kontrolü
	if user.Status != model.StatusActive {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Hesap aktif değil"})
		return
	}

	// Kilitli hesap kontrolü
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Hesap geçici olarak kilitlendi"})
		return
	}

	// Şifre doğrulama
	if !verifyPassword(req.Password, user.PasswordHash) {
		// Başarısız giriş denemesi
		attempts := user.LoginAttempts + 1
		h.store.UpdateLoginAttempts(r.Context(), user.ID, attempts)

		h.log.Warnw("login başarısız - şifre hatalı", "username", req.Username, "attempt", attempts)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz kullanıcı adı veya şifre"})
		return
	}

	// 2FA kontrolü
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

	// Token üret
	accessToken, refreshToken, expiresIn, err := generateTokens(user, h.jwtSecret)
	if err != nil {
		h.log.Errorw("token oluşturma hatası", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Token oluşturulamadı"})
		return
	}

	// Başarılı giriş kaydı
	now := time.Now()
	h.store.UpdateLoginAttempts(r.Context(), user.ID, 0)
	user.LastLoginAt = &now
	user.LastLoginIP = getClientIP(r)
	h.store.UpdateUser(r.Context(), user)

	h.log.Infow("login başarılı", "username", req.Username, "ip", getClientIP(r))

	writeJSON(w, http.StatusOK, model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         sanitizeUser(user),
	})
}

// Logout çıkış işlemi
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// JWT stateless olduğu için client tarafında token silinir
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

	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz refresh token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Geçersiz token"})
		return
	}

	userID := int64(claims["user_id"].(float64))
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	accessToken, refreshToken, expiresIn, _ := generateTokens(user, h.jwtSecret)

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
	h.store.UpdateUser(r.Context(), user)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Şifre başarıyla değiştirildi"})
}

// Setup2FA 2FA kurulumu
func (h *AuthHandler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	var req struct {
		Enable bool   `json:"enable"`
		Code   string `json:"code,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Enable {
		// TOTP kurulumu (gelecek)
		user.TOTPEnabled = true
		user.TOTPSecret = "TODO-generate-secret"
	} else {
		user.TOTPEnabled = false
		user.TOTPSecret = ""
	}

	h.store.UpdateUser(r.Context(), user)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totp_enabled": user.TOTPEnabled,
	})
}

// hashPassword Argon2id ile şifre hashler, format: ospanel$v1$<hex salt>$<hex hash>
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return "ospanel$v1$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(hash), nil
}

// verifyPassword hashlenmiş şifreyi doğrular
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
	computed := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(computed, expectedHash) == 1
}

// generateTokens JWT token çifti oluşturur
func generateTokens(user *model.User, secret string) (string, string, int, error) {
	accessExpiry := 15 * time.Minute
	refreshExpiry := 7 * 24 * time.Hour

	accessClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     string(user.Role),
		"exp":      time.Now().Add(accessExpiry).Unix(),
		"iat":      time.Now().Unix(),
		"type":     "access",
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secret))
	if err != nil {
		return "", "", 0, err
	}

	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(refreshExpiry).Unix(),
		"iat":     time.Now().Unix(),
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



