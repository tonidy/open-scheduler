package commands

import (
	"context"
	"log"

	agentgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/service/heartbeat"
)

type HeartbeatCommand struct {
	service *heartbeat.HeartbeatService
}

func NewHeartbeatCommand(grpcClient *agentgrpc.GrpcClient) *HeartbeatCommand {
	service, err := heartbeat.NewHeartbeatService(grpcClient)
	if err != nil {
		log.Fatalf("[HeartbeatCommand] Failed to create service: %v", err)
	}
	return &HeartbeatCommand{
		service: service,
	}
}

func (h *HeartbeatCommand) Execute(ctx context.Context, nodeID string, token string) error {
	if h.service == nil {
		log.Printf("[HeartbeatCommand] Service not initialized, skipping heartbeat")
		return nil
	}

	return h.service.Execute(ctx, nodeID, token)
}

func (h *HeartbeatCommand) Name() string {
	return "heartbeat"
}

func (h *HeartbeatCommand) String() string {
	return "HeartbeatCommand"
}

func (h *HeartbeatCommand) IntervalSeconds() int {
	return 15
}
