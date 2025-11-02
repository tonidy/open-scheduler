package commands

import (
	"context"
	"fmt"
	"log"
)

type GetJobCommand struct {	
}

func NewGetJobCommand() *GetJobCommand {
	return &GetJobCommand{
	}
}

func (g *GetJobCommand) Execute(ctx context.Context, token string, nodeID string) error {	
	log.Printf("[GetJobCommand] Requesting job for node: %s with token: %s", nodeID, token)
	return nil
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