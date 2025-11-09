package commands

import (
	"context"
	"fmt"
	"log"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
	"github.com/open-scheduler/agent/service/job"
)

type GetJobCommand struct {		
	service *job.GetJobService	
}

func NewGetJobCommand(grpcClient *sharedgrpc.SharedClient) *GetJobCommand {
	service, err := job.NewGetJobService(grpcClient)
	if err != nil {
		log.Fatalf("[GetJobCommand] Failed to create service: %v", err)
	}
	return &GetJobCommand{
		service: service,
	}
}

func (g *GetJobCommand) Execute(ctx context.Context, nodeID string, token string) error {	
	log.Printf("[GetJobCommand] Executing GetJob for node: %s", nodeID)
	return g.service.Execute(ctx, nodeID, token)
}

func (g *GetJobCommand) Name() string {
	return "get_job"
}

func (g *GetJobCommand) String() string {
	return fmt.Sprintf("GetJobCommand")
}

func (g *GetJobCommand) IntervalSeconds() int {
	return 15
}