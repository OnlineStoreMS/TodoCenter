package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"todocenter/internal/config"
	"todocenter/internal/model"
	"todocenter/internal/seed"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresDSN)
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.SQLitePath)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.TodoCategory{},
		&model.Todo{},
	); err != nil {
		return err
	}
	if err := seed.EnsureDefaultCategories(db, 1); err != nil {
		log.Printf("seed default categories: %v", err)
	}
	return nil
}
