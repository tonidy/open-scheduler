package job

import (
	"context"
	"fmt"
	"log"
	"time"

	agentgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/taskdriver"
	pb "github.com/open-scheduler/proto"
)

type GetJobService struct {
	grpcClient *agentgrpc.GrpcClient
}

func NewGetJobService(grpcClient *agentgrpc.GrpcClient) (*GetJobService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &GetJobService{
		grpcClient: grpcClient,
	}, nil
}

func (s *GetJobService) Execute(ctx context.Context, nodeID string, token string) error {
	log.Printf("[GetJobService] Requesting job for node: %s", nodeID)

	resp, err := s.grpcClient.GetJob(ctx, nodeID, token)
	if err != nil {
		return fmt.Errorf("GetJob failed: %w", err)
	}

	if resp.JobAvailable {
		if err := s.handleJob(ctx, resp.Job, nodeID, token); err != nil {
			return fmt.Errorf("failed to handle job: %w", err)
		}
	} else {
		log.Printf("[GetJobService] No job available: %s", resp.ResponseMessage)
	}

	return nil
}

func (s *GetJobService) handleJob(ctx context.Context, job *pb.Job, nodeID string, token string) error {
	log.Printf("[GetJobService] Received job:")
	log.Printf("  Job ID: %s", job.JobId)
	log.Printf("  Name: %s", job.JobName)
	log.Printf("  Type: %s", job.JobType)
	log.Printf("  Selected Clusters: %v", job.SelectedClusters)

	if len(job.JobMetadata) > 0 {
		log.Printf("  Metadata:")
		for k, v := range job.JobMetadata {
			log.Printf("    %s: %s", k, v)
		}
	}

	// Job is already assigned/claimed by GetJob (via dequeue), so we can start executing
	log.Printf("[GetJobService] Job %s already assigned to this node, starting execution", job.JobId)

	for _, task := range job.Tasks {
		driver, err := taskdriver.NewDriver(task.DriverType)
		if err != nil {
			// Report failure to Centro
			s.updateJobStatus(ctx, job.JobId, nodeID, token, "failed", fmt.Sprintf("Failed to create driver: %v", err))
			return fmt.Errorf("failed to create driver for task %s: %w", task.TaskName, err)
		}

		log.Printf("[GetJobService] Running task: %s with driver: %s", task.TaskName, task.DriverType)

		// Update status to running
		s.updateJobStatus(ctx, job.JobId, nodeID, token, "running", fmt.Sprintf("Running task: %s", task.TaskName))

		err = driver.Run(ctx, task)
		if err != nil {
			// Report failure to Centro
			s.updateJobStatus(ctx, job.JobId, nodeID, token, "failed", fmt.Sprintf("Task %s failed: %v", task.TaskName, err))
			return fmt.Errorf("failed to run task %s: %w", task.TaskName, err)
		}
		log.Printf("[GetJobService] Task %s completed successfully", task.TaskName)
	}

	// Report completion to Centro
	if err := s.updateJobStatus(ctx, job.JobId, nodeID, token, "completed", "All tasks completed successfully"); err != nil {
		log.Printf("[GetJobService] Warning: Failed to update job status: %v", err)
	}

	return nil
}

func (s *GetJobService) updateJobStatus(ctx context.Context, jobID string, nodeID string, token string, status string, detail string) error {
	log.Printf("[GetJobService] Updating job %s status to: %s", jobID, status)

	timestamp := time.Now().Unix()
	resp, err := s.grpcClient.UpdateStatus(ctx, nodeID, token, jobID, status, detail, timestamp)
	if err != nil {
		return fmt.Errorf("UpdateStatus failed: %w", err)
	}

	if !resp.Acknowledged {
		return fmt.Errorf("UpdateStatus failed: %s", resp.ResponseMessage)
	}

	log.Printf("[GetJobService] Status updated: %s", resp.ResponseMessage)
	return nil
}
