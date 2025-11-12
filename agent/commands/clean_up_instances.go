package commands

import (
	"context"
	"fmt"
	"log"

	cleanupservice "github.com/open-scheduler/agent/service/cleanup"
)

type CleanUpInstancesCommand struct {
	service *cleanupservice.CleanupService
}

func NewCleanUpInstancesCommand(service *cleanupservice.CleanupService) *CleanUpInstancesCommand {
	return &CleanUpInstancesCommand{
		service: service,
	}
}

func (c *CleanUpInstancesCommand) Execute(ctx context.Context, nodeID string, token string) error {
	if c.service == nil {
		return fmt.Errorf("cleanup service is not initialized")
	}

	log.Printf("[CleanUpInstancesCommand] Executing cleanup for node: %s", nodeID)

	return c.service.Execute(ctx, nodeID, token)
}

func (c *CleanUpInstancesCommand) Name() string {
	return "clean_up_instances"
}

func (c *CleanUpInstancesCommand) String() string {
	return "CleanUpInstancesCommand"
}

func (c *CleanUpInstancesCommand) IntervalSeconds() int {
	return 60 // Run every 1 minute (60 seconds)
}
