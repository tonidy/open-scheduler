package taskdriver

import (
	"context"
	"fmt"

	"github.com/open-scheduler/agent/taskdriver/exec"
	"github.com/open-scheduler/agent/taskdriver/incus"
	"github.com/open-scheduler/agent/taskdriver/model"
	"github.com/open-scheduler/agent/taskdriver/podman"
	pb "github.com/open-scheduler/proto"
)

type Driver interface {
	Run(ctx context.Context, task *pb.Task) error
	StopContainer(ctx context.Context, containerID string) error
	GetContainerStatus(ctx context.Context, containerID string) (string, error)
	InspectContainer(ctx context.Context, containerID string) (model.ContainerInspect, error)
	ListContainers(ctx context.Context) ([]model.ContainerInspect, error)
}

func NewDriver(name string) (Driver, error) {
	switch name {
	case "podman":
		// Import locally to avoid import cycle
		driver := podman.NewPodmanDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create podman driver")
		}
		return driver, nil
	case "incus":
		// Import locally to avoid import cycle
		driver := incus.NewIncusDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create incus driver")
		}
		return driver, nil
	case "exec":
		// Direct shell command execution
		driver := exec.NewExecDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create exec driver")
		}
		return driver, nil
	default:
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
}
