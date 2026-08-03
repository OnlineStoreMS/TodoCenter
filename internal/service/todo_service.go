package service

import (
	"encoding/json"
	"strings"
	"time"

	"todocenter/internal/dto"
	"todocenter/internal/model"
	"todocenter/internal/repo"

	"gorm.io/gorm"
)

type TodoService struct {
	repos *repo.Repos
}

func NewTodoService(repos *repo.Repos) *TodoService {
	return &TodoService{repos: repos}
}

func (s *TodoService) ensureCats(tenantID uint64) error {
	return s.repos.EnsureDefaultCategories(tenantID)
}

func (s *TodoService) DashboardStats(tenantID uint64) (*dto.DashboardStats, error) {
	_ = s.ensureCats(tenantID)
	_ = s.EnsureMonthlyInstances(tenantID)
	total, err := s.repos.Todo.CountAll(tenantID)
	if err != nil {
		return nil, err
	}
	pending, _ := s.repos.Todo.CountByStatus(tenantID, model.TodoStatusPending)
	inProgress, _ := s.repos.Todo.CountByStatus(tenantID, model.TodoStatusInProgress)
	done, _ := s.repos.Todo.CountByStatus(tenantID, model.TodoStatusDone)
	cancelled, _ := s.repos.Todo.CountByStatus(tenantID, model.TodoStatusCancelled)
	byCatID, _ := s.repos.Todo.CountGroupByCategory(tenantID)
	cats, _ := s.repos.Category.ListAll(tenantID)
	byCategory := map[string]int64{}
	for _, c := range cats {
		byCategory[c.Code] = byCatID[c.ID]
	}
	return &dto.DashboardStats{
		Total:      total,
		Pending:    pending,
		InProgress: inProgress,
		Done:       done,
		Cancelled:  cancelled,
		ByCategory: byCategory,
	}, nil
}

func (s *TodoService) ListCategories(tenantID uint64) ([]dto.CategoryDTO, error) {
	_ = s.ensureCats(tenantID)
	list, err := s.repos.Category.ListAll(tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CategoryDTO, 0, len(list))
	for _, c := range list {
		out = append(out, dto.CategoryDTO{
			ID: c.ID, Name: c.Name, Code: c.Code, Sort: c.Sort, Enabled: c.Enabled,
		})
	}
	return out, nil
}

func (s *TodoService) CreateCategory(tenantID uint64, req dto.CategoryCreateReq) (*dto.CategoryDTO, error) {
	_ = s.ensureCats(tenantID)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrBadRequest
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = "custom_" + strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	}
	exists, err := s.repos.Category.ExistsCode(tenantID, code, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConflict
	}
	enabled := 1
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &model.TodoCategory{
		TenantID: repo.NormalizeTenantID(tenantID),
		Name:     name,
		Code:     code,
		Sort:     req.Sort,
		Enabled:  enabled,
	}
	if err := s.repos.Category.Create(row); err != nil {
		return nil, err
	}
	return &dto.CategoryDTO{ID: row.ID, Name: row.Name, Code: row.Code, Sort: row.Sort, Enabled: row.Enabled}, nil
}

func (s *TodoService) UpdateCategory(tenantID, id uint64, req dto.CategoryUpdateReq) (*dto.CategoryDTO, error) {
	row, err := s.repos.Category.Get(tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if req.Name != nil {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.Sort != nil {
		row.Sort = *req.Sort
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repos.Category.Update(row); err != nil {
		return nil, err
	}
	return &dto.CategoryDTO{ID: row.ID, Name: row.Name, Code: row.Code, Sort: row.Sort, Enabled: row.Enabled}, nil
}

func (s *TodoService) DeleteCategory(tenantID, id uint64) error {
	row, err := s.repos.Category.Get(tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return err
	}
	n, err := s.repos.Category.CountTodos(tenantID, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrConflict
	}
	// protect seeded codes from delete if you want — allow for custom only
	_ = row
	return s.repos.Category.Delete(tenantID, id)
}

func (s *TodoService) ListTodos(tenantID uint64, q dto.TodoListQuery) ([]dto.TodoDTO, int64, error) {
	_ = s.ensureCats(tenantID)
	_ = s.EnsureMonthlyInstances(tenantID)
	list, total, err := s.repos.Todo.List(tenantID, q)
	if err != nil {
		return nil, 0, err
	}
	cats, _ := s.repos.Category.ListAll(tenantID)
	catMap := map[uint64]model.TodoCategory{}
	for _, c := range cats {
		catMap[c.ID] = c
	}
	out := make([]dto.TodoDTO, 0, len(list))
	for i := range list {
		out = append(out, toTodoDTO(&list[i], catMap[list[i].CategoryID]))
	}
	return out, total, nil
}

func (s *TodoService) GetTodo(tenantID, id uint64) (*dto.TodoDTO, error) {
	row, err := s.repos.Todo.Get(tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	cat, _ := s.repos.Category.Get(tenantID, row.CategoryID)
	var c model.TodoCategory
	if cat != nil {
		c = *cat
	}
	d := toTodoDTO(row, c)
	return &d, nil
}

func (s *TodoService) CreateTodo(tenantID, userID uint64, req dto.TodoCreateReq) (*dto.TodoDTO, error) {
	_ = s.ensureCats(tenantID)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrBadRequest
	}
	if _, err := s.repos.Category.Get(tenantID, req.CategoryID); err != nil {
		return nil, ErrBadRequest
	}
	status := req.Status
	if status == "" {
		status = model.TodoStatusPending
	}
	if !validStatus(status) {
		return nil, ErrBadRequest
	}
	priority := req.Priority
	if priority == "" {
		priority = model.TodoPriorityNormal
	}
	if !validPriority(priority) {
		return nil, ErrBadRequest
	}
	recurrence := normalizeRecurrence(req.Recurrence)
	recDay := normalizeRecurrenceDay(req.RecurrenceDay)

	row := &model.Todo{
		TenantID:       repo.NormalizeTenantID(tenantID),
		CategoryID:     req.CategoryID,
		Title:          title,
		Description:    strings.TrimSpace(req.Description),
		Status:         status,
		Priority:       priority,
		Recurrence:     recurrence,
		RecurrenceDay:  recDay,
		ImagesJSON:     mustJSON(req.Images),
		AssigneeUserID: req.AssigneeUserID,
		CreatedBy:      userID,
	}
	if recurrence == model.RecurrenceMonthly {
		// 模板本身不走完成态；本月实例单独生成
		row.Status = model.TodoStatusPending
		row.CompletedAt = nil
		row.DueAt = nil
	} else {
		if due, ok := parseOptionalTime(req.DueAt); ok {
			row.DueAt = due
		}
		if status == model.TodoStatusDone {
			now := time.Now()
			row.CompletedAt = &now
		}
	}
	if err := s.repos.Todo.Create(row); err != nil {
		return nil, err
	}
	if recurrence == model.RecurrenceMonthly {
		if _, err := s.materializeInstance(tenantID, userID, row, time.Now()); err != nil {
			return nil, err
		}
		// 返回本月实例，便于列表立即看到
		period := periodKeyOf(time.Now())
		inst, err := s.repos.Todo.FindInstance(tenantID, row.ID, period)
		if err == nil && inst != nil {
			return s.GetTodo(tenantID, inst.ID)
		}
	}
	return s.GetTodo(tenantID, row.ID)
}

func (s *TodoService) UpdateTodo(tenantID, id uint64, req dto.TodoUpdateReq) (*dto.TodoDTO, error) {
	row, err := s.repos.Todo.Get(tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if req.CategoryID != nil {
		if _, err := s.repos.Category.Get(tenantID, *req.CategoryID); err != nil {
			return nil, ErrBadRequest
		}
		row.CategoryID = *req.CategoryID
	}
	if req.Title != nil {
		row.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		row.Description = strings.TrimSpace(*req.Description)
	}
	if req.Priority != nil {
		if !validPriority(*req.Priority) {
			return nil, ErrBadRequest
		}
		row.Priority = *req.Priority
	}
	if req.Status != nil {
		if !validStatus(*req.Status) {
			return nil, ErrBadRequest
		}
		// 模板不改状态；改状态请操作本月实例
		if !(row.ParentID == 0 && row.Recurrence == model.RecurrenceMonthly) {
			s.applyStatus(row, *req.Status)
		}
	}
	if req.Recurrence != nil || req.RecurrenceDay != nil {
		// 仅模板或普通待办可改循环；实例跟随模板，不允许单独改循环
		if row.ParentID > 0 {
			return nil, ErrBadRequest
		}
		if req.Recurrence != nil {
			row.Recurrence = normalizeRecurrence(*req.Recurrence)
		}
		if req.RecurrenceDay != nil {
			row.RecurrenceDay = normalizeRecurrenceDay(*req.RecurrenceDay)
		}
		if row.Recurrence == model.RecurrenceNone {
			row.RecurrenceDay = 1
		}
	}
	if req.ClearDueAt {
		row.DueAt = nil
	} else if req.DueAt != nil {
		due, ok := parseOptionalTime(*req.DueAt)
		if !ok && strings.TrimSpace(*req.DueAt) != "" {
			return nil, ErrBadRequest
		}
		row.DueAt = due
	}
	if req.Images != nil {
		row.ImagesJSON = mustJSON(req.Images)
	}
	if req.AssigneeUserID != nil {
		row.AssigneeUserID = *req.AssigneeUserID
	}
	if err := s.repos.Todo.Update(row); err != nil {
		return nil, err
	}
	// 升级为月待办后补生成本月实例
	if row.ParentID == 0 && row.Recurrence == model.RecurrenceMonthly {
		_, _ = s.materializeInstance(tenantID, row.CreatedBy, row, time.Now())
	}
	return s.GetTodo(tenantID, id)
}

func (s *TodoService) UpdateTodoStatus(tenantID, id uint64, status string) (*dto.TodoDTO, error) {
	if !validStatus(status) {
		return nil, ErrBadRequest
	}
	row, err := s.repos.Todo.Get(tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.applyStatus(row, status)
	if err := s.repos.Todo.Update(row); err != nil {
		return nil, err
	}
	return s.GetTodo(tenantID, id)
}

func (s *TodoService) DeleteTodo(tenantID, id uint64) error {
	row, err := s.repos.Todo.Get(tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return err
	}
	return s.repos.Todo.Delete(tenantID, row.ID)
}

// EnsureMonthlyInstances 为所有月待办模板补齐当前账期实例。
func (s *TodoService) EnsureMonthlyInstances(tenantID uint64) error {
	templates, err := s.repos.Todo.ListMonthlyTemplates(tenantID)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range templates {
		if _, err := s.materializeInstance(tenantID, templates[i].CreatedBy, &templates[i], now); err != nil {
			return err
		}
	}
	return nil
}

func (s *TodoService) materializeInstance(tenantID, userID uint64, tpl *model.Todo, at time.Time) (*model.Todo, error) {
	if tpl == nil || tpl.ID == 0 || tpl.Recurrence != model.RecurrenceMonthly {
		return nil, nil
	}
	period := periodKeyOf(at)
	if existing, err := s.repos.Todo.FindInstance(tenantID, tpl.ID, period); err == nil && existing != nil {
		return existing, nil
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	due := dueAtForMonth(at, tpl.RecurrenceDay)
	inst := &model.Todo{
		TenantID:       repo.NormalizeTenantID(tenantID),
		CategoryID:     tpl.CategoryID,
		Title:          tpl.Title,
		Description:    tpl.Description,
		Status:         model.TodoStatusPending,
		Priority:       tpl.Priority,
		Recurrence:     model.RecurrenceNone, // 实例本身不再循环
		RecurrenceDay:  tpl.RecurrenceDay,
		ParentID:       tpl.ID,
		PeriodKey:      period,
		DueAt:          &due,
		ImagesJSON:     tpl.ImagesJSON,
		AssigneeUserID: tpl.AssigneeUserID,
		CreatedBy:      userID,
	}
	if err := s.repos.Todo.Create(inst); err != nil {
		// 并发下可能撞唯一索引，再查一次
		if existing, e2 := s.repos.Todo.FindInstance(tenantID, tpl.ID, period); e2 == nil {
			return existing, nil
		}
		return nil, err
	}
	return inst, nil
}

func (s *TodoService) applyStatus(row *model.Todo, status string) {
	row.Status = status
	if status == model.TodoStatusDone {
		now := time.Now()
		row.CompletedAt = &now
	} else {
		row.CompletedAt = nil
	}
}

func validStatus(s string) bool {
	switch s {
	case model.TodoStatusPending, model.TodoStatusInProgress, model.TodoStatusDone, model.TodoStatusCancelled:
		return true
	default:
		return false
	}
}

func validPriority(p string) bool {
	switch p {
	case model.TodoPriorityLow, model.TodoPriorityNormal, model.TodoPriorityHigh:
		return true
	default:
		return false
	}
}

func normalizeRecurrence(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case model.RecurrenceMonthly:
		return model.RecurrenceMonthly
	default:
		return model.RecurrenceNone
	}
}

func normalizeRecurrenceDay(d int) int {
	if d < 1 {
		return 1
	}
	if d > 28 {
		return 28
	}
	return d
}

func periodKeyOf(t time.Time) string {
	return t.Format("2006-01")
}

func dueAtForMonth(at time.Time, day int) time.Time {
	day = normalizeRecurrenceDay(day)
	y, m, _ := at.Date()
	loc := at.Location()
	// 取当月最后一天，避免 2 月越界（已限制 1-28）
	return time.Date(y, m, day, 23, 59, 59, 0, loc)
}

func mustJSON(images []dto.MediaItem) string {
	if images == nil {
		images = []dto.MediaItem{}
	}
	b, _ := json.Marshal(images)
	return string(b)
}

func parseImages(raw string) []dto.MediaItem {
	if strings.TrimSpace(raw) == "" {
		return []dto.MediaItem{}
	}
	var out []dto.MediaItem
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []dto.MediaItem{}
	}
	return out
}

func parseOptionalTime(s string) (*time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, true
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, true
		}
	}
	return nil, false
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func toTodoDTO(row *model.Todo, cat model.TodoCategory) dto.TodoDTO {
	rec := row.Recurrence
	if rec == "" {
		rec = model.RecurrenceNone
	}
	isTpl := row.ParentID == 0 && rec == model.RecurrenceMonthly
	return dto.TodoDTO{
		ID:             row.ID,
		CategoryID:     row.CategoryID,
		CategoryName:   cat.Name,
		CategoryCode:   cat.Code,
		Title:          row.Title,
		Description:    row.Description,
		Status:         row.Status,
		Priority:       row.Priority,
		Recurrence:     rec,
		RecurrenceDay:  row.RecurrenceDay,
		ParentID:       row.ParentID,
		PeriodKey:      row.PeriodKey,
		IsTemplate:     isTpl,
		IsMonthlyInst:  row.ParentID > 0,
		DueAt:          formatTime(row.DueAt),
		CompletedAt:    formatTime(row.CompletedAt),
		Images:         parseImages(row.ImagesJSON),
		AssigneeUserID: row.AssigneeUserID,
		CreatedBy:      row.CreatedBy,
		CreatedAt:      row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      row.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
