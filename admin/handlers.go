package admin

import (
	"errors"
	"net/http"
	"strconv"

	"todocenter/internal/dto"
	"todocenter/internal/pkg/authcontext"
	"todocenter/internal/pkg/response"
	"todocenter/internal/service"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	svc *service.TodoService
}

func NewHandlers(svc *service.TodoService) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrBadRequest):
		response.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrConflict):
		response.Fail(c, http.StatusConflict, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, err.Error())
	}
}

func (h *Handlers) DashboardStats(c *gin.Context) {
	stats, err := h.svc.DashboardStats(authcontext.TenantID(c))
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, stats)
}

func (h *Handlers) ListCategories(c *gin.Context) {
	list, err := h.svc.ListCategories(authcontext.TenantID(c))
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *Handlers) CreateCategory(c *gin.Context) {
	var req dto.CategoryCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	node, err := h.svc.CreateCategory(authcontext.TenantID(c), req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Created(c, node)
}

func (h *Handlers) UpdateCategory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req dto.CategoryUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	node, err := h.svc.UpdateCategory(authcontext.TenantID(c), id, req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, node)
}

func (h *Handlers) DeleteCategory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.DeleteCategory(authcontext.TenantID(c), id); err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handlers) ListTodos(c *gin.Context) {
	var q dto.TodoListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	list, total, err := h.svc.ListTodos(authcontext.TenantID(c), q)
	if err != nil {
		h.mapError(c, err)
		return
	}
	page, pageSize := q.Page, q.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	response.OK(c, gin.H{"list": list, "total": total, "page": page, "pageSize": pageSize})
}

func (h *Handlers) GetTodo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	item, err := h.svc.GetTodo(authcontext.TenantID(c), id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) CreateTodo(c *gin.Context) {
	var req dto.TodoCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.CreateTodo(authcontext.TenantID(c), authcontext.UserID(c), req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *Handlers) UpdateTodo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req dto.TodoUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.UpdateTodo(authcontext.TenantID(c), id, req)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) UpdateTodoStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req dto.TodoStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.UpdateTodoStatus(authcontext.TenantID(c), id, req.Status)
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handlers) DeleteTodo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.DeleteTodo(authcontext.TenantID(c), id); err != nil {
		h.mapError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
