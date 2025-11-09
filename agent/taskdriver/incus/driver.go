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

func (d *IncusDriver) Run(ctx context.Context, task *pb.Task) error {
	// TODO: Implement Incus driver
	return nil
}

// LogOptions configures log retrieval behavior for Incus
type LogOptions struct {
	Follow     bool   // Stream logs continuously
	Tail       string // Get last N lines (e.g., "100")
	Since      string // RFC3339 timestamp or relative (e.g., "1h")
	Until      string // RFC3339 timestamp
	Timestamps bool   // Include timestamps in output
	Stdout     bool   // Include stdout
	Stderr     bool   // Include stderr
}

// GetLogs retrieves container logs (to be implemented)
func (d *IncusDriver) GetLogs(ctx context.Context, containerID string, opts *LogOptions) (stdout, stderr chan string, err error) {
	// TODO: Implement Incus log retrieval
	return nil, nil, fmt.Errorf("GetLogs not implemented for Incus driver")
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
