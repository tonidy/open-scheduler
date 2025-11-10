package scheduler

import (
	"context"
	"log"

	etcdstorage "github.com/open-scheduler/centro/storage/etcd"
	"time"
)

type Queue struct {
	storage *etcdstorage.Storage
}

func NewQueue(storage *etcdstorage.Storage) *Queue {
	return &Queue{storage: storage}
}

func (q *Queue) StartScheduler(ctx context.Context) {
	log.Printf("[Scheduler] Starting scheduler run loop.")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				q.moveFailedJobsToQueue(ctx)
			case <-ctx.Done():
				log.Printf("[Scheduler] Stopping scheduler run loop.")
				return
			}
		}
}

func (q *Queue) moveFailedJobsToQueue(ctx context.Context) {
	failedJobs, err := q.storage.GetAllFailedJobs(ctx);
	if err != nil {
		log.Printf("[Scheduler] Failed to get all failed jobs: %v", err)
		return
	}
	for _, job := range failedJobs {
		if err := q.storage.EnqueueJob(ctx, job); err != nil {
			log.Printf("[Scheduler] Failed to enqueue job: %v", err)
			continue
		}
		if err := q.storage.DeleteFailedJob(ctx, job.JobId); err != nil {
			log.Printf("[Scheduler] Failed to delete failed job: %v", err)
			continue
		}
	}
}