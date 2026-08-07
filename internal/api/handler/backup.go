package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/adapter/backup"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// BackupHandler yedekleme yönetimi
type BackupHandler struct {
	store  store.Store
	log    *logger.Logger
	engine *backup.Engine
}

// NewBackupHandler yeni BackupHandler
func NewBackupHandler(s store.Store, log *logger.Logger) *BackupHandler {
	return &BackupHandler{store: s, log: log, engine: backup.NewEngine(log)}
}

// List kullanıcının backup job'larını listeler
func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	jobs, err := h.store.ListBackupJobs(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Yedekleme işleri listelenemedi"})
		return
	}
	if jobs == nil {
		jobs = []*model.BackupJob{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"backups": jobs, "total": len(jobs)})
}

// Create yeni backup job oluşturur
func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req struct {
		DomainID    *int64 `json:"domain_id,omitempty"`
		Type        string `json:"type"`
		Destination string `json:"destination"`
		Schedule    string `json:"schedule"`
		Retention   int    `json:"retention"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if req.Type == "" {
		req.Type = "full"
	}
	if req.Destination == "" {
		req.Destination = "local"
	}
	if req.Retention <= 0 {
		req.Retention = 7
	}

	job := &model.BackupJob{
		UserID:      userID,
		DomainID:    req.DomainID,
		Type:        req.Type,
		Destination: req.Destination,
		Schedule:    req.Schedule,
		Retention:   req.Retention,
	}

	if err := h.store.CreateBackupJob(r.Context(), job); err != nil {
		h.log.Errorw("backup job oluşturulamadı", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Yedekleme işi oluşturulamadı"})
		return
	}

	// Schedule varsa engine'e kaydet
	if req.Schedule != "" {
		h.engine.Schedule(job)
	}

	// Next run hesapla
	nextRun := time.Now().Add(24 * time.Hour)
	job.NextRun = &nextRun
	h.store.UpdateBackupJob(r.Context(), job)

	h.log.Infow("backup job oluşturuldu", "id", job.ID, "type", req.Type)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"backup":  job,
		"message": "Yedekleme işi oluşturuldu",
	})
}

// Update backup job günceller
func (h *BackupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	job, err := h.store.GetBackupJob(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Yedekleme işi bulunamadı"})
		return
	}

	var updates map[string]interface{}
	json.NewDecoder(r.Body).Decode(&updates)

	if v, ok := updates["schedule"]; ok {
		job.Schedule = v.(string)
	}
	if v, ok := updates["retention"]; ok {
		job.Retention = int(v.(float64))
	}
	if v, ok := updates["status"]; ok {
		job.Status = v.(string)
	}

	job.UpdatedAt = time.Now()
	if err := h.store.UpdateBackupJob(r.Context(), job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Yedekleme işi güncellenemedi"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"backup": job, "message": "Güncellendi"})
}

// Delete backup job siler
func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.store.DeleteBackupJob(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Yedekleme işi silinemedi"})
		return
	}
	h.log.Infow("backup job silindi", "id", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Yedekleme işi silindi"})
}

// Run backup'ı manuel çalıştırır
func (h *BackupHandler) Run(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	job, err := h.store.GetBackupJob(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Yedekleme işi bulunamadı"})
		return
	}

	h.log.Infow("backup manuel başlatılıyor", "id", id, "type", job.Type)

	// Async çalıştır
	go func() {
		if err := h.engine.Run(job); err != nil {
			h.log.Errorw("backup başarısız", "id", job.ID, "error", err)
			job.Status = "failed"
		} else {
			job.Status = "completed"
			now := time.Now()
			job.LastRun = &now
		}
		h.store.UpdateBackupJob(r.Context(), job)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"message": "Yedekleme başlatıldı"})
}

// ListBackups son backup sonuçlarını listeler
func (h *BackupHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	results := h.engine.ListResults()
	if results == nil {
		results = []backup.Result{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results, "total": len(results)})
}
