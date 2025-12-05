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
	failedDeployments, err := q.storage.GetAllFailedDeployments(ctx)
	if err != nil {
		log.Printf("[Scheduler] Failed to get all failed deployments: %v", err)
		return
	}

	for _, deployment := range failedDeployments {

		// Check if deployment has exceeded max retries (if max_retries > 0)
		if deployment.MaxRetries > 0 && deployment.RetryCount >= deployment.MaxRetries {
			log.Printf("[Scheduler] Deployment %s exceeded max retries (%d/%d), moving to history",
				deployment.DeploymentId, deployment.RetryCount, deployment.MaxRetries)

			// Save to deployment history as permanently failed
			deploymentStatus := &etcdstorage.DeploymentStatus{
				Deployment: deployment,
				Status:     "failed",
				Detail:     fmt.Sprintf("Deployment exceeded maximum retry limit (%d retries)", deployment.MaxRetries),
				UpdatedAt:  time.Now(),
			}
			if err := q.storage.SaveDeploymentHistory(ctx, deployment.DeploymentId, deploymentStatus); err != nil {
				log.Printf("[Scheduler] Failed to save permanently failed deployment to history: %v", err)
			}

			// Save event
			if err := q.storage.SaveDeploymentEvent(ctx, deployment.DeploymentId,
				fmt.Sprintf("[%s] Deployment permanently failed after %d retries (max: %d)",
					time.Now().Format(time.RFC3339), deployment.RetryCount, deployment.MaxRetries)); err != nil {
				log.Printf("[Scheduler] Failed to save deployment event: %v", err)
			}

			// Delete from failed queue
			if err := q.storage.DeleteFailedDeployment(ctx, deployment.DeploymentId); err != nil {
				log.Printf("[Scheduler] Failed to delete failed deployment: %v", err)
			}
			continue
		}

		log.Printf("[Scheduler] Retrying failed deployment %s (attempt %d/%d)",
			deployment.DeploymentId, deployment.RetryCount+1, deployment.MaxRetries)
		deployment.RetryCount = deployment.RetryCount + 1
		deployment.LastRetryTime = time.Now().Unix()
		if err := q.storage.EnqueueDeployment(ctx, deployment); err != nil {
			log.Printf("[Scheduler] Failed to enqueue deployment: %v", err)
			continue
		}

		// Save retry event
		if err := q.storage.SaveDeploymentEvent(ctx, deployment.DeploymentId,
			fmt.Sprintf("[%s] Retrying deployment (attempt %d)",
				time.Now().Format(time.RFC3339), deployment.RetryCount)); err != nil {
			log.Printf("[Scheduler] Failed to save retry event: %v", err)
		}

		if err := q.storage.DeleteFailedDeployment(ctx, deployment.DeploymentId); err != nil {
			log.Printf("[Scheduler] Failed to delete failed deployment: %v", err)
			continue
		}
	}
}

// checkStaleJobs detects deployments that are stuck in "assigned" or "running" state
// without updates for too long and moves them back to the queue or failed queue
func (q *Queue) checkStaleJobs(ctx context.Context) {
	activeDeployments, err := q.storage.GetAllActiveDeployments(ctx)
	if err != nil {
		log.Printf("[Scheduler] Failed to get active deployments: %v", err)
		return
	}

	now := time.Now()
	// Timeout thresholds
	assignedTimeout := 5 * time.Minute // Deployment assigned but never started running
	runningTimeout := 30 * time.Minute // Deployment running but no status updates

	for deploymentID, deploymentStatus := range activeDeployments {
		timeSinceUpdate := now.Sub(deploymentStatus.UpdatedAt)

		// Check if deployment is stale based on its status
		isStale := false
		reason := ""

		if deploymentStatus.Status == "assigned" && timeSinceUpdate > assignedTimeout {
			isStale = true
			reason = fmt.Sprintf("Deployment assigned to node %s but never started running (timeout: %v)",
				deploymentStatus.NodeID, assignedTimeout)
		} else if deploymentStatus.Status == "running" && timeSinceUpdate > runningTimeout {
			isStale = true
			reason = fmt.Sprintf("Deployment running on node %s with no status updates (timeout: %v)",
				deploymentStatus.NodeID, runningTimeout)
		}

		if isStale {
			log.Printf("[Scheduler] Detected stale deployment %s: %s", deploymentID, reason)

			// Save event
			if err := q.storage.SaveDeploymentEvent(ctx, deploymentID,
				fmt.Sprintf("[%s] Deployment detected as stale: %s",
					time.Now().Format(time.RFC3339), reason)); err != nil {
				log.Printf("[Scheduler] Failed to save stale deployment event: %v", err)
			}

			// Move deployment to failed queue for retry
			if deploymentStatus.Deployment != nil {
				if err := q.storage.EnqueueFailedDeployment(ctx, deploymentStatus.Deployment); err != nil {
					log.Printf("[Scheduler] Failed to enqueue stale deployment: %v", err)
				} else {
					log.Printf("[Scheduler] Moved stale deployment %s to failed queue for retry", deploymentID)
				}
			}

			// Remove from active deployments
			if err := q.storage.DeleteDeploymentActive(ctx, deploymentID); err != nil {
				log.Printf("[Scheduler] Failed to delete stale active deployment: %v", err)
			}
		}
	}
}
