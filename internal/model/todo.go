package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	TodoStatusPending    = "pending"
	TodoStatusInProgress = "in_progress"
	TodoStatusDone       = "done"
	TodoStatusCancelled  = "cancelled"

	TodoPriorityLow    = "low"
	TodoPriorityNormal = "normal"
	TodoPriorityHigh   = "high"

	// RecurrenceNone 普通待办；RecurrenceMonthly 固定月待办（模板，每月生成实例）
	RecurrenceNone    = "none"
	RecurrenceMonthly = "monthly"
)

// TodoCategory 待办分类（电商/发货/售后/门店等）
type TodoCategory struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	TenantID  uint64         `gorm:"not null;uniqueIndex:uk_todo_cat_tenant_code,priority:1;index" json:"tenantId"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Code      string         `gorm:"size:64;not null;uniqueIndex:uk_todo_cat_tenant_code,priority:2" json:"code"`
	Sort      int            `gorm:"not null;default:0" json:"sort"`
	Enabled   int            `gorm:"not null;default:1" json:"enabled"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TodoCategory) TableName() string { return "todo_categories" }

// Todo 待办事项。
// 月待办：recurrence=monthly 且 parent_id=0 为模板；每月生成 parent_id=模板ID、period_key=YYYY-MM 的实例。
type Todo struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	TenantID       uint64         `gorm:"not null;index:idx_todo_tenant_status,priority:1;index:idx_todo_tenant_cat,priority:1;index:idx_todo_parent_period,priority:1" json:"tenantId"`
	CategoryID     uint64         `gorm:"not null;default:0;index:idx_todo_tenant_cat,priority:2" json:"categoryId"`
	Title          string         `gorm:"size:256;not null" json:"title"`
	Description    string         `gorm:"type:text" json:"description"`
	Status         string         `gorm:"size:32;not null;default:pending;index:idx_todo_tenant_status,priority:2" json:"status"`
	Priority       string         `gorm:"size:16;not null;default:normal" json:"priority"`
	Recurrence     string         `gorm:"size:16;not null;default:none;index" json:"recurrence"` // none | monthly
	RecurrenceDay  int            `gorm:"not null;default:1" json:"recurrenceDay"`                // 月待办：每月几号 1-28
	ParentID       uint64         `gorm:"not null;default:0;index;index:idx_todo_parent_period,priority:2" json:"parentId"`
	PeriodKey      string         `gorm:"size:16;not null;default:'';index:idx_todo_parent_period,priority:3" json:"periodKey"` // 实例：2026-07
	DueAt          *time.Time     `json:"dueAt"`
	CompletedAt    *time.Time     `json:"completedAt"`
	ImagesJSON     string         `gorm:"type:text" json:"imagesJson"` // JSON MediaItem[]
	AssigneeUserID uint64         `gorm:"not null;default:0" json:"assigneeUserId"`
	CreatedBy      uint64         `gorm:"not null;default:0" json:"createdBy"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Todo) TableName() string { return "todos" }
