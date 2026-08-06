package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter basit token-bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     float64
	burst    int
}

type visitor struct {
	tokens    float64
	lastCheck time.Time
}

// NewRateLimiter yeni bir rate limiter oluşturur
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
	}

	// Periyodik temizlik
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			rl.cleanup()
		}
	}()

	return rl
}

// Limit rate limiting middleware'i
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			v = &visitor{
				tokens:    float64(rl.burst),
				lastCheck: time.Now(),
			}
			rl.visitors[ip] = v
		}

		// Token yenileme
		now := time.Now()
		elapsed := now.Sub(v.lastCheck).Seconds()
		v.tokens += elapsed * rl.rate
		if v.tokens > float64(rl.burst) {
			v.tokens = float64(rl.burst)
		}
		v.lastCheck = now

		// Token tüket
		if v.tokens < 1 {
			rl.mu.Unlock()
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"Çok fazla istek, lütfen bekleyin"}`, http.StatusTooManyRequests)
			return
		}
		v.tokens--
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// cleanup eski ziyaretçi kayıtlarını temizler
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, v := range rl.visitors {
		if time.Since(v.lastCheck) > 1*time.Hour {
			delete(rl.visitors, ip)
		}
	}
}

// getClientIP istemci IP'sini alır (proxy arkası desteği ile)
func getClientIP(r *http.Request) string {
	// X-Forwarded-For header'ını kontrol et
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := splitAndTrim(xff, ",")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	// X-Real-IP header'ını kontrol et
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// RemoteAddr kullan
	ip := r.RemoteAddr
	if idx := len(ip) - 1; idx >= 0 {
		// Port kısmını kaldır
		for i := len(ip) - 1; i >= 0; i-- {
			if ip[i] == ':' {
				return ip[:i]
			}
		}
	}
	return ip
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
