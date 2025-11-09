package status

import (
	"context"
	"fmt"
	"log"
	"time"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
)

type UpdateStatusService struct {
	grpcClient *sharedgrpc.SharedClient
	token string
	nodeID string
}

func NewUpdateStatusService(grpcClient *sharedgrpc.SharedClient, token string, nodeID string) (*UpdateStatusService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &UpdateStatusService{
		grpcClient: grpcClient,
		token: token,
		nodeID: nodeID,
	}, nil
}

func (s *UpdateStatusService) Execute(ctx context.Context, nodeID string, token string) error {
	log.Printf("[UpdateStatusService] Updating status for  node: %s",s.nodeID)

	resp, err := s.grpcClient.UpdateStatus(
		ctx,
		s.nodeID,
		s.token,
		"jobID",
		"status",
		"detail",
		time.Now().Unix(),
	)

	if err != nil {
		return fmt.Errorf("update status failed: %w", err)
	}

	if !resp.Ok {
		return fmt.Errorf("update status rejected: %s", resp.Message)
	}

	log.Printf("[UpdateStatusService] Status update successful: %s", resp.Message)
	return nil
}



