package handler

import (
	"net/http"

	"github.com/mkoteknik/ospanel/internal/adapter/cache"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// CacheHandler Redis cache yönetimi
type CacheHandler struct {
	redis *cache.RedisClient
	log   *logger.Logger
}

// NewCacheHandler yeni CacheHandler
func NewCacheHandler(redis *cache.RedisClient, log *logger.Logger) *CacheHandler {
	return &CacheHandler{redis: redis, log: log}
}

// Status Redis durumunu döndürür
func (h *CacheHandler) Status(w http.ResponseWriter, r *http.Request) {
	stats := h.redis.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

// Info detaylı Redis bilgisi
func (h *CacheHandler) Info(w http.ResponseWriter, r *http.Request) {
	info := h.redis.GetInfo()
	writeJSON(w, http.StatusOK, info)
}

// FlushCache cache temizler
func (h *CacheHandler) FlushCache(w http.ResponseWriter, r *http.Request) {
	if err := h.redis.FlushCache(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.log.Infow("redis cache temizlendi")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Cache temizlendi"})
}
