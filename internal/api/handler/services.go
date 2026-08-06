package handler

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// ServiceInfo servis bilgisi
type ServiceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
	SystemdName string `json:"systemd"`
	InstallCmd  string `json:"install_cmd"`
	Installed   bool   `json:"installed"`
	Active      bool   `json:"active"`
	Enabled     bool   `json:"enabled"`
}

var allServices = []ServiceInfo{
	{Name: "redis", DisplayName: "Redis Cache", Icon: "⚡", Category: "cache", SystemdName: "redis-server", InstallCmd: "apt install -y redis-server"},
	{Name: "fail2ban", DisplayName: "Fail2ban", Icon: "🛡️", Category: "security", SystemdName: "fail2ban", InstallCmd: "apt install -y fail2ban"},
	{Name: "mariadb", DisplayName: "MariaDB", Icon: "🗄️", Category: "database", SystemdName: "mariadb", InstallCmd: "apt install -y mariadb-server"},
	{Name: "postgresql", DisplayName: "PostgreSQL", Icon: "🐘", Category: "database", SystemdName: "postgresql", InstallCmd: "apt install -y postgresql"},
	{Name: "postfix", DisplayName: "Postfix", Icon: "📧", Category: "email", SystemdName: "postfix", InstallCmd: "apt install -y postfix"},
	{Name: "dovecot", DisplayName: "Dovecot IMAP/POP3", Icon: "📨", Category: "email", SystemdName: "dovecot", InstallCmd: "apt install -y dovecot-core dovecot-imapd dovecot-pop3d"},
	{Name: "pdns", DisplayName: "PowerDNS", Icon: "🔧", Category: "dns", SystemdName: "pdns", InstallCmd: "apt install -y pdns-server pdns-backend-sqlite3"},
	{Name: "podman", DisplayName: "Podman", Icon: "🐳", Category: "container", SystemdName: "podman", InstallCmd: "apt install -y podman"},
	{Name: "spamassassin", DisplayName: "SpamAssassin", Icon: "📊", Category: "email", SystemdName: "spamassassin", InstallCmd: "apt install -y spamassassin"},
	{Name: "opendkim", DisplayName: "OpenDKIM", Icon: "🔑", Category: "email", SystemdName: "opendkim", InstallCmd: "apt install -y opendkim opendkim-tools"},
	{Name: "ospanel-watchdog", DisplayName: ".htaccess Watchdog", Icon: "🔄", Category: "web", SystemdName: "ospanel-htaccess-watchdog", InstallCmd: ""},
}

// ServicesHandler servis yönetimi
type ServicesHandler struct{ log *logger.Logger }
func NewServicesHandler(log *logger.Logger) *ServicesHandler { return &ServicesHandler{log: log} }

// List tüm servisleri listeler
func (h *ServicesHandler) List(w http.ResponseWriter, r *http.Request) {
	var services []ServiceInfo
	for _, svc := range allServices {
		s := svc
		s.checkStatus()
		services = append(services, s)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"services": services, "total": len(services)})
}

// Action servis işlemi (start/stop/restart/enable/disable/install)
func (h *ServicesHandler) Action(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Service string `json:"service"`
		Action  string `json:"action"` // start, stop, restart, enable, disable, install
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	// Servisi bul
	var svc *ServiceInfo
	for _, s := range allServices {
		if s.Name == req.Service {
			svc = &s
			break
		}
	}
	if svc == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Servis bulunamadı"})
		return
	}

	var err error

	switch req.Action {
	case "install":
		if svc.InstallCmd == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Bu servis için kurulum komutu yok"})
			return
		}
		cmd := exec.Command("bash", "-c", svc.InstallCmd)
		out, e := cmd.CombinedOutput()
		if e != nil {
			err = e
			h.log.Errorw("servis kurulum başarısız", "service", svc.Name, "error", e, "out", string(out))
		} else {
			// Kurulumdan sonra başlat
			exec.Command("systemctl", "enable", svc.SystemdName).Run()
			exec.Command("systemctl", "start", svc.SystemdName).Run()
			h.log.Infow("servis kuruldu", "service", svc.Name)
		}
	case "start":
		_, e := exec.Command("systemctl", "start", svc.SystemdName).CombinedOutput()
		err = e
	case "stop":
		_, e := exec.Command("systemctl", "stop", svc.SystemdName).CombinedOutput()
		err = e
	case "restart":
		_, e := exec.Command("systemctl", "restart", svc.SystemdName).CombinedOutput()
		err = e
	case "enable":
		_, e := exec.Command("systemctl", "enable", svc.SystemdName).CombinedOutput()
		err = e
	case "disable":
		_, e := exec.Command("systemctl", "disable", svc.SystemdName).CombinedOutput()
		err = e
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz işlem: " + req.Action})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Güncel durumu döndür
	svc.checkStatus()
	writeJSON(w, http.StatusOK, map[string]interface{}{"service": svc, "message": req.Action + " tamamlandı"})
}

func (s *ServiceInfo) checkStatus() {
	// Kurulu mu?
	if _, err := exec.LookPath(strings.Split(s.SystemdName, ".")[0]); err == nil {
		s.Installed = true
	} else if out, _ := exec.Command("dpkg", "-l", s.Name+"*").CombinedOutput(); strings.Contains(string(out), "ii") {
		s.Installed = true
	} else if out, _ := exec.Command("which", s.SystemdName).CombinedOutput(); len(out) > 0 {
		s.Installed = true
	}

	// systemd durumu
	if out, err := exec.Command("systemctl", "is-active", s.SystemdName).CombinedOutput(); err == nil {
		s.Active = strings.TrimSpace(string(out)) == "active"
	}
	if out, err := exec.Command("systemctl", "is-enabled", s.SystemdName).CombinedOutput(); err == nil {
		s.Enabled = strings.TrimSpace(string(out)) == "enabled"
	}
}
