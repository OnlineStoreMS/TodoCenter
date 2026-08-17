package repo

import (
	"strings"
	"time"

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
	switch strings.TrimSpace(q.RecurrenceFilter) {
	case "none":
		// 普通待办（非模板、非实例）
		dbq = dbq.Where("parent_id = 0 AND recurrence = ?", model.RecurrenceNone)
	case "monthly", "templates":
		// 固定月待办模板
		dbq = dbq.Where("parent_id = 0 AND recurrence = ?", model.RecurrenceMonthly)
	case "instances":
		dbq = dbq.Where("parent_id > 0")
	default:
		// 默认：普通待办 + 月实例（隐藏模板，避免与本月实例重复）
		dbq = dbq.Where("parent_id > 0 OR recurrence = ?", model.RecurrenceNone)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Todo
	err := dbq.Order(todoListOrderClause(q)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func todoListOrderClause(q dto.TodoListQuery) string {
	order := strings.ToLower(strings.TrimSpace(q.SortOrder))
	if order != "asc" && order != "desc" {
		order = ""
	}
	by := strings.TrimSpace(q.SortBy)
	switch by {
	case "category", "categoryName", "categoryId":
		if order == "" {
			order = "asc"
		}
		return "category_id " + order + ", id DESC"
	case "priority":
		if order == "" {
			order = "desc"
		}
		// high > normal > low
		if order == "desc" {
			return "CASE priority WHEN 'high' THEN 0 WHEN 'normal' THEN 1 WHEN 'low' THEN 2 ELSE 3 END ASC, id DESC"
		}
		return "CASE priority WHEN 'low' THEN 0 WHEN 'normal' THEN 1 WHEN 'high' THEN 2 ELSE 3 END ASC, id DESC"
	case "status":
		if order == "" {
			order = "asc"
		}
		if order == "asc" {
			return "CASE status WHEN 'pending' THEN 0 WHEN 'in_progress' THEN 1 WHEN 'done' THEN 2 ELSE 3 END ASC, id DESC"
		}
		return "CASE status WHEN 'pending' THEN 0 WHEN 'in_progress' THEN 1 WHEN 'done' THEN 2 ELSE 3 END DESC, id DESC"
	case "dueAt", "due_at":
		if order == "" {
			order = "asc"
		}
		// NULLS LAST for both directions via CASE
		return "CASE WHEN due_at IS NULL THEN 1 ELSE 0 END ASC, due_at " + order + ", id DESC"
	case "updatedAt", "updated_at":
		if order == "" {
			order = "desc"
		}
		return "updated_at " + order + ", id DESC"
	case "title":
		if order == "" {
			order = "asc"
		}
		return "title " + order + ", id DESC"
	case "id":
		if order == "" {
			order = "desc"
		}
		return "id " + order
	default:
		// 默认：待处理优先 → 优先级高 → 新创建靠前
		return "CASE status WHEN 'pending' THEN 0 WHEN 'in_progress' THEN 1 WHEN 'done' THEN 2 ELSE 3 END, CASE priority WHEN 'high' THEN 0 WHEN 'normal' THEN 1 WHEN 'low' THEN 2 ELSE 3 END, id DESC"
	}
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

func (r *TodoRepo) ListMonthlyTemplates(tenantID uint64) ([]model.Todo, error) {
	var list []model.Todo
	err := r.db.Scopes(scopeTenant(tenantID)).
		Where("parent_id = 0 AND recurrence = ?", model.RecurrenceMonthly).
		Find(&list).Error
	return list, err
}

func (r *TodoRepo) FindInstance(tenantID, parentID uint64, periodKey string) (*model.Todo, error) {
	var row model.Todo
	err := r.db.Scopes(scopeTenant(tenantID)).
		Where("parent_id = ? AND period_key = ?", parentID, periodKey).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *TodoRepo) CountByStatus(tenantID uint64, status string) (int64, error) {
	var n int64
	// 统计默认列表口径：普通 + 实例
	err := r.db.Model(&model.Todo{}).Scopes(scopeTenant(tenantID)).
		Where("(parent_id > 0 OR recurrence = ?) AND status = ?", model.RecurrenceNone, status).
		Count(&n).Error
	return n, err
}

func (r *TodoRepo) CountAll(tenantID uint64) (int64, error) {
	var n int64
	err := r.db.Model(&model.Todo{}).Scopes(scopeTenant(tenantID)).
		Where("parent_id > 0 OR recurrence = ?", model.RecurrenceNone).
		Count(&n).Error
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
		Where("parent_id > 0 OR recurrence = ?", model.RecurrenceNone).
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

// ListDueForNotify 返回需提醒的待办：普通待办 + 月实例（排除模板），状态 pending/in_progress，有截止时间。
func (r *TodoRepo) ListDueForNotify(tenantID uint64, until time.Time) ([]model.Todo, error) {
	var list []model.Todo
	err := r.db.Scopes(scopeTenant(tenantID)).
		Where("parent_id > 0 OR recurrence = ?", model.RecurrenceNone).
		Where("status IN ?", []string{model.TodoStatusPending, model.TodoStatusInProgress}).
		Where("due_at IS NOT NULL AND due_at <= ?", until).
		Order("due_at ASC, id ASC").
		Limit(200).
		Find(&list).Error
	return list, err
}
