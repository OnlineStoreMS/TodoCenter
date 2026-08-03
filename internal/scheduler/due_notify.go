package scheduler

import (
	"context"
	"log"
	"time"

	"todocenter/internal/service"
)

// StartDueNotify 后台轮询：到期/逾期待办飞书提醒（含月待办实例）。
func StartDueNotify(svc *service.NotifyService) {
	go func() {
		time.Sleep(45 * time.Second)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			svc.RunAllEnabled(ctx)
			cancel()
			wait := svc.MinPollInterval()
			if wait < time.Minute {
				wait = time.Minute
			}
			log.Printf("todo due-notify: next poll in %s", wait)
			time.Sleep(wait)
		}
	}()
	log.Printf("todo due-notify scheduler started")
}
