package incus

import (
	"context"
	"fmt"

	"github.com/open-scheduler/agent/taskdriver/model"
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

func (d *IncusDriver) Run(ctx context.Context, task *pb.Task) error {
	// TODO: Implement Incus driver
	return nil
}

// StopContainer stops a running container (to be implemented)
func (d *IncusDriver) StopContainer(ctx context.Context, containerID string) error {
	// TODO: Implement Incus container stop
	return fmt.Errorf("StopContainer not implemented for Incus driver")
}

// GetContainerStatus retrieves the current status of a container (to be implemented)
func (d *IncusDriver) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	// TODO: Implement Incus status retrieval
	return "", fmt.Errorf("GetContainerStatus not implemented for Incus driver")
}

func (d *IncusDriver) InspectContainer(ctx context.Context, containerID string) (model.ContainerInspect, error) {
	// TODO: Implement Incus container inspection
	return model.ContainerInspect{}, fmt.Errorf("InspectContainer not implemented for Incus driver")
}

// ListContainers lists all containers (not yet implemented for incus driver)
func (d *IncusDriver) ListContainers(ctx context.Context) ([]model.ContainerInspect, error) {
	// TODO: Implement when incus driver is fully developed
	return []model.ContainerInspect{}, nil
}
