package taskdriver

import (
	"context"
	"fmt"

	"github.com/open-scheduler/pkg/taskdriver/incus"
	"github.com/open-scheduler/pkg/taskdriver/podman"
	pb "github.com/open-scheduler/proto"
)

type Driver interface {
	Run(ctx context.Context, task *pb.Task) error
}

func NewDriver(name string) (Driver, error) {
	switch name {
	case "podman":
		driver := podman.NewPodmanDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create podman driver")
		}
		return driver, nil
	case "incus":
		driver := incus.NewIncusDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create incus driver")
		}
		return driver, nil
	default:
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
}
