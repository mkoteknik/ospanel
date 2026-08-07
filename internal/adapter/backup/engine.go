package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// Result backup sonucu
type Result struct {
	ID        int64     `json:"id"`
	JobID     int64     `json:"job_id"`
	Status    string    `json:"status"`
	FilePath  string    `json:"file_path,omitempty"`
	Size      int64     `json:"size_bytes"`
	Error     string    `json:"error,omitempty"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// Engine yedekleme motoru
type Engine struct {
	log        *logger.Logger
	backupDir  string
	results    []Result
	mu         sync.RWMutex
	schedules  map[int64]chan struct{}
}

// NewEngine yeni backup motoru
func NewEngine(log *logger.Logger) *Engine {
	backupDir := "/var/backups/ospanel"
	os.MkdirAll(backupDir, 0750)
	return &Engine{
		log:       log,
		backupDir: backupDir,
		schedules: make(map[int64]chan struct{}),
	}
}

// Run backup işlemini çalıştırır
func (e *Engine) Run(job *model.BackupJob) error {
	result := Result{
		JobID:     job.ID,
		StartTime: time.Now(),
	}

	var backupPath string
	var err error

	switch job.Type {
	case "full", "incremental", "differential":
		backupPath, err = e.fileBackup(job)
	case "database":
		backupPath, err = e.databaseBackup(job)
	default:
		backupPath, err = e.fileBackup(job)
	}

	result.EndTime = time.Now()

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		e.log.Errorw("backup başarısız", "job_id", job.ID, "error", err)
	} else {
		result.Status = "completed"
		result.FilePath = backupPath
		if info, statErr := os.Stat(backupPath); statErr == nil {
			result.Size = info.Size()
		}
		e.log.Infow("backup tamamlandı", "job_id", job.ID, "size", result.Size)

		// Retention: eski backup'ları temizle
		e.cleanup(job)
	}

	e.mu.Lock()
	e.results = append(e.results, result)
	if len(e.results) > 100 {
		e.results = e.results[len(e.results)-100:]
	}
	e.mu.Unlock()

	return err
}

// fileBackup dosya yedeklemesi (tar.gz)
func (e *Engine) fileBackup(job *model.BackupJob) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("backup_%d_%s.tar.gz", job.ID, timestamp)
	destPath := filepath.Join(e.backupDir, filename)

	// Domain docroot'unu bul ve yedekle
	var sourceDir string
	if job.DomainID != nil {
		// Domain'e ait docroot (store'dan alınamaz, engine'de domain store yok)
		// Fallback: /home altındaki tüm kullanıcı dizinlerini yedekle
		sourceDir = "/home"
	} else {
		sourceDir = "/home"
	}

	cmd := exec.Command("tar", "-czf", destPath, "-C", sourceDir, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tar arşivleme hatası: %s - %w", string(out), err)
	}

	return destPath, nil
}

// databaseBackup veritabanı yedeklemesi (mysqldump)
func (e *Engine) databaseBackup(job *model.BackupJob) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("dbbackup_%d_%s.sql.gz", job.ID, timestamp)
	destPath := filepath.Join(e.backupDir, filename)

	// mysqldump ile tüm veritabanlarını yedekle
	// Bağlantı bilgilerini config'den oku
	config := readDBConfig()

	args := []string{
		"--single-transaction",
		"--routines",
		"--triggers",
		"--all-databases",
		"-h", "127.0.0.1",
	}

	if config["DB_USER"] != "" {
		args = append(args, "-u", config["DB_USER"])
		if config["DB_PASS"] != "" {
			args = append(args, "-p"+config["DB_PASS"])
		}
	}

	// mysqldump | gzip
	dumpCmd := exec.Command("mysqldump", args...)
	gzipCmd := exec.Command("gzip")

	// Pipe: mysqldump -> gzip -> file
	gzipCmd.Stdin, _ = dumpCmd.StdoutPipe()
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("yedek dosyası oluşturulamadı: %w", err)
	}
	defer outFile.Close()
	gzipCmd.Stdout = outFile

	if err := gzipCmd.Start(); err != nil {
		return "", fmt.Errorf("gzip başlatılamadı: %w", err)
	}
	if err := dumpCmd.Start(); err != nil {
		return "", fmt.Errorf("mysqldump başlatılamadı: %w", err)
	}

	dumpCmd.Wait()
	gzipCmd.Wait()

	return destPath, nil
}

// Schedule periyodik backup başlatır
func (e *Engine) Schedule(job *model.BackupJob) {
	// Mevcut schedule varsa durdur
	e.mu.Lock()
	if stop, exists := e.schedules[job.ID]; exists {
		close(stop)
	}
	stop := make(chan struct{})
	e.schedules[job.ID] = stop
	e.mu.Unlock()

	go func() {
		e.log.Infow("backup schedule başlatıldı", "job_id", job.ID, "schedule", job.Schedule)
		// Basit cron benzeri döngü
		for {
			select {
			case <-stop:
				return
			default:
				now := time.Now()
				next := getNextRun(job.Schedule, now)
				time.Sleep(time.Until(next))

				select {
				case <-stop:
					return
				default:
					e.log.Infow("zamanlanmış backup başlatılıyor", "job_id", job.ID)
					e.Run(job)
				}
			}
		}
	}()
}

// ListResults son backup sonuçlarını döndürür
func (e *Engine) ListResults() []Result {
	e.mu.RLock()
	defer e.mu.RUnlock()
	results := make([]Result, len(e.results))
	copy(results, e.results)
	return results
}

// cleanup retention'dan eski backup'ları siler
func (e *Engine) cleanup(job *model.BackupJob) {
	if job.Retention <= 0 {
		return
	}

	cutoff := time.Now().Add(-time.Duration(job.Retention) * 24 * time.Hour)

	entries, err := os.ReadDir(e.backupDir)
	if err != nil {
		return
	}

	prefix := fmt.Sprintf("backup_%d_", job.ID)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(e.backupDir, entry.Name())
			if err := os.Remove(path); err == nil {
				e.log.Infow("eski backup silindi", "file", entry.Name())
			}
		}
	}
}

// getNextRun basit cron parser
func getNextRun(schedule string, now time.Time) time.Time {
	// Basit implementasyon: her gün belirli saatte
	// Format: "HH:MM" veya "daily"
	if schedule == "daily" || schedule == "" {
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
		return next
	}
	// HH:MM formatı
	parts := strings.Split(schedule, ":")
	if len(parts) == 2 {
		hour := 2
		minute := 0
		fmt.Sscanf(parts[0], "%d", &hour)
		fmt.Sscanf(parts[1], "%d", &minute)
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}
		return next
	}
	return now.Add(24 * time.Hour)
}

func readDBConfig() map[string]string {
	config := map[string]string{}
	data, err := os.ReadFile("/etc/ospanel/db.conf")
	if err != nil {
		return config
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return config
}
