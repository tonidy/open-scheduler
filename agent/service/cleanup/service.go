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

	// List all instances
	instances, err := s.driver.ListInstances(ctx)
	if err != nil {
		log.Printf("[CleanupService] Failed to list instances: %v", err)
		return fmt.Errorf("failed to list instances: %w", err)
	}

	log.Printf("[CleanupService] Found %d instances", len(instances))

	stoppedCount := 0
	cleanedCount := 0

	// Filter for stopped instances and stop them
	for _, instance := range instances {
		if instance.Status == "stopped" ||
			instance.Status == "exited" {
			stoppedCount++
			log.Printf("[CleanupService] Found stopped instance: %s (Status: %s)", instance.InstanceId, instance.Status)

			// Stop the instance (this will also remove it based on the StopInstance implementation)
			err := s.driver.StopInstance(ctx, instance.InstanceId)
			if err != nil {
				log.Printf("[CleanupService] Failed to stop instance %s: %v", instance.InstanceId, err)
				continue
			}

			cleanedCount++
			log.Printf("[CleanupService] Successfully cleaned up instance: %s", instance.InstanceId)
		}
	}

	log.Printf("[CleanupService] Cleanup complete - Found: %d stopped, Cleaned: %d", stoppedCount, cleanedCount)
	return nil
}
