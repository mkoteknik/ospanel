package handler

import (
	"encoding/json"
	"net/http"
)

// writeJSON JSON yanıtı yazar
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// getClientIP istekten IP adresini alır
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// encodeHash production placeholder
func encodeHash(salt, hash []byte) string {
	return "ospanel-v1$...placeholder..."
}
