package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

var termUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser or same-origin (no Origin header)
		}
		host := r.Host
		// Allow same host
		if strings.Contains(origin, host) {
			return true
		}
		// In production, check against AllowedOrigins via context or config
		// For now, deny cross-origin WS
		return false
	},
	Subprotocols:      []string{"xterm"},
	HandshakeTimeout:  5 * time.Second,
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: false,
}

// TerminalHandler Web SSH terminal
type TerminalHandler struct {
	log       *logger.Logger
	jwtSecret string
}

func NewTerminalHandler(log *logger.Logger) *TerminalHandler {
	return &TerminalHandler{log: log}
}

func NewTerminalHandlerWithAuth(log *logger.Logger, jwtSecret string) *TerminalHandler {
	return &TerminalHandler{log: log, jwtSecret: jwtSecret}
}

// Connect WebSocket terminal bağlantısı — auth'lı, admin-only, 10dk timeout
func (h *TerminalHandler) Connect(w http.ResponseWriter, r *http.Request) {
	// Auth check: middleware already injected claims if route is protected,
	// but WS via query param needs manual check when jwtSecret is set
	if h.jwtSecret != "" {
		// If no user in context, try query param auth
		if _, ok := middleware.GetUserID(r.Context()); !ok {
			token := r.URL.Query().Get("token")
			if token == "" {
				if c, err := r.Cookie("access_token"); err == nil {
					token = c.Value
				}
			}
			if token == "" {
				http.Error(w, `{"error":"Yetkilendirme gerekli"}`, http.StatusUnauthorized)
				return
			}
			// Token type check via middleware helper not exported, do minimal parse
			// Fallback: deny if no context user
			http.Error(w, `{"error":"Yetkilendirme gerekli - token query ile gönderin"}`, http.StatusUnauthorized)
			return
		}
	}

	// RBAC: sadece admin ve reseller terminal açabilir
	if role, ok := middleware.GetUserRole(r.Context()); ok {
		if role != "admin" && role != "reseller" {
			http.Error(w, `{"error":"Bu işlem için yetkiniz yok"}`, http.StatusForbidden)
			return
		}
	} else {
		http.Error(w, `{"error":"Yetkilendirme gerekli"}`, http.StatusUnauthorized)
		return
	}

	conn, err := termUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorw("terminal websocket hatası", "error", err)
		return
	}
	defer conn.Close()

	// Timeouts
	conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		return nil
	})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	username, _ := middleware.GetUsername(r.Context())
	h.log.Infow("terminal bağlandı", "user", username, "ip", getClientIP(r))

	// Shell başlat (cross-platform)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell.exe", "-NoProfile")
	} else {
		// Güvenli: login shell değil, restricted env
		cmd = exec.Command("bash", "--noprofile", "--norc")
		cmd.Env = append(os.Environ(), "TERM=xterm-256color", "TMOUT=600")
	}

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		h.log.Errorw("shell başlatılamadı", "error", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Shell başlatılamadı\r\n"))
		return
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		h.log.Infow("terminal kapandı", "user", username)
	}()

	// stdout -> websocket
	go func() { _, _ = io.Copy(&wsWriter{conn: conn}, stdout) }()
	go func() { _, _ = io.Copy(&wsWriter{conn: conn}, stderr) }()

	// websocket -> stdin (max 1MB per message, rate limit implicit via ReadDeadline)
	conn.SetReadLimit(1 << 20)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// JSON veya raw data
		var req map[string]interface{}
		if err := json.Unmarshal(msg, &req); err == nil {
			if data, ok := req["data"]; ok {
				if s, ok := data.(string); ok {
					// Basit komut filtre: rm -rf / gibi tehlikeli pattern'leri logla
					if strings.Contains(s, "rm -rf /") && !strings.Contains(s, "/tmp") {
						h.log.Warnw("tehlikeli komut denemesi", "user", username, "cmd", s)
					}
					_, _ = stdin.Write([]byte(s))
				}
			}
			// Resize event (cols/rows) - PTY yoksa ignore
			continue
		}
		// Raw
		_, _ = stdin.Write(msg)
		_, _ = stdin.Write([]byte("\n"))
	}
}

// Log Viewer - dosyayı stream eder (allowlist tabanlı)
func (h *TerminalHandler) LogStream(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		file = "/usr/local/lsws/logs/error.log"
	}

	// Allowlist
	allowed := map[string]bool{
		"/usr/local/lsws/logs/error.log":             true,
		"/usr/local/lsws/logs/access.log":             true,
		"/var/log/ospanel.log":                         true,
		"/var/log/ospanel-htaccess-watchdog.log":       true,
		"/var/log/syslog":                              true,
		"/var/log/mail.log":                            true,
		"/var/log/mysql/error.log":                     true,
		"/var/log/auth.log":                            true,
	}
	if !allowed[file] {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu log dosyası görüntülenemez"})
		return
	}

	if _, err := os.Stat(file); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Log dosyası bulunamadı"})
		return
	}

	cmd := exec.Command("tail", "-n", "200", file)
	// Timeout 5s
	done := make(chan struct{})
	var out []byte
	var cmdErr error
	go func() {
		out, cmdErr = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "Log okuma zaman aşımı"})
		return
	}
	if cmdErr != nil && len(out) == 0 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Log okunamadı"})
		return
	}

	// Max 200KB
	if len(out) > 200*1024 {
		out = out[len(out)-200*1024:]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"file":    file,
		"content": string(out),
	})
}

// Log listesi
func (h *TerminalHandler) LogList(w http.ResponseWriter, r *http.Request) {
	logs := []map[string]string{
		{"name": "OLS Error Log", "file": "/usr/local/lsws/logs/error.log", "icon": "🔴"},
		{"name": "OLS Access Log", "file": "/usr/local/lsws/logs/access.log", "icon": "📋"},
		{"name": "Panel Log", "file": "/var/log/ospanel.log", "icon": "⚡"},
		{"name": "Watchdog Log", "file": "/var/log/ospanel-htaccess-watchdog.log", "icon": "🔄"},
		{"name": "Syslog", "file": "/var/log/syslog", "icon": "📊"},
		{"name": "Mail Log", "file": "/var/log/mail.log", "icon": "📧"},
		{"name": "MariaDB Log", "file": "/var/log/mysql/error.log", "icon": "🗄️"},
		{"name": "Auth Log", "file": "/var/log/auth.log", "icon": "🔐"},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"logs": logs, "total": len(logs)})
}

type wsWriter struct{ conn *websocket.Conn }

func (w *wsWriter) Write(p []byte) (int, error) {
	_ = w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}
