package commands

import (
	"context"
	"fmt"
	"log"

	cleanupservice "github.com/open-scheduler/agent/service/cleanup"
)

type CleanUpContainersCommand struct {
	service *cleanupservice.CleanupService
}

func NewCleanUpContainersCommand(service *cleanupservice.CleanupService) *CleanUpContainersCommand {
	return &CleanUpContainersCommand{
		service: service,
	}
}

func (c *CleanUpContainersCommand) Execute(ctx context.Context, nodeID string, token string) error {
	if c.service == nil {
		return fmt.Errorf("cleanup service is not initialized")
	}

	log.Printf("[CleanUpContainersCommand] Executing cleanup for node: %s", nodeID)

	return c.service.Execute(ctx, nodeID, token)
}

func (c *CleanUpContainersCommand) Name() string {
	return "clean_up_containers"
}

func (c *CleanUpContainersCommand) String() string {
	return "CleanUpContainersCommand"
}

func (c *CleanUpContainersCommand) IntervalSeconds() int {
	return 60 // Run every 1 minute (60 seconds)
}
