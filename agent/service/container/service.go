package container

import (
	"context"
	"fmt"
	"log"
	"time"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/taskdriver"
)

type SetContainerDataService struct {
	grpcClient *sharedgrpc.GrpcClient
	driver     taskdriver.Driver
	token      string
	nodeID     string
}

func NewSetContainerDataService(grpcClient *sharedgrpc.GrpcClient, driver taskdriver.Driver, token string, nodeID string) (*SetContainerDataService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &SetContainerDataService{
		grpcClient: grpcClient,
		driver:     driver,
		token:      token,
		nodeID:     nodeID,
	}, nil
}

func (s *SetContainerDataService) Execute(ctx context.Context, nodeID string, token string) error {
	log.Printf("[SetContainerDataService] Collecting container data for node: %s", s.nodeID)

	if s.driver == nil {
		log.Printf("[SetContainerDataService] No driver configured, skipping container data collection")
		return nil
	}

	containers, err := s.driver.ListContainers(ctx)
	if err != nil {
		log.Printf("[SetContainerDataService] Failed to list containers: %v", err)
		return fmt.Errorf("failed to list containers: %w", err)
	}

	log.Printf("[SetContainerDataService] Found %d containers", len(containers))

	for _, container := range containers {
		jobID, hasJobID := container.Labels["open-scheduler.job-id"]
		if !hasJobID || jobID == "" {
			log.Printf("[SetContainerDataService] Container %s has no job-id label, skipping", container.ContainerId)
			continue
		}

		log.Printf("[SetContainerDataService] Sending container data for job %s, container %s", jobID, container.ContainerId)

		resp, err := s.grpcClient.SetContainerData(
			ctx,
			s.nodeID,
			s.token,
			jobID,
			container,
			time.Now().Unix(),
		)

		if err != nil {
			log.Printf("[SetContainerDataService] Failed to send container data for job %s: %v", jobID, err)
			continue
		}

		if !resp.Acknowledged {
			log.Printf("[SetContainerDataService] Container data rejected for job %s: %s", jobID, resp.ResponseMessage)
			continue
		}

		log.Printf("[SetContainerDataService] Container data sent successfully for job %s: %s", jobID, resp.ResponseMessage)
	}

	return nil
}
