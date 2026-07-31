package repo

import (
	"todocenter/internal/model"

	"gorm.io/gorm"
)

type CategoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepo(db *gorm.DB) *CategoryRepo {
	return &CategoryRepo{db: db}
}

func (r *CategoryRepo) ListAll(tenantID uint64) ([]model.TodoCategory, error) {
	var list []model.TodoCategory
	err := r.db.Scopes(scopeTenant(tenantID)).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *CategoryRepo) Get(tenantID, id uint64) (*model.TodoCategory, error) {
	var row model.TodoCategory
	err := r.db.Scopes(scopeTenant(tenantID)).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CategoryRepo) GetByCode(tenantID uint64, code string) (*model.TodoCategory, error) {
	var row model.TodoCategory
	err := r.db.Scopes(scopeTenant(tenantID)).Where("code = ?", code).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CategoryRepo) Create(row *model.TodoCategory) error {
	return r.db.Create(row).Error
}

func (r *CategoryRepo) Update(row *model.TodoCategory) error {
	return r.db.Save(row).Error
}

func (r *CategoryRepo) Delete(tenantID, id uint64) error {
	return r.db.Scopes(scopeTenant(tenantID)).Delete(&model.TodoCategory{}, id).Error
}

func (r *CategoryRepo) CountTodos(tenantID, categoryID uint64) (int64, error) {
	var n int64
	err := r.db.Model(&model.Todo{}).
		Scopes(scopeTenant(tenantID)).
		Where("category_id = ?", categoryID).
		Count(&n).Error
	return n, err
}

func (r *CategoryRepo) ExistsCode(tenantID uint64, code string, excludeID uint64) (bool, error) {
	q := r.db.Model(&model.TodoCategory{}).Scopes(scopeTenant(tenantID)).Where("code = ?", code)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *CategoryRepo) ExistsName(tenantID uint64, name string, excludeID uint64) (bool, error) {
	q := r.db.Model(&model.TodoCategory{}).Scopes(scopeTenant(tenantID)).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}
