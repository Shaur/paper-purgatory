package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"paper/purgatory/model"
	"paper/purgatory/service"
	"paper/purgatory/utils"
	"time"

	"gorm.io/gorm"
)

func (w *Worker) Run(ctx context.Context) {
	log.Println("worker started")
	for {
		if w.ProcessOnce() {
			continue
		}
		select {
		case <-ctx.Done():
			log.Println("worker stopped")
			return
		case <-time.After(w.interval):
		}
	}
}

type Worker struct {
	db       *gorm.DB
	service  service.PurgatoryService
	interval time.Duration
}

func NewWorker(db *gorm.DB, service service.PurgatoryService, interval time.Duration) *Worker {
	return &Worker{
		db:       db,
		service:  service,
		interval: interval,
	}
}

func (w *Worker) ProcessOnce() bool {
	var job model.Job
	res := w.db.Raw(`
		update job set status='started', update_time = now(), attempts = attempts + 1
		where id = (select id from job where status='pending' order by id limit 1 for update skip locked) returning *
	`).Scan(&job)

	if res.Error != nil {
		log.Println("claim failed:", res.Error)
		return false
	}

	if res.RowsAffected == 0 {
		return false
	}

	err := w.process(job)
	if err != nil {
		var args = map[string]any{
			"status": "failed", "error_message": err.Error(), "update_time": time.Now(),
		}

		if err = w.db.Model(&model.Job{Id: job.Id}).Updates(args).Error; err != nil {
			log.Println("status update failed:", err)
		}
		return true
	}

	var args = map[string]any{"status": "done", "update_time": time.Now()}
	if err = w.db.Model(&model.Job{Id: job.Id}).Updates(args).Error; err != nil {
		log.Println("status update failed:", err)
	}

	return true
}

func (w *Worker) process(job model.Job) error {
	file, err := os.Open(job.FilePath)
	if err != nil {
		return fmt.Errorf("open job file: %w", err)
	}

	defer utils.HandleClose(file.Close)

	return w.service.Save(file, job.FileName)
}
