package taskdriver

import (
	"context"
	"fmt"

	"github.com/open-scheduler/agent/taskdriver/process"
	"github.com/open-scheduler/agent/taskdriver/incus"
	"github.com/open-scheduler/agent/taskdriver/podman"
	pb "github.com/open-scheduler/proto"
)

type Driver interface {
	Run(ctx context.Context, job *pb.Job) (string, error)
	StopInstance(ctx context.Context, instanceID string) error
	RestartInstance(ctx context.Context, instanceID string) error
	GetInstanceStatus(ctx context.Context, instanceID string) (string, error)
	InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error)
	ListInstances(ctx context.Context) ([]*pb.InstanceData, error)
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
	case "process":
		// Direct shell command execution
		driver := process.NewProcessDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create process driver")
		}
		return driver, nil
	default:
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
}
