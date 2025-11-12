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
	resp, err := s.grpcClient.GetJob(ctx, nodeID, token)
	if err != nil {
		return fmt.Errorf("GetJob failed: %w", err)
	}

	if resp.JobAvailable {
		log.Printf("[GetJobService] Received job: %s (%s)", resp.Job.JobName, resp.Job.JobId)
		if err := s.handleJob(ctx, resp.Job, nodeID, token); err != nil {
			return fmt.Errorf("failed to handle job: %w", err)
		}
	}
	// Only log when job is available, silence "no job" messages

	return nil
}

func (s *GetJobService) handleJob(ctx context.Context, job *pb.Job, nodeID string, token string) error {
	// Create driver for the job
	driver, err := taskdriver.NewDriver(job.DriverType)
	if err != nil {
		// Report failure to Centro
		s.updateJobStatus(ctx, job.JobId, nodeID, token, "failed", fmt.Sprintf("Failed to create driver: %v", err))
		return fmt.Errorf("failed to create driver for job %s: %w", job.JobName, err)
	}

	log.Printf("[GetJobService] Running job: %s with driver: %s", job.JobName, job.DriverType)

	s.updateJobStatus(ctx, job.JobId, nodeID, token, "running", fmt.Sprintf("Running job: %s", job.JobName))

	// Run the job directly (each job is now a single container)
	err = driver.Run(ctx, job)
	if err != nil {
		// Report failure to Centro
		s.updateJobStatus(ctx, job.JobId, nodeID, token, "failed", fmt.Sprintf("Job failed: %v", err))
		return fmt.Errorf("failed to run job %s: %w", job.JobName, err)
	}

	log.Printf("[GetJobService] Job %s started successfully", job.JobName)

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
