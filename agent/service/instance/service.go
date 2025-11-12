package instance

import (
	"context"
	"fmt"
	"log"
	"time"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/taskdriver"
)

type SetInstanceDataService struct {
	grpcClient *sharedgrpc.GrpcClient
	driver     taskdriver.Driver
	token      string
	nodeID     string
}

func NewSetInstanceDataService(grpcClient *sharedgrpc.GrpcClient, driver taskdriver.Driver, token string, nodeID string) (*SetInstanceDataService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &SetInstanceDataService{
		grpcClient: grpcClient,
		driver:     driver,
		token:      token,
		nodeID:     nodeID,
	}, nil
}

func (s *SetInstanceDataService) Execute(ctx context.Context, nodeID string, token string) error {
	log.Printf("[SetInstanceDataService] Collecting instance data for node: %s", s.nodeID)

	if s.driver == nil {
		log.Printf("[SetInstanceDataService] No driver configured, skipping instance data collection")
		return nil
	}

	instances, err := s.driver.ListInstances(ctx)
	if err != nil {
		log.Printf("[SetInstanceDataService] Failed to list instances: %v", err)
		return fmt.Errorf("failed to list instances: %w", err)
	}

	log.Printf("[SetInstanceDataService] Found %d instances", len(instances))

	for _, instance := range instances {
		jobID, hasJobID := instance.Labels["open-scheduler.job-id"]
		if !hasJobID || jobID == "" {
			log.Printf("[SetInstanceDataService] Instance %s has no job-id label, skipping", instance.InstanceId)
			continue
		}

		log.Printf("[SetInstanceDataService] Sending instance data for job %s, instance %s", jobID, instance.InstanceId)

		resp, err := s.grpcClient.SetInstanceData(
			ctx,
			s.nodeID,
			s.token,
			jobID,
			instance,
			time.Now().Unix(),
		)

		if err != nil {
			log.Printf("[SetInstanceDataService] Failed to send instance data for job %s: %v", jobID, err)
			continue
		}

		if !resp.Acknowledged {
			log.Printf("[SetInstanceDataService] Instance data rejected for job %s: %s", jobID, resp.ResponseMessage)
			continue
		}

		log.Printf("[SetInstanceDataService] Instance data sent successfully for job %s: %s", jobID, resp.ResponseMessage)
	}

	return nil
}
