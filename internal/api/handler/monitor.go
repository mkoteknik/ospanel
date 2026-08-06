package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/openspeed-panel/ospanel/internal/adapter/system"
	"github.com/openspeed-panel/ospanel/internal/pkg/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// MonitorHandler sistem izleme
type MonitorHandler struct {
	log *logger.Logger
}

// NewMonitorHandler yeni MonitorHandler
func NewMonitorHandler(log *logger.Logger) *MonitorHandler {
	return &MonitorHandler{log: log}
}

// Stats gerçek sistem istatistiklerini döndürür
func (h *MonitorHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats := system.GetSystemStats()
	writeJSON(w, http.StatusOK, stats)
}

// LiveStats WebSocket canlı istatistikler
func (h *MonitorHandler) LiveStats(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorw("websocket upgrade hatası", "error", err)
		return
	}
	defer conn.Close()

	h.log.Infow("monitoring websocket bağlandı")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// İstek tipine göre yanıt
		if string(msg) == "stats" {
			stats := system.GetSystemStats()
			conn.WriteJSON(map[string]interface{}{
				"type": "stats",
				"data": stats,
			})
		}
	}
}
