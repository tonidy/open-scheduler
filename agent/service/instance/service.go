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
		deploymentID, hasDeploymentID := instance.Labels["open-scheduler.deployment-id"]
		if !hasDeploymentID || deploymentID == "" {
			log.Printf("[SetInstanceDataService] Instance %s has no deployment-id label, skipping", instance.InstanceId)
			continue
		}

		log.Printf("[SetInstanceDataService] Sending instance data for deployment %s, instance %s", deploymentID, instance.InstanceId)

		resp, err := s.grpcClient.SetInstanceData(
			ctx,
			s.nodeID,
			s.token,
			deploymentID,
			instance,
			time.Now().Unix(),
		)

		if err != nil {
			log.Printf("[SetInstanceDataService] Failed to send instance data for deployment %s: %v", deploymentID, err)
			continue
		}

		if !resp.Acknowledged {
			log.Printf("[SetInstanceDataService] Instance data rejected for deployment %s: %s", deploymentID, resp.ResponseMessage)
			continue
		}

		log.Printf("[SetInstanceDataService] Instance data sent successfully for deployment %s: %s", deploymentID, resp.ResponseMessage)
	}

	return nil
}

func (s *SetInstanceDataService) SetInstanceData(ctx context.Context, nodeID string, token string, deploymentID string, instanceID string) error {
	log.Printf("[SetInstanceDataService] Setting instance data for deployment %s, instance %s", deploymentID, instanceID)

	instance, err := s.driver.InspectInstance(ctx, instanceID)
	if err != nil {
		log.Printf("[SetInstanceDataService] Failed to inspect instance: %v", err)
		return fmt.Errorf("failed to inspect instance: %w", err)
	}

	log.Printf("[SetInstanceDataService] Instance data: %+v", instance)
	resp, err := s.grpcClient.SetInstanceData(
		ctx,
		s.nodeID,
		s.token,
		deploymentID,
		instance,
		time.Now().Unix(),
	)
	if err != nil {
		log.Printf("[SetInstanceDataService] Failed to send instance data: %v", err)
		return fmt.Errorf("failed to send instance data: %w", err)
	}

	if !resp.Acknowledged {
		log.Printf("[SetInstanceDataService] Instance data rejected: %s", resp.ResponseMessage)
		return fmt.Errorf("instance data rejected: %s", resp.ResponseMessage)
	}

	log.Printf("[SetInstanceDataService] Instance data sent successfully: %s", resp.ResponseMessage)

	return nil
}
