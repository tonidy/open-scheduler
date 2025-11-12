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

// StopContainer stops a running container (to be implemented)
func (d *IncusDriver) StopContainer(ctx context.Context, containerID string) error {
	// TODO: Implement Incus container stop
	return fmt.Errorf("StopContainer not implemented for Incus driver")
}

// RestartContainer restarts a running container (to be implemented)
func (d *IncusDriver) RestartContainer(ctx context.Context, containerID string) error {
	// TODO: Implement Incus container restart
	return fmt.Errorf("RestartContainer not implemented for Incus driver")
}

// GetContainerStatus retrieves the current status of a container (to be implemented)
func (d *IncusDriver) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	// TODO: Implement Incus status retrieval
	return "", fmt.Errorf("GetContainerStatus not implemented for Incus driver")
}

func (d *IncusDriver) InspectContainer(ctx context.Context, containerID string) (*pb.ContainerData, error) {
	return nil, fmt.Errorf("InspectContainer not implemented for Incus driver")
}

func (d *IncusDriver) ListContainers(ctx context.Context) ([]*pb.ContainerData, error) {
	return []*pb.ContainerData{}, nil
}
