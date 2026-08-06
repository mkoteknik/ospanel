package container

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// DockerClient Docker/Podman yönetim istemcisi
type DockerClient struct {
	socket string
	http   *http.Client
}

// Container bilgisi
type Container struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Status  string   `json:"status"`
	State   string   `json:"state"`
	Ports   []string `json:"ports"`
	Created string   `json:"created"`
}

// NewDockerClient yeni Docker client oluşturur
func NewDockerClient() *DockerClient {
	// Docker veya Podman socket'ini bul
	socket := "/run/podman/podman.sock"
	if _, err := os.Stat(socket); os.IsNotExist(err) {
		socket = "/var/run/docker.sock"
		if _, err := os.Stat(socket); os.IsNotExist(err) {
			home := os.Getenv("HOME")
			if home != "" {
				podmanSocket := home + "/.local/share/containers/podman/machine/podman.sock"
				if _, err := os.Stat(podmanSocket); err == nil {
					socket = podmanSocket
				}
			}
		}
	}

	return &DockerClient{
		socket: socket,
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socket)
				},
			},
			Timeout: 10 * time.Second,
		},
	}
}

// IsAvailable Docker/Podman kullanılabilir mi?
func (d *DockerClient) IsAvailable() bool {
	_, err := os.Stat(d.socket)
	return err == nil
}

// GetSocketPath socket yolunu döndürür
func (d *DockerClient) GetSocketPath() string {
	return d.socket
}

// ListContainers tüm konteynerleri listeler
func (d *DockerClient) ListContainers() ([]Container, error) {
	if !d.IsAvailable() {
		return nil, fmt.Errorf("Docker/Podman kullanılabilir değil")
	}

	resp, err := d.http.Get("http://localhost/containers/json?all=true")
	if err != nil {
		return nil, fmt.Errorf("konteyner listesi alınamadı: %w", err)
	}
	defer resp.Body.Close()

	var rawContainers []struct {
		ID      string   `json:"Id"`
		Names   []string `json:"Names"`
		Image   string   `json:"Image"`
		Status  string   `json:"Status"`
		State   string   `json:"State"`
		Ports   []struct {
			PublicPort int    `json:"PublicPort"`
			Type       string `json:"Type"`
		} `json:"Ports"`
		Created int64 `json:"Created"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawContainers); err != nil {
		return nil, err
	}

	var containers []Container
	for _, c := range rawContainers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		var ports []string
		for _, p := range c.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.PublicPort, p.Type))
		}
		containers = append(containers, Container{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Ports:   ports,
			Created: time.Unix(c.Created, 0).Format("2006-01-02 15:04"),
		})
	}

	return containers, nil
}

// GetStats konteyner istatistikleri
func (d *DockerClient) GetStats() map[string]interface{} {
	if !d.IsAvailable() {
		return map[string]interface{}{"installed": false}
	}

	containers, err := d.ListContainers()
	running := 0
	for _, c := range containers {
		if c.State == "running" {
			running++
		}
	}

	return map[string]interface{}{
		"installed":  true,
		"socket":     d.socket,
		"total":      len(containers),
		"running":    running,
		"containers": containers,
		"error":      fmt.Sprint(err),
	}
}

// StartContainer konteyner başlatır
func (d *DockerClient) StartContainer(id string) error {
	req, _ := http.NewRequest("POST", "http://localhost/containers/"+id+"/start", nil)
	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// StopContainer konteyner durdurur
func (d *DockerClient) StopContainer(id string) error {
	req, _ := http.NewRequest("POST", "http://localhost/containers/"+id+"/stop", nil)
	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// RestartContainer konteyner yeniden başlatır
func (d *DockerClient) RestartContainer(id string) error {
	req, _ := http.NewRequest("POST", "http://localhost/containers/"+id+"/restart", nil)
	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
