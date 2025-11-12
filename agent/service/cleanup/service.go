package cleanup

import (
	"context"
	"fmt"
	"log"

	"github.com/open-scheduler/agent/taskdriver"
)

type CleanupService struct {
	driver taskdriver.Driver
	nodeID string
}

func NewCleanupService(driver taskdriver.Driver, nodeID string) (*CleanupService, error) {
	if driver == nil {
		return nil, fmt.Errorf("driver cannot be nil")
	}

	return &CleanupService{
		driver: driver,
		nodeID: nodeID,
	}, nil
}

func (s *CleanupService) Execute(ctx context.Context, nodeID string, token string) error {
	log.Printf("[CleanupService] Starting cleanup for node: %s", s.nodeID)

	if s.driver == nil {
		log.Printf("[CleanupService] No driver configured, skipping cleanup")
		return nil
	}

	// List all containers
	containers, err := s.driver.ListContainers(ctx)
	if err != nil {
		log.Printf("[CleanupService] Failed to list containers: %v", err)
		return fmt.Errorf("failed to list containers: %w", err)
	}

	log.Printf("[CleanupService] Found %d containers", len(containers))

	stoppedCount := 0
	cleanedCount := 0

	// Filter for stopped containers and stop them
	for _, container := range containers {
		if container.Status == "stopped" ||
			container.Status == "exited" {
			stoppedCount++
			log.Printf("[CleanupService] Found stopped container: %s (Status: %s)", container.ContainerId, container.Status)

			// Stop the container (this will also remove it based on the StopContainer implementation)
			err := s.driver.StopContainer(ctx, container.ContainerId)
			if err != nil {
				log.Printf("[CleanupService] Failed to stop container %s: %v", container.ContainerId, err)
				continue
			}

			cleanedCount++
			log.Printf("[CleanupService] Successfully cleaned up container: %s", container.ContainerId)
		}
	}

	log.Printf("[CleanupService] Cleanup complete - Found: %d stopped, Cleaned: %d", stoppedCount, cleanedCount)
	return nil
}
