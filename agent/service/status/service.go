package status

import (
	"context"
	"fmt"
	"log"
	"time"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/taskdriver"
	"github.com/open-scheduler/agent/taskdriver/model"
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
		log.Printf("[UpdateStatusService] No driver configured, skipping container status updates")
		return nil
	}

	containers, err := s.driver.ListContainers(ctx)
	if err != nil {
		log.Printf("[UpdateStatusService] Failed to list containers: %v", err)
		return fmt.Errorf("failed to list containers: %w", err)
	}

	log.Printf("[UpdateStatusService] Found %d containers to update", len(containers))

	for _, container := range containers {
		jobID, hasJobID := container.Labels["open-scheduler.job-id"]
		if !hasJobID || jobID == "" {
			log.Printf("[UpdateStatusService] Container %s has no job-id label, skipping", container.ID)
			continue
		}

		jobStatus := mapContainerStatusToJobStatus(container.Status)
		statusMessage := fmt.Sprintf("Container %s is %s", container.Name, container.Status)

		if container.Status == model.ContainerStatusExited || container.Status == model.ContainerStatusFailed {
			statusMessage = fmt.Sprintf("%s (exit code: %d)", statusMessage, container.ExitCode)
		}

		log.Printf("[UpdateStatusService] Updating job %s: status=%s, container=%s", jobID, jobStatus, container.ID)

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

func mapContainerStatusToJobStatus(containerStatus model.ContainerStatus) string {
	switch containerStatus {
	case model.ContainerStatusRunning:
		return "running"
	case model.ContainerStatusExited:
		return "completed"
	case model.ContainerStatusFailed:
		return "failed"
	case model.ContainerStatusStopped:
		return "stopped"
	default:
		return "unknown"
	}
}
