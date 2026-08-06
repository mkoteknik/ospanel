package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Client PowerDNS API istemcisi
type Client struct {
	apiURL string
	apiKey string
	http   *http.Client
}

// Zone DNS zone
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Serial int    `json:"serial"`
}

// Record DNS kaydı
type Record struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Prio    int    `json:"priority,omitempty"`
	Disabled bool  `json:"disabled"`
}

// NewClient yeni PowerDNS client oluşturur
func NewClient() *Client {
	apiURL := "http://127.0.0.1:8081/api/v1"
	apiKey := ""

	// API key'i dosyadan oku
	if data, err := os.ReadFile("/etc/ospanel/pdns_api_key"); err == nil {
		apiKey = strings.TrimSpace(string(data))
	}

	return &Client{
		apiURL: apiURL,
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

// IsAvailable PowerDNS API kullanılabilir mi?
func (c *Client) IsAvailable() bool {
	if c.apiKey == "" {
		return false
	}
	resp, err := c.doRequest("GET", "/servers/localhost", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// ListZones tüm zone'ları listeler
func (c *Client) ListZones() ([]Zone, error) {
	var zones []Zone
	resp, err := c.doRequest("GET", "/servers/localhost/zones", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&zones); err != nil {
		return nil, err
	}
	return zones, nil
}

// CreateZone yeni zone oluşturur
func (c *Client) CreateZone(domain string) error {
	zone := Zone{
		Name: domain + ".",
		Kind: "Native",
	}

	body, _ := json.Marshal(zone)
	resp, err := c.doRequest("POST", "/servers/localhost/zones", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zone oluşturulamadı: %s - %s", resp.Status, string(respBody))
	}
	return nil
}

// DeleteZone zone siler
func (c *Client) DeleteZone(domain string) error {
	resp, err := c.doRequest("DELETE", "/servers/localhost/zones/"+domain+".", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ListRecords zone'a ait kayıtları listeler
func (c *Client) ListRecords(domain string) ([]Record, error) {
	var records []Record
	resp, err := c.doRequest("GET", "/servers/localhost/zones/"+domain+".", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return []Record{}, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}
	return records, nil
}

// CreateRecord DNS kaydı oluşturur
func (c *Client) CreateRecord(domain string, record Record) error {
	rrSet := struct {
		RRSets []struct {
			Name       string   `json:"name"`
			Type       string   `json:"type"`
			TTL        int      `json:"ttl"`
			ChangeType string   `json:"changetype"`
			Records    []Record `json:"records"`
		} `json:"rrsets"`
	}{
		RRSets: []struct {
			Name       string   `json:"name"`
			Type       string   `json:"type"`
			TTL        int      `json:"ttl"`
			ChangeType string   `json:"changetype"`
			Records    []Record `json:"records"`
		}{
			{
				Name:       record.Name + "." + domain + ".",
				Type:       record.Type,
				TTL:        record.TTL,
				ChangeType: "REPLACE",
				Records:    []Record{record},
			},
		},
	}

	body, _ := json.Marshal(rrSet)
	resp, err := c.doRequest("PATCH", "/servers/localhost/zones/"+domain+".", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("kayıt oluşturulamadı: %s - %s", resp.Status, string(respBody))
	}
	return nil
}

// GetStats PowerDNS istatistikleri
func (c *Client) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"installed": c.IsAvailable(),
		"api_url":   c.apiURL,
	}

	if !c.IsAvailable() {
		return stats
	}

	zones, err := c.ListZones()
	stats["zone_count"] = len(zones)
	stats["error"] = fmt.Sprint(err)

	return stats
}

// doRequest HTTP isteği yapar
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := c.apiURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	return c.http.Do(req)
}
