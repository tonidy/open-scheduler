package job

import (
	"context"
	"fmt"
	"log"
	"time"

	agentgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/service/instance"
	"github.com/open-scheduler/agent/taskdriver"
	pb "github.com/open-scheduler/proto"
)

type GetDeploymentService struct {
	grpcClient *agentgrpc.GrpcClient
	instanceService *instance.SetInstanceDataService
}

func NewGetDeploymentService(grpcClient *agentgrpc.GrpcClient, instanceService *instance.SetInstanceDataService) (*GetDeploymentService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &GetDeploymentService{
		grpcClient: grpcClient,
		instanceService: instanceService,
	}, nil
}

func (s *GetDeploymentService) Execute(ctx context.Context, nodeID string, token string) error {
	resp, err := s.grpcClient.GetDeployment(ctx, nodeID, token)
	if err != nil {
		return fmt.Errorf("GetDeployment failed: %w", err)
	}

	if resp.DeploymentAvailable {
		log.Printf("[GetDeploymentService] Received deployment: %s (%s)", resp.Deployment.DeploymentName, resp.Deployment.DeploymentId)
		if err := s.handleDeployment(ctx, resp.Deployment, nodeID, token); err != nil {
			return fmt.Errorf("failed to handle deployment: %w", err)
		}
	}
	// Only log when deployment is available, silence "no deployment" messages

	return nil
}

func (s *GetDeploymentService) handleDeployment(ctx context.Context, deployment *pb.Deployment, nodeID string, token string) error {
	// Create driver for the deployment
	driver, err := taskdriver.NewDriver(deployment.DriverType)
	if err != nil {
		// Report failure to Centro
		errMsg := fmt.Sprintf("Failed to create driver '%s': %v", deployment.DriverType, err)
		log.Printf("[GetDeploymentService] Deployment %s failed: %s", deployment.DeploymentId, errMsg)
		s.updateDeploymentStatus(ctx, deployment.DeploymentId, nodeID, token, "failed", errMsg)
		return fmt.Errorf("failed to create driver for deployment %s: %w", deployment.DeploymentName, err)
	}

	log.Printf("[GetDeploymentService] Running deployment: %s (%s) with driver: %s", deployment.DeploymentName, deployment.DeploymentId, deployment.DriverType)

	s.updateDeploymentStatus(ctx, deployment.DeploymentId, nodeID, token, "provisioning", fmt.Sprintf("Provisioning deployment: %s", deployment.DeploymentName))
	
	id, err := driver.Run(ctx, deployment)
	if err != nil {
		// Report failure to Centro
		errMsg := fmt.Sprintf("Deployment execution failed: %v", err)
		log.Printf("[GetDeploymentService] Deployment %s (%s) failed: %s", deployment.DeploymentName, deployment.DeploymentId, errMsg)
		s.updateDeploymentStatus(ctx, deployment.DeploymentId, nodeID, token, "failed", errMsg)
		return fmt.Errorf("failed to run deployment %s: %w", deployment.DeploymentName, err)
	}

	err = s.instanceService.SetInstanceData(ctx, nodeID, token, deployment.DeploymentId, id)
	if err != nil {
		return fmt.Errorf("failed to set instance data: %w", err)
	}

	s.updateDeploymentStatus(ctx, deployment.DeploymentId, nodeID, token, "running", fmt.Sprintf("Running deployment: %s", deployment.DeploymentName))

	log.Printf("[GetDeploymentService] Deployment %s started successfully", deployment.DeploymentName)

	return nil
}

func (s *GetDeploymentService) updateDeploymentStatus(ctx context.Context, deploymentID string, nodeID string, token string, status string, detail string) error {
	log.Printf("[GetDeploymentService] Updating deployment %s status to: %s", deploymentID, status)

	timestamp := time.Now().Unix()
	resp, err := s.grpcClient.UpdateStatus(ctx, nodeID, token, deploymentID, status, detail, timestamp)
	if err != nil {
		return fmt.Errorf("UpdateStatus failed: %w", err)
	}

	if !resp.Acknowledged {
		return fmt.Errorf("UpdateStatus failed: %s", resp.ResponseMessage)
	}	

	log.Printf("[GetDeploymentService] Status updated: %s", resp.ResponseMessage)
	return nil
}
