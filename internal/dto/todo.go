package dto

type CategoryDTO struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Sort    int    `json:"sort"`
	Enabled int    `json:"enabled"`
}

type CategoryCreateReq struct {
	Name    string `json:"name" binding:"required"`
	Code    string `json:"code"`
	Sort    int    `json:"sort"`
	Enabled *int   `json:"enabled"`
}

type CategoryUpdateReq struct {
	Name    *string `json:"name"`
	Sort    *int    `json:"sort"`
	Enabled *int    `json:"enabled"`
}

type MediaItem struct {
	URL       string `json:"url"`
	MediaType string `json:"mediaType"`
	FileName  string `json:"fileName,omitempty"`
	Mime      string `json:"mime,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
}

type TodoDTO struct {
	ID             uint64      `json:"id"`
	CategoryID     uint64      `json:"categoryId"`
	CategoryName   string      `json:"categoryName,omitempty"`
	CategoryCode   string      `json:"categoryCode,omitempty"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Status         string      `json:"status"`
	Priority       string      `json:"priority"`
	DueAt          string      `json:"dueAt,omitempty"`
	CompletedAt    string      `json:"completedAt,omitempty"`
	Images         []MediaItem `json:"images"`
	AssigneeUserID uint64      `json:"assigneeUserId"`
	CreatedBy      uint64      `json:"createdBy"`
	CreatedAt      string      `json:"createdAt"`
	UpdatedAt      string      `json:"updatedAt"`
}

type TodoCreateReq struct {
	CategoryID     uint64      `json:"categoryId" binding:"required"`
	Title          string      `json:"title" binding:"required"`
	Description    string      `json:"description"`
	Status         string      `json:"status"`
	Priority       string      `json:"priority"`
	DueAt          string      `json:"dueAt"`
	Images         []MediaItem `json:"images"`
	AssigneeUserID uint64      `json:"assigneeUserId"`
}

type TodoUpdateReq struct {
	CategoryID     *uint64     `json:"categoryId"`
	Title          *string     `json:"title"`
	Description    *string     `json:"description"`
	Status         *string     `json:"status"`
	Priority       *string     `json:"priority"`
	DueAt          *string     `json:"dueAt"`
	ClearDueAt     bool        `json:"clearDueAt"`
	Images         []MediaItem `json:"images"`
	AssigneeUserID *uint64     `json:"assigneeUserId"`
}

type TodoStatusReq struct {
	Status string `json:"status" binding:"required"`
}

type TodoListQuery struct {
	CategoryID uint64 `form:"categoryId"`
	Status     string `form:"status"`
	Priority   string `form:"priority"`
	Keyword    string `form:"keyword"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

type DashboardStats struct {
	Total      int64            `json:"total"`
	Pending    int64            `json:"pending"`
	InProgress int64            `json:"inProgress"`
	Done       int64            `json:"done"`
	Cancelled  int64            `json:"cancelled"`
	ByCategory map[string]int64 `json:"byCategory"`
}
