package model

import "time"

// HostingPackage hosting paketi
type HostingPackage struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	CPUShares  int       `json:"cpu_shares"`  // 1024 = 1 core
	MemoryMB   int       `json:"memory_mb"`   // RAM MB
	Nproc      int       `json:"nproc"`       // max process
	DiskMB     int       `json:"disk_mb"`     // disk MB
	MaxDomains int       `json:"max_domains"`
	MaxEmails  int       `json:"max_emails"`
	MaxDB      int       `json:"max_db"`
	CreatedAt  time.Time `json:"created_at"`
}
