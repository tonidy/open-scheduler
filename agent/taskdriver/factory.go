package taskdriver

import (
	"context"

	"github.com/open-scheduler/agent/taskdriver/incus"
	"github.com/open-scheduler/agent/taskdriver/podman"
	pb "github.com/open-scheduler/proto"
)

// Factory functions to avoid import cycles

// PodmanDriverWrapper wraps the podman driver to implement the Driver interface
type PodmanDriverWrapper struct {
	driver *podman.PodmanDriver
}

func createPodmanDriver() Driver {
	return &PodmanDriverWrapper{driver: podman.NewPodmanDriver()}
}

func (w *PodmanDriverWrapper) Run(ctx context.Context, task *pb.Task) error {
	return w.driver.Run(ctx, task)
}

func (w *PodmanDriverWrapper) GetLogs(ctx context.Context, containerID string, opts *LogOptions) (stdout, stderr chan string, err error) {
	// Convert LogOptions to podman.LogOptions
	podmanOpts := &podman.LogOptions{
		Follow:     opts.Follow,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Until:      opts.Until,
		Timestamps: opts.Timestamps,
		Stdout:     opts.Stdout,
		Stderr:     opts.Stderr,
	}
	return w.driver.GetLogs(ctx, containerID, podmanOpts)
}

func (w *PodmanDriverWrapper) StopContainer(ctx context.Context, containerID string) error {
	return w.driver.StopContainer(ctx, containerID)
}

func (w *PodmanDriverWrapper) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	return w.driver.GetContainerStatus(ctx, containerID)
}

// IncusDriverWrapper wraps the incus driver to implement the Driver interface
type IncusDriverWrapper struct {
	driver *incus.IncusDriver
}

func createIncusDriver() Driver {
	return &IncusDriverWrapper{driver: incus.NewIncusDriver()}
}

func (w *IncusDriverWrapper) Run(ctx context.Context, task *pb.Task) error {
	return w.driver.Run(ctx, task)
}

func (w *IncusDriverWrapper) GetLogs(ctx context.Context, containerID string, opts *LogOptions) (stdout, stderr chan string, err error) {
	// Convert LogOptions to incus.LogOptions
	incusOpts := &incus.LogOptions{
		Follow:     opts.Follow,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Until:      opts.Until,
		Timestamps: opts.Timestamps,
		Stdout:     opts.Stdout,
		Stderr:     opts.Stderr,
	}
	return w.driver.GetLogs(ctx, containerID, incusOpts)
}

func (w *IncusDriverWrapper) StopContainer(ctx context.Context, containerID string) error {
	return w.driver.StopContainer(ctx, containerID)
}

func (w *IncusDriverWrapper) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	return w.driver.GetContainerStatus(ctx, containerID)
}

