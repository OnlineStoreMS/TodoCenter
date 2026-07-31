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
	row := &model.Todo{
		TenantID:       repo.NormalizeTenantID(tenantID),
		CategoryID:     req.CategoryID,
		Title:          title,
		Description:    strings.TrimSpace(req.Description),
		Status:         status,
		Priority:       priority,
		ImagesJSON:     mustJSON(req.Images),
		AssigneeUserID: req.AssigneeUserID,
		CreatedBy:      userID,
	}
	if due, ok := parseOptionalTime(req.DueAt); ok {
		row.DueAt = due
	}
	if status == model.TodoStatusDone {
		now := time.Now()
		row.CompletedAt = &now
	}
	if err := s.repos.Todo.Create(row); err != nil {
		return nil, err
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
		s.applyStatus(row, *req.Status)
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
	_, err := s.repos.Todo.Get(tenantID, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrNotFound
		}
		return err
	}
	return s.repos.Todo.Delete(tenantID, id)
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
	return dto.TodoDTO{
		ID:             row.ID,
		CategoryID:     row.CategoryID,
		CategoryName:   cat.Name,
		CategoryCode:   cat.Code,
		Title:          row.Title,
		Description:    row.Description,
		Status:         row.Status,
		Priority:       row.Priority,
		DueAt:          formatTime(row.DueAt),
		CompletedAt:    formatTime(row.CompletedAt),
		Images:         parseImages(row.ImagesJSON),
		AssigneeUserID: row.AssigneeUserID,
		CreatedBy:      row.CreatedBy,
		CreatedAt:      row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      row.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
