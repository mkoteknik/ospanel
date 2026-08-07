package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// TOTPHandler 2FA yönetimi
type TOTPHandler struct {
	store store.Store
	log   *logger.Logger
}

func NewTOTPHandler(s store.Store, log *logger.Logger) *TOTPHandler { return &TOTPHandler{store: s, log: log} }

// Setup 2FA kurulumu başlatır, secret ve QR URL döndürür
func (h *TOTPHandler) Setup(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	if user.TOTPEnabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "2FA zaten aktif"})
		return
	}

	// Rastgele secret üret
	secret := make([]byte, 20)
	rand.Read(secret)
	encodedSecret := strings.TrimRight(base32.StdEncoding.EncodeToString(secret), "=")

	// QR URL oluştur (Google Authenticator formatı)
	qrURL := "otpauth://totp/OpenSpeed%20Panel:" + user.Username + "?secret=" + encodedSecret + "&issuer=OSPanel"

	// Secret'ı geçici olarak kaydet
	user.TOTPSecret = encodedSecret
	h.store.UpdateUser(r.Context(), user)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"secret": encodedSecret,
		"qr_url": qrURL,
		"message": "Bu QR'ı Google Authenticator ile tarayın",
	})
}

// Verify 2FA doğrulama
func (h *TOTPHandler) Verify(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if !validateTOTP(req.Code, user.TOTPSecret) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz kod"})
		return
	}

	user.TOTPEnabled = true
	h.store.UpdateUser(r.Context(), user)
	h.log.Infow("2FA aktifleştirildi", "user", user.Username)
	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA başarıyla aktifleştirildi"})
}

// Disable 2FA devre dışı bırakır
func (h *TOTPHandler) Disable(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	user.TOTPEnabled = false
	user.TOTPSecret = ""
	h.store.UpdateUser(r.Context(), user)
	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA devre dışı bırakıldı"})
}

// Status 2FA durumunu döndürür
func (h *TOTPHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	user, _ := h.store.GetUser(r.Context(), userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"enabled": user.TOTPEnabled})
}

// CheckTotpCode giriş sırasında TOTP kodunu doğrular
func CheckTotpCode(secret, code string) bool { return validateTOTP(code, secret) }

// TOTP implementasyonu
func validateTOTP(code, secret string) bool {
	if len(code) != 6 { return false }
	// 30 saniye tolerans (±1 adım)
	for i := -1; i <= 1; i++ {
		if generateTOTP(secret, time.Now().Unix()+int64(i*30)) == code {
			return true
		}
	}
	return false
}

func generateTOTP(secret string, timestamp int64) string {
	counter := timestamp / 30
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	key, _ := base32.StdEncoding.DecodeString(secret + strings.Repeat("=", (8-len(secret)%8)%8))
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	binary := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	otp := binary % 1000000

	result := make([]byte, 6)
	result[0] = byte('0' + otp/100000)
	result[1] = byte('0' + (otp/10000)%10)
	result[2] = byte('0' + (otp/1000)%10)
	result[3] = byte('0' + (otp/100)%10)
	result[4] = byte('0' + (otp/10)%10)
	result[5] = byte('0' + otp%10)

	return string(result)
}

// generateQR PNG QR kodu olusturur
func generateQR(w http.ResponseWriter, data string) {
	// Basit QR kodu: terminal-friendly text tabanli QR
	// Production'da go-qrcode veya rsc.io/qr kullanilabilir
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("QR Code URL: " + data + "\n"))
	w.Write([]byte("Bu URL'yi Google Authenticator veya baska bir TOTP uygulamasi ile tarayin.\n"))
}
