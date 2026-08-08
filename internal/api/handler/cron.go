package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// CronHandler cron job yönetimi
type CronHandler struct{ log *logger.Logger }

func NewCronHandler(log *logger.Logger) *CronHandler { return &CronHandler{log: log} }

type CronJob struct {
	Minute  string `json:"minute"`
	Hour    string `json:"hour"`
	Day     string `json:"day"`
	Month   string `json:"month"`
	Weekday string `json:"weekday"`
	Command string `json:"command"`
	User    string `json:"user,omitempty"`
}

// List cron'ları listeler
func (h *CronHandler) List(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	jobs := h.getCronJobs(user)
	if jobs == nil { jobs = []CronJob{} }
	writeJSON(w, http.StatusOK, map[string]interface{}{"jobs": jobs, "total": len(jobs)})
}

// Add cron ekler (admin-only, valide edilmiş)
func (h *CronHandler) Add(w http.ResponseWriter, r *http.Request) {
	var job CronJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	// Zorunlu alan validasyonu
	if job.Command == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Komut gerekli"})
		return
	}

	// Cron zaman alanları varsayılan değerler
	if job.Minute == "" { job.Minute = "*" }
	if job.Hour == "" { job.Hour = "*" }
	if job.Day == "" { job.Day = "*" }
	if job.Month == "" { job.Month = "*" }
	if job.Weekday == "" { job.Weekday = "*" }
	if job.User == "" { job.User = "root" }

	// Cron zaman alanı validasyonu (sadece *, rakam, /, -, , karakterlerine izin ver)
	if !isValidCronField(job.Minute) || !isValidCronField(job.Hour) ||
		!isValidCronField(job.Day) || !isValidCronField(job.Month) ||
		!isValidCronField(job.Weekday) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz cron zaman formatı"})
		return
	}

	// Kullanıcı adı validasyonu
	if !isValidUsername(job.User) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz kullanıcı adı"})
		return
	}

	// Komut içinde yeni satır karakteri engelle (cron injection)
	if strings.Contains(job.Command, "\n") || strings.Contains(job.Command, "\r") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Komut geçersiz karakter içeriyor"})
		return
	}
	job.Command = strings.TrimSpace(job.Command)
	if len(job.Command) > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Komut çok uzun (max 1000 karakter)"})
		return
	}

	// cron.d formatı: minute hour day month weekday USER command
	cronExpr := fmt.Sprintf("%s %s %s %s %s %s %s",
		job.Minute, job.Hour, job.Day, job.Month, job.Weekday, job.User, job.Command)

	if err := h.writeCron(cronExpr); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	h.log.Infow("cron eklendi", "user", job.User, "command", job.Command)
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Cron eklendi", "expression": cronExpr})
}

// Delete cron siler
func (h *CronHandler) Delete(w http.ResponseWriter, r *http.Request) {
	command := r.URL.Query().Get("command")
	if command == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Komut gerekli"})
		return
	}

	jobs := h.getCronJobs("")
	var newJobs []string
	for _, j := range jobs {
		if j.Command != command {
			newJobs = append(newJobs, fmt.Sprintf("%s %s %s %s %s %s %s",
				j.Minute, j.Hour, j.Day, j.Month, j.Weekday, j.User, j.Command))
		}
	}

	file := "/etc/cron.d/ospanel-custom"
	if len(newJobs) == 0 {
		os.Remove(file)
	} else {
		os.WriteFile(file, []byte(strings.Join(newJobs, "\n")+"\n"), 0644)
	}

	h.log.Infow("cron silindi", "command", command)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Cron silindi"})
}

func (h *CronHandler) getCronJobs(user string) []CronJob {
	var jobs []CronJob
	files := []string{"/etc/cron.d/ospanel-custom", "/etc/cron.d/ospanel-ssl-renew"}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil { continue }
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") { continue }
			parts := strings.Fields(line)
			// cron.d formatı: minute hour day month weekday USER command [args...]
			if len(parts) < 7 { continue }
			jobs = append(jobs, CronJob{
				Minute: parts[0], Hour: parts[1], Day: parts[2],
				Month: parts[3], Weekday: parts[4], User: parts[5],
				Command: strings.Join(parts[6:], " "),
			})
		}
	}

	// crontab -l
	if user != "" {
		cmd := exec.Command("crontab", "-l", "-u", user)
		out, _ := cmd.CombinedOutput()
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") { continue }
			parts := strings.Fields(line)
			if len(parts) < 6 { continue }
			jobs = append(jobs, CronJob{
				Minute: parts[0], Hour: parts[1], Day: parts[2],
				Month: parts[3], Weekday: parts[4],
				Command: strings.Join(parts[5:], " "), User: user,
			})
		}
	}

	return jobs
}

func (h *CronHandler) writeCron(expr string) error {
	file := "/etc/cron.d/ospanel-custom"
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil { return err }
	defer f.Close()
	_, err = f.WriteString(expr + "\n")
	return err
}

// isValidCronField cron zaman alanlarını valide eder
func isValidCronField(field string) bool {
	if field == "" {
		return false
	}
	// İzin verilen: * , - / 0-9
	for _, c := range field {
		if !((c >= '0' && c <= '9') || c == '*' || c == ',' || c == '-' || c == '/') {
			return false
		}
	}
	return len(field) <= 20
}

// isValidUsername basit kullanıcı adı validasyonu
func isValidUsername(user string) bool {
	if len(user) == 0 || len(user) > 32 {
		return false
	}
	for _, c := range user {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
