package podman

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/containers/podman/v4/libpod/define"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/specgen"
	pb "github.com/open-scheduler/proto"
)

type PodmanDriver struct {
	ctx context.Context
}

func NewPodmanDriver() *PodmanDriver {
	// Get Podman socket location
	sock_dir := os.Getenv("XDG_RUNTIME_DIR")
	socket := "unix:" + sock_dir + "/podman/podman.sock"

	// Connect to Podman socket
	conn, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		log.Default().Printf("failed to connect to Podman socket: %w", err)
		return nil
	}
	return &PodmanDriver{
		ctx: conn,
	}
}

func (d *PodmanDriver) Run(ctx context.Context, task *pb.Task) error {
	err := d.pullImage(ctx, task.Config.Image)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", task.Config.Image, err)
	}
	err = d.createContainer(ctx, task.Config.Image)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", task.Config.Image, err)
	}
	return nil
}

func (d *PodmanDriver) pullImage(ctx context.Context, image string) error {
	fmt.Println("Pulling image:", image)
	_, err := images.Pull(d.ctx, image, &images.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	return nil
}

func (d *PodmanDriver) createContainer(ctx context.Context, image string) error {

	s := specgen.NewSpecGenerator(image, false)
	s.Terminal = true

	r, err := containers.CreateWithSpec(d.ctx, s, &containers.CreateOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Container start
	fmt.Println("Starting container...")
	err = containers.Start(d.ctx, r.ID, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = containers.Wait(d.ctx, r.ID, &containers.WaitOptions{
		Condition: []define.ContainerStatus{define.ContainerStateRunning},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return nil
}

func (d *PodmanDriver) getContainerLogs(ctx context.Context, id string, stdoutChan, stderrChan chan string) error {
	containers.Logs(d.ctx, id, &containers.LogOptions{}, stdoutChan, stderrChan)
	return nil
}

func (d *PodmanDriver) stopContainer(ctx context.Context, id string) error {
	err := containers.Stop(d.ctx, id, &containers.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", id, err)
	}
	return nil
}

func (d *PodmanDriver) deleteContainer(ctx context.Context, id string) error {
	_, err := containers.Remove(d.ctx, id, &containers.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete container %s: %w", id, err)
	}
	return nil
}

func (d *PodmanDriver) getContainerStatus(ctx context.Context, id string) (string, error) {
	inspectData, err := containers.Inspect(d.ctx, id, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get container status for %s: %w", id, err)
	}
	return inspectData.State.Status, nil
}
