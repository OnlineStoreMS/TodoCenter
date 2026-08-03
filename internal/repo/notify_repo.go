package repo

import (
	"time"

	"todocenter/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotifyRepo struct {
	db *gorm.DB
}

func NewNotifyRepo(db *gorm.DB) *NotifyRepo {
	return &NotifyRepo{db: db}
}

func (r *NotifyRepo) GetOrCreate(tenantID uint64) (*model.TenantTodoNotify, error) {
	tenantID = NormalizeTenantID(tenantID)
	var row model.TenantTodoNotify
	err := r.db.Where("tenant_id = ?", tenantID).First(&row).Error
	if err == nil {
		return &row, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	row = model.TenantTodoNotify{
		TenantID:            tenantID,
		Enabled:             false,
		PollIntervalMinutes: 5,
		LeadMinutes:         24 * 60,
		NotifyOverdue:       false,
	}
	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return nil, err
	}
	if err := r.db.Where("tenant_id = ?", tenantID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *NotifyRepo) Save(row *model.TenantTodoNotify) error {
	return r.db.Save(row).Error
}

func (r *NotifyRepo) ListEnabled() ([]model.TenantTodoNotify, error) {
	var list []model.TenantTodoNotify
	err := r.db.Where("enabled = ? AND webhook_url <> ''", true).Find(&list).Error
	return list, err
}

func (r *NotifyRepo) UpdateRunState(tenantID uint64, ok bool, errMsg string, sent int) error {
	now := time.Now()
	return r.db.Model(&model.TenantTodoNotify{}).
		Where("tenant_id = ?", NormalizeTenantID(tenantID)).
		Updates(map[string]any{
			"last_run_at":     now,
			"last_run_ok":     ok,
			"last_error":      errMsg,
			"last_sent_count": sent,
		}).Error
}
