package commands

import (
	"context"
	"fmt"
	"log"

	statusservice "github.com/open-scheduler/agent/service/status"
)

type UpdateStatusCommand struct {
	service *statusservice.UpdateStatusService	
}

func NewUpdateStatusCommand(service *statusservice.UpdateStatusService) *UpdateStatusCommand {
	return &UpdateStatusCommand{
		service: service,		
	}
}

func (u *UpdateStatusCommand) Execute(ctx context.Context, nodeID string, token string) error {
	if u.service == nil {
		return fmt.Errorf("status service is not initialized")
	}

	log.Printf("[UpdateStatusCommand] Executing UpdateStatus for node: %s", nodeID)


	return u.service.Execute(ctx, nodeID, token)
}

func (u *UpdateStatusCommand) Name() string {
	return "update_status"
}

func (u *UpdateStatusCommand) String() string {
	return "UpdateStatusCommand"
}

func (u *UpdateStatusCommand) IntervalSeconds() int {
	return 15
}
