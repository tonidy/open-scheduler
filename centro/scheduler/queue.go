package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	etcdstorage "github.com/open-scheduler/centro/storage/etcd"
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
			q.checkStaleJobs(ctx)
		case <-ctx.Done():
			log.Printf("[Scheduler] Stopping scheduler run loop.")
			return
		}
	}
}

func (q *Queue) moveFailedJobsToQueue(ctx context.Context) {
	failedJobs, err := q.storage.GetAllFailedJobs(ctx)
	if err != nil {
		log.Printf("[Scheduler] Failed to get all failed jobs: %v", err)
		return
	}

	for _, job := range failedJobs {

		// Check if job has exceeded max retries (if max_retries > 0)
		if job.MaxRetries > 0 && job.RetryCount >= job.MaxRetries {
			log.Printf("[Scheduler] Job %s exceeded max retries (%d/%d), moving to history",
				job.JobId, job.RetryCount, job.MaxRetries)

			// Save to job history as permanently failed
			jobStatus := &etcdstorage.JobStatus{
				Job:       job,
				Status:    "failed",
				Detail:    fmt.Sprintf("Job exceeded maximum retry limit (%d retries)", job.MaxRetries),
				UpdatedAt: time.Now(),
			}
			if err := q.storage.SaveJobHistory(ctx, job.JobId, jobStatus); err != nil {
				log.Printf("[Scheduler] Failed to save permanently failed job to history: %v", err)
			}

			// Save event
			if err := q.storage.SaveJobEvent(ctx, job.JobId,
				fmt.Sprintf("[%s] Job permanently failed after %d retries (max: %d)",
					time.Now().Format(time.RFC3339), job.RetryCount, job.MaxRetries)); err != nil {
				log.Printf("[Scheduler] Failed to save job event: %v", err)
			}

			// Delete from failed queue
			if err := q.storage.DeleteFailedJob(ctx, job.JobId); err != nil {
				log.Printf("[Scheduler] Failed to delete failed job: %v", err)
			}
			continue
		}

		log.Printf("[Scheduler] Retrying failed job %s (attempt %d/%d)",
			job.JobId, job.RetryCount+1, job.MaxRetries)
		job.RetryCount = job.RetryCount + 1
		job.LastRetryTime = time.Now().Unix()
		if err := q.storage.EnqueueJob(ctx, job); err != nil {
			log.Printf("[Scheduler] Failed to enqueue job: %v", err)
			continue
		}

		// Save retry event
		if err := q.storage.SaveJobEvent(ctx, job.JobId,
			fmt.Sprintf("[%s] Retrying job (attempt %d)",
				time.Now().Format(time.RFC3339), job.RetryCount)); err != nil {
			log.Printf("[Scheduler] Failed to save retry event: %v", err)
		}

		if err := q.storage.DeleteFailedJob(ctx, job.JobId); err != nil {
			log.Printf("[Scheduler] Failed to delete failed job: %v", err)
			continue
		}
	}
}

// checkStaleJobs detects jobs that are stuck in "assigned" or "running" state
// without updates for too long and moves them back to the queue or failed queue
func (q *Queue) checkStaleJobs(ctx context.Context) {
	activeJobs, err := q.storage.GetAllActiveJobs(ctx)
	if err != nil {
		log.Printf("[Scheduler] Failed to get active jobs: %v", err)
		return
	}

	now := time.Now()
	// Timeout thresholds
	assignedTimeout := 5 * time.Minute // Job assigned but never started running
	runningTimeout := 30 * time.Minute // Job running but no status updates

	for jobID, jobStatus := range activeJobs {
		timeSinceUpdate := now.Sub(jobStatus.UpdatedAt)

		// Check if job is stale based on its status
		isStale := false
		reason := ""

		if jobStatus.Status == "assigned" && timeSinceUpdate > assignedTimeout {
			isStale = true
			reason = fmt.Sprintf("Job assigned to node %s but never started running (timeout: %v)",
				jobStatus.NodeID, assignedTimeout)
		} else if jobStatus.Status == "running" && timeSinceUpdate > runningTimeout {
			isStale = true
			reason = fmt.Sprintf("Job running on node %s with no status updates (timeout: %v)",
				jobStatus.NodeID, runningTimeout)
		}

		if isStale {
			log.Printf("[Scheduler] Detected stale job %s: %s", jobID, reason)

			// Save event
			if err := q.storage.SaveJobEvent(ctx, jobID,
				fmt.Sprintf("[%s] Job detected as stale: %s",
					time.Now().Format(time.RFC3339), reason)); err != nil {
				log.Printf("[Scheduler] Failed to save stale job event: %v", err)
			}

			// Move job to failed queue for retry
			if jobStatus.Job != nil {
				if err := q.storage.EnqueueFailedJob(ctx, jobStatus.Job); err != nil {
					log.Printf("[Scheduler] Failed to enqueue stale job: %v", err)
				} else {
					log.Printf("[Scheduler] Moved stale job %s to failed queue for retry", jobID)
				}
			}

			// Remove from active jobs
			if err := q.storage.DeleteJobActive(ctx, jobID); err != nil {
				log.Printf("[Scheduler] Failed to delete stale active job: %v", err)
			}
		}
	}
}
