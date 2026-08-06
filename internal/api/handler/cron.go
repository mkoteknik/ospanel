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

// Add cron ekler
func (h *CronHandler) Add(w http.ResponseWriter, r *http.Request) {
	var job CronJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	cronExpr := fmt.Sprintf("%s %s %s %s %s %s", job.Minute, job.Hour, job.Day, job.Month, job.Weekday, job.Command)

	if err := h.writeCron(cronExpr); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	h.log.Infow("cron eklendi", "expr", cronExpr)
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Cron eklendi", "expression": cronExpr})
}

// Delete cron siler
func (h *CronHandler) Delete(w http.ResponseWriter, r *http.Request) {
	command := r.URL.Query().Get("command")
	jobs := h.getCronJobs("")
	var newJobs []string
	for _, j := range jobs {
		if !strings.Contains(j.Command, command) && j.Command != command {
			newJobs = append(newJobs, fmt.Sprintf("%s %s %s %s %s %s",
				j.Minute, j.Hour, j.Day, j.Month, j.Weekday, j.Command))
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
			if len(parts) < 6 { continue }
			jobs = append(jobs, CronJob{
				Minute: parts[0], Hour: parts[1], Day: parts[2],
				Month: parts[3], Weekday: parts[4],
				Command: strings.Join(parts[5:], " "),
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
