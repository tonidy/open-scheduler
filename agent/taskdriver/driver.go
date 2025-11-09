package taskdriver

import (
	"context"
	"fmt"

	pb "github.com/open-scheduler/proto"
)

// LogOptions configures log retrieval behavior
type LogOptions struct {
	Follow     bool   // Stream logs continuously
	Tail       string // Get last N lines (e.g., "100")
	Since      string // RFC3339 timestamp or relative (e.g., "1h")
	Until      string // RFC3339 timestamp
	Timestamps bool   // Include timestamps in output
	Stdout     bool   // Include stdout
	Stderr     bool   // Include stderr
}

type Driver interface {
	Run(ctx context.Context, task *pb.Task) error
	GetLogs(ctx context.Context, containerID string, opts *LogOptions) (stdout, stderr chan string, err error)
	StopContainer(ctx context.Context, containerID string) error
	GetContainerStatus(ctx context.Context, containerID string) (string, error)
}

func NewDriver(name string) (Driver, error) {
	switch name {
	case "podman":
		// Import locally to avoid import cycle
		driver := createPodmanDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create podman driver")
		}
		return driver, nil
	case "incus":
		// Import locally to avoid import cycle
		driver := createIncusDriver()
		if driver == nil {
			return nil, fmt.Errorf("failed to create incus driver")
		}
		return driver, nil
	default:
		return nil, fmt.Errorf("unknown driver: %s", name)
	}
}
