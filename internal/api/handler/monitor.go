package handler

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/mkoteknik/ospanel/internal/adapter/system"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Sadece aynı origin'den gelen WS bağlantılarına izin ver
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Same-origin request (no Origin header)
		}
		host := r.Host
		if host == "" {
			host = r.URL.Host
		}
		// Origin ve Host karşılaştırması
		if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
			return true
		}
		// Development: localhost
		if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
			return true
		}
		return false
	},
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
