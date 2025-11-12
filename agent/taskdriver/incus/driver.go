package incus

import (
	"context"
	"fmt"

	pb "github.com/open-scheduler/proto"
)

type IncusDriver struct {
	ctx context.Context
}

func NewIncusDriver() *IncusDriver {
	return &IncusDriver{
		ctx: context.Background(),
	}
}

func (d *IncusDriver) Run(ctx context.Context, job *pb.Job) error {
	// TODO: Implement Incus driver
	return nil
}

// StopInstance stops a running instance (to be implemented)
func (d *IncusDriver) StopInstance(ctx context.Context, instanceID string) error {
	// TODO: Implement Incus instance stop
	return fmt.Errorf("StopInstance not implemented for Incus driver")
}

// RestartInstance restarts a running instance (to be implemented)
func (d *IncusDriver) RestartInstance(ctx context.Context, instanceID string) error {
	// TODO: Implement Incus instance restart
	return fmt.Errorf("RestartInstance not implemented for Incus driver")
}

// GetInstanceStatus retrieves the current status of an instance (to be implemented)
func (d *IncusDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	// TODO: Implement Incus status retrieval
	return "", fmt.Errorf("GetInstanceStatus not implemented for Incus driver")
}

func (d *IncusDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	return nil, fmt.Errorf("InspectInstance not implemented for Incus driver")
}

func (d *IncusDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	return []*pb.InstanceData{}, nil
}
