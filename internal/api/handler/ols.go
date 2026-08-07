package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// OLSHandler OLS WebAdmin proxy
type OLSHandler struct {
	adminURL string
	username string
	password string
}

// NewOLSHandler OLS proxy handler oluşturur
func NewOLSHandler(adminURL string) *OLSHandler {
	user := "admin"
	pass := ""

	// Şifreyi dosyadan oku (install script kaydeder)
	if data, err := os.ReadFile("/etc/ospanel/ols_admin_pass"); err == nil {
		pass = strings.TrimSpace(string(data))
	}

	// Fallback: env'den dene
	if pass == "" {
		pass = os.Getenv("OSPANEL_ADMIN_PASS")
	}

	return &OLSHandler{
		adminURL: adminURL,
		username: user,
		password: pass,
	}
}

// Proxy OLS WebAdmin'e proxy yapar (auto-login)
func (h *OLSHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	target, _ := url.Parse(h.adminURL)
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Basic Auth header ekle
	auth := h.username + ":" + h.password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))

	r.Host = target.Host
	r.Header.Set("Authorization", "Basic "+encoded)

	proxy.ServeHTTP(w, r)
}

// GetOLSInfo OLS bilgilerini döndürür
func (h *OLSHandler) GetOLSInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"url":      h.adminURL,
		"username": h.username,
		"has_pass": h.password != "",
		"proxy":    "/api/v1/ols/proxy",
		"message":  "OLS WebAdmin'e panel uzerinden otomatik giris yapabilirsiniz",
	})
}

// GetOLSAuthURL direkt OLS URL'i döndürür (basic auth embed edilmiş)
func (h *OLSHandler) GetOLSAuthURL(w http.ResponseWriter, r *http.Request) {
	if h.password == "" {
		writeJSON(w, http.StatusOK, map[string]string{"url": h.adminURL})
		return
	}

	// http://admin:pass@host:7080/ formatı
	authURL := strings.Replace(h.adminURL, "://", "://"+h.username+":"+h.password+"@", 1)
	writeJSON(w, http.StatusOK, map[string]string{
		"url":     authURL,
		"message": "Bu URL ile OLS Admin'e otomatik giris yapabilirsiniz",
	})
}

// LoginInfo OLS login bilgilerini dondurur (sifre maskeli)
func (h *OLSHandler) LoginInfo(w http.ResponseWriter, r *http.Request) {
	maskedPass := ""
	if len(h.password) > 2 {
		maskedPass = h.password[:2] + strings.Repeat("*", len(h.password)-2)
	}
	host := strings.TrimPrefix(strings.TrimPrefix(h.adminURL, "http://"), "https://")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ols_admin_url": h.adminURL,
		"username":      h.username,
		"has_password":  h.password != "",
		"masked_pass":   maskedPass,
		"proxy_url":     "/api/v1/ols/proxy",
	})
}

// ChangePassword OLS admin sifresini degistirir
func (h *OLSHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Yeni sifre gerekli"})
		return
	}

	// Sifreyi dosyaya kaydet
	os.MkdirAll("/etc/ospanel", 0755)
	if err := os.WriteFile("/etc/ospanel/ols_admin_pass", []byte(req.NewPassword), 0600); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Sifre kaydedilemedi"})
		return
	}

	// OLS admin sifresini degistir
	adminScript := "/usr/local/lsws/admin/misc/admpass.sh"
	if _, err := os.Stat(adminScript); err == nil {
		cmd := exec.Command("bash", adminScript, req.NewPassword)
		if out, err := cmd.CombinedOutput(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "OLS sifre degistirilemedi: " + string(out),
			})
			return
		}
	}

	h.password = req.NewPassword
	writeJSON(w, http.StatusOK, map[string]string{"message": "OLS admin sifresi degistirildi"})
}
