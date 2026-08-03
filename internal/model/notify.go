package model

import "time"

// Notify level keys (escalation toward due).
const (
	NotifyLevelWarning  = "warning"
	NotifyLevelCritical = "critical"
	NotifyLevelImminent = "imminent"
)

// TenantTodoNotify 租户级待办飞书通知配置（Webhook 存库，不进 YAML）。
type TenantTodoNotify struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	TenantID            uint64     `gorm:"uniqueIndex;not null" json:"tenantId"`
	Enabled             bool       `gorm:"not null;default:false" json:"enabled"`
	WebhookURL          string     `gorm:"type:text" json:"webhookUrl"`
	Secret              string     `gorm:"size:256" json:"-"`
	PollIntervalMinutes int        `gorm:"not null;default:5" json:"pollIntervalMinutes"`
	LeadMinutes         int        `gorm:"not null;default:60" json:"leadMinutes"`       // legacy；新逻辑用 LevelsJSON
	NotifyOverdue       bool       `gorm:"not null;default:false" json:"notifyOverdue"` // legacy，已停用逾期提醒
	LevelsJSON          string     `gorm:"type:text" json:"-"`                         // 等级配置 JSON
	LastRunAt           *time.Time `json:"lastRunAt"`
	LastRunOK           bool       `gorm:"not null;default:false" json:"lastRunOk"`
	LastError           string     `gorm:"type:text" json:"lastError"`
	LastSentCount       int        `gorm:"not null;default:0" json:"lastSentCount"`
	NotifiedJSON        string     `gorm:"type:text" json:"-"` // dedup map JSON
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

func (TenantTodoNotify) TableName() string { return "tenant_todo_notifies" }
