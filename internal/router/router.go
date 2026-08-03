package router

import (
	"path/filepath"

	"todocenter/admin"
	adminmw "todocenter/admin/middleware"
	"todocenter/internal/config"
	jwtmgr "todocenter/internal/pkg/jwt"
	"todocenter/internal/repo"
	"todocenter/internal/service"
	"todocenter/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	if cfg.Storage.Driver == "local" || cfg.Storage.Driver == "" {
		uploadDir := filepath.Join(cfg.Storage.LocalPath, cfg.Storage.Prefix)
		r.Static("/uploads", uploadDir)
	}

	store, err := storage.New(&cfg.Storage)
	if err != nil {
		panic(err)
	}

	repos := repo.New(db)
	svc := service.NewTodoService(repos)
	notifySvc := service.NewNotifyService(repos, svc)
	h := admin.NewHandlers(svc, notifySvc)
	uploadH := admin.NewUploadHandler(store)
	photoH := admin.NewPhotoUploadHandler(store)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "todocenter"})
	})

	v1 := r.Group("/api/v1")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)

	mobile := v1.Group("/mobile")
	{
		mobile.GET("/photo-upload/:token", photoH.MobileGet)
		mobile.POST("/photo-upload/:token", photoH.MobileUpload)
	}

	adminGroup := v1.Group("/admin")
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	admin.RegisterRoutes(adminGroup, h)
	adminGroup.POST("/upload", uploadH.Upload)
	adminGroup.POST("/photo-upload-sessions", photoH.CreateSession)
	adminGroup.GET("/photo-upload-sessions/:token", photoH.GetSession)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
