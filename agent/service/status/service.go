package status

import (
	"context"
	"fmt"
	"log"
	"time"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/taskdriver"
)

type UpdateStatusService struct {
	grpcClient *sharedgrpc.GrpcClient
	driver     taskdriver.Driver
	token      string
	nodeID     string
}

func NewUpdateStatusService(grpcClient *sharedgrpc.GrpcClient, driver taskdriver.Driver, token string, nodeID string) (*UpdateStatusService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &UpdateStatusService{
		grpcClient: grpcClient,
		driver:     driver,
		token:      token,
		nodeID:     nodeID,
	}, nil
}

func (s *UpdateStatusService) Execute(ctx context.Context, nodeID string, token string) error {
	log.Printf("[UpdateStatusService] Updating status for node: %s", s.nodeID)

	if s.driver == nil {
		log.Printf("[UpdateStatusService] No driver configured, skipping instance status updates")
		return nil
	}

	instances, err := s.driver.ListInstances(ctx)
	if err != nil {
		log.Printf("[UpdateStatusService] Failed to list instances: %v", err)
		return fmt.Errorf("failed to list instances: %w", err)
	}

	log.Printf("[UpdateStatusService] Found %d instances to update", len(instances))

	for _, instance := range instances {
		jobID, hasJobID := instance.Labels["open-scheduler.job-id"]
		if !hasJobID || jobID == "" {
			log.Printf("[UpdateStatusService] Instance %s has no job-id label, skipping", instance.InstanceId)
			continue
		}

		jobStatus := mapInstanceStatusToJobStatus(instance.Status)
		statusMessage := fmt.Sprintf("Instance %s is %s", instance.InstanceName, instance.Status)

		if instance.Status == "exited" || instance.Status == "failed" {
			statusMessage = fmt.Sprintf("%s (exit code: %d)", statusMessage, instance.ExitCode)
		}

		log.Printf("[UpdateStatusService] Updating job %s: status=%s, instance=%s", jobID, jobStatus, instance.InstanceId)

		resp, err := s.grpcClient.UpdateStatus(
			ctx,
			s.nodeID,
			s.token,
			jobID,
			jobStatus,
			statusMessage,
			time.Now().Unix(),
		)

		if err != nil {
			log.Printf("[UpdateStatusService] Failed to update status for job %s: %v", jobID, err)
			continue
		}

		if !resp.Acknowledged {
			log.Printf("[UpdateStatusService] Status update rejected for job %s: %s", jobID, resp.ResponseMessage)
			continue
		}

		log.Printf("[UpdateStatusService] Status update successful for job %s: %s", jobID, resp.ResponseMessage)
	}

	return nil
}

func mapInstanceStatusToJobStatus(instanceStatus string) string {
	switch instanceStatus {
	case "running":
		return "running"
	case "exited":
		return "completed"
	case "failed":
		return "failed"
	case "stopped":
		return "stopped"
	default:
		return "unknown"
	}
}
