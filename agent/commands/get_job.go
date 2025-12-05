package commands

import (
	"context"
	"log"

	agentgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/service/job"
	"github.com/open-scheduler/agent/service/instance"
)

type GetDeploymentCommand struct {
	service *job.GetDeploymentService
}

func NewGetDeploymentCommand(grpcClient *agentgrpc.GrpcClient, instanceService *instance.SetInstanceDataService) *GetDeploymentCommand {
	service, err := job.NewGetDeploymentService(grpcClient, instanceService)
	if err != nil {
		log.Fatalf("[GetDeploymentCommand] Failed to create service: %v", err)
	}
	return &GetDeploymentCommand{
		service: service,
	}
}

func (g *GetDeploymentCommand) Execute(ctx context.Context, nodeID string, token string) error {
	return g.service.Execute(ctx, nodeID, token)
}

func (g *GetDeploymentCommand) Name() string {
	return "get_deployment"
}

func (g *GetDeploymentCommand) String() string {
	return "GetDeploymentCommand"
}

func (g *GetDeploymentCommand) IntervalSeconds() int {
	return 15
}
