package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/gorilla/websocket"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

var termUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	Subprotocols: []string{"xterm"},
}

// TerminalHandler Web SSH terminal
type TerminalHandler struct {
	log *logger.Logger
}

func NewTerminalHandler(log *logger.Logger) *TerminalHandler { return &TerminalHandler{log: log} }

// WebSocket terminal bağlantısı
func (h *TerminalHandler) Connect(w http.ResponseWriter, r *http.Request) {
	conn, err := termUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorw("terminal websocket hatası", "error", err)
		return
	}
	defer conn.Close()

	// Shell başlat (cross-platform)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell.exe")
	} else {
		cmd = exec.Command("bash", "--login")
	}

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	cmd.Start()

	// stdout -> websocket
	go func() { io.Copy(&wsWriter{conn}, stdout) }()
	go func() { io.Copy(&wsWriter{conn}, stderr) }()

	// websocket -> stdin
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { break }

		var req map[string]interface{}
		json.Unmarshal(msg, &req)

		if data, ok := req["data"]; ok {
			stdin.Write([]byte(data.(string)))
		}
	}

	cmd.Process.Kill()
}

// Log Viewer - dosyayı stream eder
func (h *TerminalHandler) LogStream(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		file = "/usr/local/lsws/logs/error.log"
	}

	// Son 100 satırı gönder
	cmd := exec.Command("tail", "-100", file)
	out, _ := cmd.CombinedOutput()

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
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}
