package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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

// LoginInfo OLS login bilgilerini döndürür
func (h *OLSHandler) LoginInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ols_admin_url": h.adminURL,
		"username":      h.username,
		"password":      h.password,
		"direct_url":    fmt.Sprintf("http://%s:%s@%s", h.username, h.password,
			strings.TrimPrefix(strings.TrimPrefix(h.adminURL, "http://"), "https://")),
	})
}
