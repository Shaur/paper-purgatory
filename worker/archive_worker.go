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

func (w *Worker) RunGC(ctx context.Context) {
	log.Println("GC started")
	for {
		func() {
			if err := w.recovery(); err != nil {
				log.Println(err)
			}
			w.collectAndDelete()
		}()

		select {
		case <-ctx.Done():
			log.Println("GC worker stopped")
			return
		case <-time.After(1 * time.Hour):
		}
	}
}

func (w *Worker) recovery() error {
	res := w.db.Exec(`update job set status  = 'pending', update_time = now()
	where status = 'started' and update_time < now() - interval '1 hour' and attempts < 3
	`)

	if res.Error != nil {
		return res.Error
	}

	res = w.db.Exec(`update job set status = 'failed', update_time = now(),
			error_message = 'max attempts exceeded'
		where status = 'started' and update_time < now() - interval '1 hour'`)

	return res.Error
}

func (w *Worker) collectAndDelete() {
	var jobs []model.Job
	res := w.db.Find(&jobs, "status = 'done'")
	if res.Error != nil {
		log.Println(res.Error)
		return
	}

	for _, job := range jobs {
		err := os.Remove(job.FilePath)
		if err != nil && !os.IsNotExist(err) {
			log.Println(err)
			continue
		}

		res = w.db.Delete(&job)
		if res.Error != nil {
			log.Println(res.Error)
		}
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
