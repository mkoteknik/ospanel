package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// AppTemplate one-click deploy şablonu
type AppTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Ports       string `json:"ports"`
	Env         string `json:"env"`
	Volumes     string `json:"volumes"`
	Category    string `json:"category"`
}

var appTemplates = []AppTemplate{
	{ID: "wordpress", Name: "WordPress", Icon: "📝", Category: "CMS",
		Description: "Dünyanın en popüler içerik yönetim sistemi",
		Image: "wordpress:latest", Ports: "80:80", Env: "WORDPRESS_DB_HOST=db\nWORDPRESS_DB_USER=wp\nWORDPRESS_DB_PASSWORD=secret\nWORDPRESS_DB_NAME=wordpress",
	},
	{ID: "nodejs", Name: "Node.js App", Icon: "🟢", Category: "Runtime",
		Description: "Node.js uygulaması - Express, Fastify veya vanilla",
		Image: "node:20-alpine", Ports: "3000:3000", Env: "NODE_ENV=production",
		Volumes: "./app:/app",
	},
	{ID: "python", Name: "Python Flask", Icon: "🐍", Category: "Runtime",
		Description: "Python Flask/FastAPI web uygulaması",
		Image: "python:3.12-slim", Ports: "5000:5000", Env: "FLASK_ENV=production",
		Volumes: "./app:/app",
	},
	{ID: "redis", Name: "Redis", Icon: "⚡", Category: "Database",
		Description: "Yüksek performanslı bellek içi veritabanı ve cache",
		Image: "redis:7-alpine", Ports: "6379:6379", Env: "",
	},
	{ID: "postgres", Name: "PostgreSQL", Icon: "🐘", Category: "Database",
		Description: "Gelişmiş açık kaynak ilişkisel veritabanı",
		Image: "postgres:16-alpine", Ports: "5432:5432",
		Env: "POSTGRES_USER=admin\nPOSTGRES_PASSWORD=changeme\nPOSTGRES_DB=mydb",
	},
	{ID: "mongo", Name: "MongoDB", Icon: "🍃", Category: "Database",
		Description: "NoSQL döküman veritabanı",
		Image: "mongo:7", Ports: "27017:27017",
		Env: "MONGO_INITDB_ROOT_USERNAME=admin\nMONGO_INITDB_ROOT_PASSWORD=changeme",
	},
	{ID: "mysql", Name: "MySQL 8", Icon: "🐬", Category: "Database",
		Description: "Ek MySQL sunucusu (ayrı container)",
		Image: "mysql:8", Ports: "3307:3306",
		Env: "MYSQL_ROOT_PASSWORD=changeme\nMYSQL_DATABASE=mydb\nMYSQL_USER=admin\nMYSQL_PASSWORD=changeme",
	},
	{ID: "phpmyadmin", Name: "phpMyAdmin", Icon: "🗄️", Category: "Tool",
		Description: "Web tabanlı MySQL/MariaDB yönetim arayüzü",
		Image: "phpmyadmin:latest", Ports: "8081:80",
		Env: "PMA_HOST=host.docker.internal\nPMA_PORT=3306",
	},
	{ID: "pgadmin", Name: "pgAdmin 4", Icon: "🐘", Category: "Tool",
		Description: "Web tabanlı PostgreSQL yönetim arayüzü",
		Image: "dpage/pgadmin4:latest", Ports: "8082:80",
		Env: "PGADMIN_DEFAULT_EMAIL=admin@admin.com\nPGADMIN_DEFAULT_PASSWORD=changeme",
	},
}

// DeployHandler Docker one-click deploy
type DeployHandler struct {
	log *logger.Logger
}

// NewDeployHandler yeni DeployHandler
func NewDeployHandler(log *logger.Logger) *DeployHandler {
	return &DeployHandler{log: log}
}

// ListTemplates şablonları listeler
func (h *DeployHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"templates": appTemplates,
		"total":     len(appTemplates),
	})
}

// GetTemplate tek şablon getirir
func (h *DeployHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	for _, t := range appTemplates {
		if t.ID == id {
			writeJSON(w, http.StatusOK, t)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Şablon bulunamadı"})
}

// Deploy şablonu konteyner olarak çalıştırır
func (h *DeployHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TemplateID  string            `json:"template_id"`
		Name        string            `json:"name"`
		CustomEnv   map[string]string `json:"custom_env,omitempty"`
		CustomPorts string            `json:"custom_ports,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	// Şablonu bul
	var template *AppTemplate
	for _, t := range appTemplates {
		if t.ID == req.TemplateID {
			template = &t
			break
		}
	}
	if template == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Şablon bulunamadı"})
		return
	}

	// Konteyner adı
	containerName := req.Name
	if containerName == "" {
		containerName = template.ID + "-" + generateRandomPass(6)
	}

	// Docker/Podman komutunu oluştur
	runtime := "docker"
	if _, err := exec.LookPath("podman"); err == nil {
		if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
			runtime = "podman"
		}
	}

	args := []string{"run", "-d", "--name", containerName}

	// Port mapping
	ports := template.Ports
	if req.CustomPorts != "" {
		ports = req.CustomPorts
	}
	if ports != "" {
		args = append(args, "-p", ports)
	}

	// Environment variables
	envs := parseEnvVars(template.Env)
	for k, v := range req.CustomEnv {
		envs[k] = v
	}
	// Zayif sabit sifreleri rastgele ile degistir (changeme, secret vb)
	for k, v := range envs {
		if v == "changeme" || v == "secret" || strings.Contains(k, "PASSWORD") && (v == "" || v == "changeme" || v == "secret") {
			envs[k] = generateRandomPass(20)
		}
	}
	for k, v := range envs {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Volume mounts
	if template.Volumes != "" {
		for _, vol := range strings.Split(template.Volumes, "\n") {
			vol = strings.TrimSpace(vol)
			if vol != "" {
				parts := strings.SplitN(vol, ":", 2)
				hostPath := parts[0]
				os.MkdirAll(hostPath, 0755)
				args = append(args, "-v", vol)
			}
		}
	}

	// Restart policy
	args = append(args, "--restart", "unless-stopped")

	// Image
	args = append(args, template.Image)

	h.log.Infow("deploy başlatılıyor", "template", template.ID, "name", containerName, "runtime", runtime)

	cmd := exec.Command(runtime, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.CombinedOutput()

	if err != nil {
		h.log.Errorw("deploy başarısız", "error", err, "stderr", stderr.String())
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error":   "Konteyner başlatılamadı: " + stderr.String(),
			"command": runtime + " " + strings.Join(args, " "),
		})
		return
	}

	containerID := strings.TrimSpace(string(out))[:12]

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":       "Konteyner başlatıldı!",
		"container_id":  containerID,
		"container_name": containerName,
		"template":      template.Name,
		"url":           getContainerURL(template, containerName),
	})
}

// getContainerURL konteyner erişim URL'sini döndürür
func getContainerURL(t *AppTemplate, name string) string {
	port := strings.Split(t.Ports, ":")[0]
	return fmt.Sprintf("http://localhost:%s", port)
}

// parseEnvVars env string'ini map'e çevirir
func parseEnvVars(env string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(env, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
