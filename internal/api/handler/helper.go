package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// writeJSON JSON yanıtı yazar
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// getClientIP istekten IP adresini alır — trusted proxy ise XFF'e güvenir
func getClientIP(r *http.Request) string {
	remoteIP := stripPort(r.RemoteAddr)
	if isTrustedProxy(remoteIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				candidate := strings.TrimSpace(parts[0])
				if net.ParseIP(candidate) != nil {
					return candidate
				}
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			candidate := strings.TrimSpace(xri)
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}
	return remoteIP
}

func isTrustedProxy(ip string) bool {
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, block := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7"} {
		_, cidr, _ := net.ParseCIDR(block)
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

func stripPort(addr string) string {
	if strings.HasPrefix(addr, "[") {
		if idx := strings.LastIndex(addr, "]"); idx != -1 {
			if len(addr) > idx+1 && addr[idx+1] == ':' {
				return addr[1:idx]
			}
			return addr[1:idx]
		}
	}
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		ipPart := addr[:idx]
		if net.ParseIP(ipPart) != nil {
			return ipPart
		}
		if strings.Count(addr, ":") == 1 {
			return addr[:idx]
		}
	}
	return addr
}

// encodeHash production placeholder
func encodeHash(salt, hash []byte) string {
	return "ospanel-v1$...placeholder..."
}
