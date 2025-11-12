package commands

import (
	"context"
	"fmt"
	"log"

	instanceservice "github.com/open-scheduler/agent/service/instance"
)

type SetInstanceDataCommand struct {
	service *instanceservice.SetInstanceDataService
}

func NewSetInstanceDataCommand(service *instanceservice.SetInstanceDataService) *SetInstanceDataCommand {
	return &SetInstanceDataCommand{
		service: service,
	}
}

func (c *SetInstanceDataCommand) Execute(ctx context.Context, nodeID string, token string) error {
	if c.service == nil {
		return fmt.Errorf("instance service is not initialized")
	}

	log.Printf("[SetInstanceDataCommand] Executing SetInstanceData for node: %s", nodeID)

	return c.service.Execute(ctx, nodeID, token)
}

func (c *SetInstanceDataCommand) Name() string {
	return "set_instance_data"
}

func (c *SetInstanceDataCommand) String() string {
	return "SetInstanceDataCommand"
}

func (c *SetInstanceDataCommand) IntervalSeconds() int {
	return 30 // Run every 30 seconds to collect instance data
}

