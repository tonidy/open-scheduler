package job

import (
	"context"
	"fmt"
	"log"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
	pb "github.com/open-scheduler/proto"
)

type GetJobService struct {
	grpcClient *sharedgrpc.SharedClient
}

func NewGetJobService(grpcClient *sharedgrpc.SharedClient) (*GetJobService, error) {
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

	// TODO: Execute the job tasks
	log.Printf("[GetJobService] Job handling not yet implemented")

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