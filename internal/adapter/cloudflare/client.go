package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client CloudFlare API v4 istemcisi
type Client struct {
	apiKey   string
	email    string
	zoneID   string
	http     *http.Client
}

// DNSRecord CloudFlare DNS kaydı
type CFDNSRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// Zone CloudFlare zone
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CF API yanıt yapısı
type cfResponse struct {
	Success  bool             `json:"success"`
	Errors   []json.RawMessage `json:"errors"`
	Messages []string          `json:"messages"`
	Result   json.RawMessage   `json:"result"`
}

// NewClient yeni CloudFlare client
func NewClient() *Client {
	apiKey := os.Getenv("CF_API_KEY")
	email := os.Getenv("CF_EMAIL")
	zoneID := os.Getenv("CF_ZONE_ID")

	// Env yoksa config dosyasından dene
	if apiKey == "" {
		data, _ := os.ReadFile("/etc/ospanel/cf.conf")
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				switch strings.TrimSpace(parts[0]) {
				case "CF_API_KEY":
					apiKey = strings.TrimSpace(parts[1])
				case "CF_EMAIL":
					email = strings.TrimSpace(parts[1])
				case "CF_ZONE_ID":
					zoneID = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return &Client{
		apiKey: apiKey,
		email:  email,
		zoneID: zoneID,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Configure CloudFlare bilgilerini kaydeder
func (c *Client) Configure(email, apiKey, zoneID string) error {
	c.email = email
	c.apiKey = apiKey
	c.zoneID = zoneID

	cfg := fmt.Sprintf("CF_EMAIL=%s\nCF_API_KEY=%s\nCF_ZONE_ID=%s\n", email, apiKey, zoneID)
	os.MkdirAll("/etc/ospanel", 0755)
	return os.WriteFile("/etc/ospanel/cf.conf", []byte(cfg), 0600)
}

// IsConfigured CloudFlare yapılandırılmış mı?
func (c *Client) IsConfigured() bool {
	return c.apiKey != "" && c.email != ""
}

// SetZoneID zone ID ayarla
func (c *Client) SetZoneID(id string) { c.zoneID = id }

// GetZoneInfo zone bilgisi
func (c *Client) GetZoneInfo() (map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/zones/"+c.zoneID, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(resp, &result)
	return result, nil
}

// GetDomainZone domain için zone ID bulur
func (c *Client) GetDomainZone(domain string) (string, error) {
	params := url.Values{}
	params.Set("name", domain)
	resp, err := c.doRequest("GET", "/zones?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}

	var zones []map[string]interface{}
	if err := json.Unmarshal(resp, &zones); err != nil {
		return "", err
	}

	if len(zones) == 0 {
		return "", fmt.Errorf("zone bulunamadı: %s", domain)
	}

	return fmt.Sprint(zones[0]["id"]), nil
}

// ListDNSRecords DNS kayıtlarını listeler
func (c *Client) ListDNSRecords() ([]CFDNSRecord, error) {
	resp, err := c.doRequest("GET", "/zones/"+c.zoneID+"/dns_records?per_page=500", nil)
	if err != nil {
		return nil, err
	}

	var records []CFDNSRecord
	if err := json.Unmarshal(resp, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// CreateDNSRecord DNS kaydı oluşturur
func (c *Client) CreateDNSRecord(record CFDNSRecord) (*CFDNSRecord, error) {
	body, _ := json.Marshal(record)
	resp, err := c.doRequest("POST", "/zones/"+c.zoneID+"/dns_records", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var r CFDNSRecord
	json.Unmarshal(resp, &r)
	return &r, nil
}

// UpdateDNSRecord DNS kaydı günceller
func (c *Client) UpdateDNSRecord(recordID string, record CFDNSRecord) error {
	body, _ := json.Marshal(record)
	_, err := c.doRequest("PUT", "/zones/"+c.zoneID+"/dns_records/"+recordID, bytes.NewReader(body))
	return err
}

// DeleteDNSRecord DNS kaydı siler
func (c *Client) DeleteDNSRecord(recordID string) error {
	_, err := c.doRequest("DELETE", "/zones/"+c.zoneID+"/dns_records/"+recordID, nil)
	return err
}

// PurgeCache tüm cache'i temizler
func (c *Client) PurgeCache() error {
	data := `{"purge_everything":true}`
	_, err := c.doRequest("POST", "/zones/"+c.zoneID+"/purge_cache", bytes.NewReader([]byte(data)))
	return err
}

// PurgeURL belirli URL'leri temizler
func (c *Client) PurgeURL(urls []string) error {
	data, _ := json.Marshal(map[string][]string{"files": urls})
	_, err := c.doRequest("POST", "/zones/"+c.zoneID+"/purge_cache", bytes.NewReader(data))
	return err
}

// GetSSLSettings SSL/TLS ayarlarını döndürür
func (c *Client) GetSSLSettings() (map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/zones/"+c.zoneID+"/settings/ssl", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(resp, &result)
	return result, nil
}

// SetSSLMode SSL modunu değiştirir (off, flexible, full, strict)
func (c *Client) SetSSLMode(mode string) error {
	data, _ := json.Marshal(map[string]string{"value": mode})
	_, err := c.doRequest("PATCH", "/zones/"+c.zoneID+"/settings/ssl", bytes.NewReader(data))
	return err
}

// GetAnalytics trafik analitiği (son 7 gün)
func (c *Client) GetAnalytics() (map[string]interface{}, error) {
	since := time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	resp, err := c.doRequest("GET", "/zones/"+c.zoneID+"/analytics/dashboard?since="+since+"&continuous=true", nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	json.Unmarshal(resp, &result)
	return result, nil
}

// GetFirewallRules güvenlik duvarı kuralları
func (c *Client) GetFirewallRules() ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/zones/"+c.zoneID+"/firewall/rules?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	var rules []map[string]interface{}
	json.Unmarshal(resp, &rules)
	return rules, nil
}

// CreateFirewallRule IP bloklama kuralı
func (c *Client) CreateFirewallRule(ip, note string) error {
	rule := map[string]interface{}{
		"action": "block",
		"filter": map[string]interface{}{
			"expression": fmt.Sprintf(`ip.src eq "%s"`, ip),
			"description": note,
		},
	}
	body, _ := json.Marshal([]map[string]interface{}{rule})
	_, err := c.doRequest("POST", "/zones/"+c.zoneID+"/firewall/rules", bytes.NewReader(body))
	return err
}

// GetStats özet bilgiler
func (c *Client) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"configured": c.IsConfigured(),
		"zone_id":    c.zoneID,
	}

	if !c.IsConfigured() {
		return stats
	}

	zone, err := c.GetZoneInfo()
	stats["zone"] = zone
	stats["error"] = fmt.Sprint(err)

	// DNS kayıt sayısı
	records, err := c.ListDNSRecords()
	stats["dns_count"] = len(records)
	if err != nil {
		stats["dns_error"] = err.Error()
	}

	return stats
}

// doRequest CF API isteği yapar
func (c *Client) doRequest(method, path string, body io.Reader) (json.RawMessage, error) {
	url := "https://api.cloudflare.com/client/v4" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Email", c.email)
	req.Header.Set("X-Auth-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cfResp cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, err
	}

	if !cfResp.Success {
		errs, _ := json.Marshal(cfResp.Errors)
		return nil, fmt.Errorf("CF API hatası: %s", string(errs))
	}

	return cfResp.Result, nil
}
