package commands

import (
	"context"
	"fmt"
	"log"
)

type HeartbeatCommand struct {	
}

func NewHeartbeatCommand() *HeartbeatCommand {
	return &HeartbeatCommand{
	}
}

func (h *HeartbeatCommand) Execute(ctx context.Context, token string, nodeID string) error {
	// TODO: Implement actual heartbeat logic using gRPC
	log.Printf("[HeartbeatCommand] Sending heartbeat for node: %s with token: %s", nodeID, token)
	return nil
}

func (h *HeartbeatCommand) Name() string {
	return "heartbeat"
}

func (h *HeartbeatCommand) String() string {
	return fmt.Sprintf("HeartbeatCommand")
}

func (h *HeartbeatCommand) IntervalSeconds() int {
	return 15
}