package repo

import (
	"strings"

	"todocenter/internal/dto"
	"todocenter/internal/model"

	"gorm.io/gorm"
)

type TodoRepo struct {
	db *gorm.DB
}

func NewTodoRepo(db *gorm.DB) *TodoRepo {
	return &TodoRepo{db: db}
}

func (r *TodoRepo) List(tenantID uint64, q dto.TodoListQuery) ([]model.Todo, int64, error) {
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	dbq := r.db.Model(&model.Todo{}).Scopes(scopeTenant(tenantID))
	if q.CategoryID > 0 {
		dbq = dbq.Where("category_id = ?", q.CategoryID)
	}
	if s := strings.TrimSpace(q.Status); s != "" {
		dbq = dbq.Where("status = ?", s)
	}
	if p := strings.TrimSpace(q.Priority); p != "" {
		dbq = dbq.Where("priority = ?", p)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		if r.db.Dialector.Name() == "postgres" {
			dbq = dbq.Where("title ILIKE ? OR description ILIKE ?", like, like)
		} else {
			dbq = dbq.Where("title LIKE ? OR description LIKE ?", like, like)
		}
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Todo
	err := dbq.Order("CASE status WHEN 'pending' THEN 0 WHEN 'in_progress' THEN 1 WHEN 'done' THEN 2 ELSE 3 END, priority DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *TodoRepo) Get(tenantID, id uint64) (*model.Todo, error) {
	var row model.Todo
	err := r.db.Scopes(scopeTenant(tenantID)).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *TodoRepo) Create(row *model.Todo) error {
	return r.db.Create(row).Error
}

func (r *TodoRepo) Update(row *model.Todo) error {
	return r.db.Save(row).Error
}

func (r *TodoRepo) Delete(tenantID, id uint64) error {
	return r.db.Scopes(scopeTenant(tenantID)).Delete(&model.Todo{}, id).Error
}

func (r *TodoRepo) CountByStatus(tenantID uint64, status string) (int64, error) {
	var n int64
	err := r.db.Model(&model.Todo{}).Scopes(scopeTenant(tenantID)).Where("status = ?", status).Count(&n).Error
	return n, err
}

func (r *TodoRepo) CountAll(tenantID uint64) (int64, error) {
	var n int64
	err := r.db.Model(&model.Todo{}).Scopes(scopeTenant(tenantID)).Count(&n).Error
	return n, err
}

func (r *TodoRepo) CountGroupByCategory(tenantID uint64) (map[uint64]int64, error) {
	type row struct {
		CategoryID uint64
		Cnt        int64
	}
	var rows []row
	err := r.db.Model(&model.Todo{}).
		Scopes(scopeTenant(tenantID)).
		Select("category_id as category_id, count(*) as cnt").
		Group("category_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint64]int64, len(rows))
	for _, x := range rows {
		out[x.CategoryID] = x.Cnt
	}
	return out, nil
}
