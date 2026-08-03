package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, h *Handlers) {
	g.GET("/dashboard/stats", h.DashboardStats)

	g.GET("/categories", h.ListCategories)
	g.POST("/categories", h.CreateCategory)
	g.PUT("/categories/:id", h.UpdateCategory)
	g.DELETE("/categories/:id", h.DeleteCategory)

	g.GET("/todos", h.ListTodos)
	g.POST("/todos", h.CreateTodo)
	g.GET("/todos/:id", h.GetTodo)
	g.PUT("/todos/:id", h.UpdateTodo)
	g.PATCH("/todos/:id/status", h.UpdateTodoStatus)
	g.DELETE("/todos/:id", h.DeleteTodo)

	g.GET("/notifications", h.GetNotify)
	g.PUT("/notifications", h.SaveNotify)
	g.POST("/notifications/test", h.TestNotify)
	g.POST("/notifications/run", h.RunNotify)
	g.POST("/notifications/reset-state", h.ResetNotifyState)
}
