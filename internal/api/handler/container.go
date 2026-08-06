package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/adapter/container"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// ContainerHandler Docker/Podman yönetimi
type ContainerHandler struct {
	docker *container.DockerClient
	log    *logger.Logger
}

// NewContainerHandler yeni ContainerHandler
func NewContainerHandler(docker *container.DockerClient, log *logger.Logger) *ContainerHandler {
	return &ContainerHandler{docker: docker, log: log}
}

// Stats konteyner istatistikleri
func (h *ContainerHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats := h.docker.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

// List kontenyner listesi
func (h *ContainerHandler) List(w http.ResponseWriter, r *http.Request) {
	containers, err := h.docker.ListContainers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if containers == nil {
		containers = []container.Container{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"containers": containers,
		"total":      len(containers),
	})
}

// Start konteyner başlatır
func (h *ContainerHandler) Start(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.docker.StartContainer(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Başlatıldı"})
}

// Stop konteyner durdurur
func (h *ContainerHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.docker.StopContainer(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Durduruldu"})
}

// Restart konteyner yeniden başlatır
func (h *ContainerHandler) Restart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.docker.RestartContainer(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Yeniden başlatıldı"})
}
