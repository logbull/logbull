package projects_models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Project struct {
	ID        uuid.UUID `json:"id"        gorm:"column:id"`
	Name      string    `json:"name"      gorm:"column:name"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`

	// Security Policies
	IsApiKeyRequired  bool     `json:"isApiKeyRequired" gorm:"column:is_api_key_required"`
	IsFilterByDomain  bool     `json:"isFilterByDomain" gorm:"column:is_filter_by_domain"`
	IsFilterByIP      bool     `json:"isFilterByIp"     gorm:"column:is_filter_by_ip"`
	AllowedDomainsRaw string   `json:"-"                gorm:"column:allowed_domains_raw"`
	AllowedDomains    []string `json:"allowedDomains"   gorm:"-"`
	AllowedIPsRaw     string   `json:"-"                gorm:"column:allowed_ips_raw"`
	AllowedIPs        []string `json:"allowedIps"       gorm:"-"`

	// Rate Limiting & Quotas
	LogsPerSecondLimit int   `json:"logsPerSecondLimit" gorm:"column:logs_per_second_limit"`
	MaxLogsAmount      int64 `json:"maxLogsAmount"      gorm:"column:max_logs_amount"`
	MaxLogsSizeMB      int   `json:"maxLogsSizeMb"      gorm:"column:max_logs_size_mb"`
	MaxLogsLifeDays    int   `json:"maxLogsLifeDays"    gorm:"column:max_logs_life_days"`
	MaxLogSizeKB       int   `json:"maxLogSizeKb"       gorm:"column:max_log_size_kb"`

	// Cache-related fields for logs insertion
	IsNotExists bool `json:"isNotExists,omitempty" gorm:"-"` // Used for caching non-existent projects
}

func (Project) TableName() string {
	return "projects"
}

func (p *Project) BeforeSave(tx *gorm.DB) error {
	if len(p.AllowedDomains) > 0 {
		p.AllowedDomainsRaw = strings.Join(p.AllowedDomains, ",")
	} else {
		p.AllowedDomainsRaw = ""
	}

	if len(p.AllowedIPs) > 0 {
		p.AllowedIPsRaw = strings.Join(p.AllowedIPs, ",")
	} else {
		p.AllowedIPsRaw = ""
	}

	return nil
}

func (p *Project) AfterFind(tx *gorm.DB) error {
	if p.AllowedDomainsRaw != "" {
		p.AllowedDomains = strings.Split(p.AllowedDomainsRaw, ",")
		for i, domain := range p.AllowedDomains {
			p.AllowedDomains[i] = strings.TrimSpace(domain)
		}
	} else {
		p.AllowedDomains = []string{}
	}

	if p.AllowedIPsRaw != "" {
		p.AllowedIPs = strings.Split(p.AllowedIPsRaw, ",")
		for i, ip := range p.AllowedIPs {
			p.AllowedIPs[i] = strings.TrimSpace(ip)
		}
	} else {
		p.AllowedIPs = []string{}
	}

	return nil
}
