package commands

import (
	"context"
	"fmt"
	"log"

	containerservice "github.com/open-scheduler/agent/service/container"
)

type SetContainerDataCommand struct {
	service *containerservice.SetContainerDataService
}

func NewSetContainerDataCommand(service *containerservice.SetContainerDataService) *SetContainerDataCommand {
	return &SetContainerDataCommand{
		service: service,
	}
}

func (c *SetContainerDataCommand) Execute(ctx context.Context, nodeID string, token string) error {
	if c.service == nil {
		return fmt.Errorf("container service is not initialized")
	}

	log.Printf("[SetContainerDataCommand] Executing SetContainerData for node: %s", nodeID)

	return c.service.Execute(ctx, nodeID, token)
}

func (c *SetContainerDataCommand) Name() string {
	return "set_container_data"
}

func (c *SetContainerDataCommand) String() string {
	return "SetContainerDataCommand"
}

func (c *SetContainerDataCommand) IntervalSeconds() int {
	return 30 // Run every 30 seconds to collect container data
}

