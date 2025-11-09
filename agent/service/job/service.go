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
	grpcClient *agentgrpc.SharedClient
}

func NewGetJobService(grpcClient *agentgrpc.SharedClient) (*GetJobService, error) {
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

	if resp.HasJob {
		if err := s.handleJob(ctx, resp.Job, nodeID, token); err != nil {
			return fmt.Errorf("failed to handle job: %w", err)
		}
	} else {
		log.Printf("[GetJobService] No job available: %s", resp.Message)
	}

	return nil
}

func (s *GetJobService) handleJob(ctx context.Context, job *pb.Job, nodeID string, token string) error {
	log.Printf("[GetJobService] Received job:")
	log.Printf("  Job ID: %s", job.JobId)
	log.Printf("  Name: %s", job.Name)
	log.Printf("  Type: %s", job.Type)
	log.Printf("  Datacenters: %s", job.Datacenters)

	if len(job.Meta) > 0 {
		log.Printf("  Metadata:")
		for k, v := range job.Meta {
			log.Printf("    %s: %s", k, v)
		}
	}

	if err := s.claimJob(ctx, job.JobId, nodeID, token); err != nil {
		return fmt.Errorf("failed to claim job: %w", err)
	}

	for _, task := range job.Tasks {
		driver, err := taskdriver.NewDriver(task.Driver)
		if err != nil {
			// Report failure to Centro
			s.updateJobStatus(ctx, job.JobId, nodeID, token, "failed", fmt.Sprintf("Failed to create driver: %v", err))
			return fmt.Errorf("failed to create driver for task %s: %w", task.Name, err)
		}

		log.Printf("[GetJobService] Running task: %s with driver: %s", task.Name, task.Driver)

		// Update status to running
		s.updateJobStatus(ctx, job.JobId, nodeID, token, "running", fmt.Sprintf("Running task: %s", task.Name))

		err = driver.Run(ctx, task)
		if err != nil {
			// Report failure to Centro
			s.updateJobStatus(ctx, job.JobId, nodeID, token, "failed", fmt.Sprintf("Task %s failed: %v", task.Name, err))
			return fmt.Errorf("failed to run task %s: %w", task.Name, err)
		}
		log.Printf("[GetJobService] Task %s completed successfully", task.Name)
	}

	// Report completion to Centro
	if err := s.updateJobStatus(ctx, job.JobId, nodeID, token, "completed", "All tasks completed successfully"); err != nil {
		log.Printf("[GetJobService] Warning: Failed to update job status: %v", err)
	}

	return nil
}

func (s *GetJobService) claimJob(ctx context.Context, jobID string, nodeID string, token string) error {
	log.Printf("[GetJobService] Claiming job: %s for node: %s", jobID, nodeID)

	resp, err := s.grpcClient.ClaimJob(ctx, nodeID, jobID, token)
	if err != nil {
		return fmt.Errorf("ClaimJob failed: %w", err)
	}

	if !resp.Ok {
		return fmt.Errorf("ClaimJob failed: %s", resp.Message)
	}

	log.Printf("[GetJobService] Claimed job: %s", resp.Message)

	return nil
}

func (s *GetJobService) updateJobStatus(ctx context.Context, jobID string, nodeID string, token string, status string, detail string) error {
	log.Printf("[GetJobService] Updating job %s status to: %s", jobID, status)

	timestamp := time.Now().Unix()
	resp, err := s.grpcClient.UpdateStatus(ctx, nodeID, token, jobID, status, detail, timestamp)
	if err != nil {
		return fmt.Errorf("UpdateStatus failed: %w", err)
	}

	if !resp.Ok {
		return fmt.Errorf("UpdateStatus failed: %s", resp.Message)
	}

	log.Printf("[GetJobService] Status updated: %s", resp.Message)
	return nil
}
