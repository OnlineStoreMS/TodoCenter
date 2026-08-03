package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"todocenter/internal/config"
	"todocenter/internal/database"
	"todocenter/internal/repo"
	"todocenter/internal/router"
	"todocenter/internal/scheduler"
	"todocenter/internal/service"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	absConfig, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.Load(absConfig)
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal(err)
	}
	log.Printf("database connected: driver=%s", cfg.Database.Driver)

	repos := repo.New(db)
	todoSvc := service.NewTodoService(repos)
	notifySvc := service.NewNotifyService(repos, todoSvc)
	scheduler.StartDueNotify(notifySvc)

	engine := router.Setup(db, cfg)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("TodoCenter API listening on http://localhost%s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatal(err)
	}
}
