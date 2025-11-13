package commands

import (
	"context"
	"log"

	agentgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/service/job"
	"github.com/open-scheduler/agent/service/instance"
)

type GetJobCommand struct {
	service *job.GetJobService
}

func NewGetJobCommand(grpcClient *agentgrpc.GrpcClient, instanceService *instance.SetInstanceDataService) *GetJobCommand {
	service, err := job.NewGetJobService(grpcClient, instanceService)
	if err != nil {
		log.Fatalf("[GetJobCommand] Failed to create service: %v", err)
	}
	return &GetJobCommand{
		service: service,
	}
}

func (g *GetJobCommand) Execute(ctx context.Context, nodeID string, token string) error {
	return g.service.Execute(ctx, nodeID, token)
}

func (g *GetJobCommand) Name() string {
	return "get_job"
}

func (g *GetJobCommand) String() string {
	return "GetJobCommand"
}

func (g *GetJobCommand) IntervalSeconds() int {
	return 15
}
