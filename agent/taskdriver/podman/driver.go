package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/containers/podman/v4/libpod/define"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/specgen"
	pb "github.com/open-scheduler/proto"
)

type PodmanConnection struct {
	Name      string `json:"Name"`
	URI       string `json:"URI"`
	Identity  string `json:"Identity"`
	IsMachine bool   `json:"IsMachine"`
	Default   bool   `json:"Default"`
}

type PodmanDriver struct {
	ctx context.Context
}

func NewPodmanDriver() *PodmanDriver {
	// Get Podman connection URI
	// Order of precedence:
	// 1. CONTAINER_HOST env var
	// 2. XDG_RUNTIME_DIR (Linux)
	// 3. Query podman for default connection (macOS)

	var connectionURI string
	var identityFile string

	if containerHost := os.Getenv("CONTAINER_HOST"); containerHost != "" {
		connectionURI = containerHost
		log.Printf("[PodmanDriver] Using CONTAINER_HOST: %s", connectionURI)
	} else if xdgDir := os.Getenv("XDG_RUNTIME_DIR"); xdgDir != "" {
		// Linux: use XDG_RUNTIME_DIR
		connectionURI = "unix:" + xdgDir + "/podman/podman.sock"
		log.Printf("[PodmanDriver] Using XDG_RUNTIME_DIR socket: %s", connectionURI)
	} else {
		// macOS or other: Get connection from podman system
		log.Printf("[PodmanDriver] Querying podman for default connection...")

		cmd := exec.Command("podman", "system", "connection", "list", "--format=json")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("failed to query podman connections: %v", err)
			log.Printf("Please set CONTAINER_HOST environment variable or ensure podman machine is running")
			return nil
		}

		var connections []PodmanConnection
		if err := json.Unmarshal(output, &connections); err != nil {
			log.Printf("failed to parse podman connections: %v", err)
			return nil
		}

		// Find default connection
		for _, conn := range connections {
			if conn.Default {
				connectionURI = conn.URI
				identityFile = conn.Identity
				log.Printf("[PodmanDriver] Using default connection: %s", connectionURI)
				break
			}
		}

		if connectionURI == "" {
			log.Printf("no default podman connection found")
			return nil
		}

		// Set SSH identity file if provided
		if identityFile != "" {
			os.Setenv("CONTAINER_SSHKEY", identityFile)
			log.Printf("[PodmanDriver] Using SSH identity: %s", identityFile)
		}
	}

	// Connect to Podman
	log.Printf("[PodmanDriver] Connecting to: %s", connectionURI)
	conn, err := bindings.NewConnection(context.Background(), connectionURI)
	if err != nil {
		log.Printf("failed to connect to Podman: %v", err)
		log.Printf("Make sure podman is running (Linux: 'podman system service', macOS: 'podman machine start')")
		return nil
	}

	log.Printf("[PodmanDriver] Successfully connected to Podman!")
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

// LogOptions configures log retrieval behavior for Podman
type LogOptions struct {
	Follow     bool   // Stream logs continuously
	Tail       string // Get last N lines (e.g., "100")
	Since      string // RFC3339 timestamp or relative (e.g., "1h")
	Until      string // RFC3339 timestamp
	Timestamps bool   // Include timestamps in output
	Stdout     bool   // Include stdout
	Stderr     bool   // Include stderr
}

// GetLogs retrieves container logs with specified options
// Returns two channels for stdout and stderr streams
func (d *PodmanDriver) GetLogs(ctx context.Context, containerID string, opts *LogOptions) (stdout, stderr chan string, err error) {
	// Create buffered channels for efficient streaming
	stdoutChan := make(chan string, 1000)
	stderrChan := make(chan string, 1000)

	// Build log options
	logOpts := &containers.LogOptions{
		Follow:     &opts.Follow,
		Timestamps: &opts.Timestamps,
		Stdout:     &opts.Stdout,
		Stderr:     &opts.Stderr,
	}

	// Set tail if specified
	if opts.Tail != "" {
		tail := opts.Tail
		logOpts.Tail = &tail
	}

	// Set time range if specified
	if opts.Since != "" {
		since := opts.Since
		logOpts.Since = &since
	}
	if opts.Until != "" {
		until := opts.Until
		logOpts.Until = &until
	}

	// Start log streaming in a goroutine
	go func() {
		defer close(stdoutChan)
		defer close(stderrChan)

		err := containers.Logs(d.ctx, containerID, logOpts, stdoutChan, stderrChan)
		if err != nil {
			log.Printf("[PodmanDriver] Error streaming logs for container %s: %v", containerID, err)
		}
	}()

	return stdoutChan, stderrChan, nil
}

// StopContainer stops a running container
func (d *PodmanDriver) StopContainer(ctx context.Context, containerID string) error {
	log.Printf("[PodmanDriver] Stopping container: %s", containerID)
	err := containers.Stop(d.ctx, containerID, &containers.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	log.Printf("[PodmanDriver] Container stopped: %s", containerID)
	return nil
}

// DeleteContainer removes a container
func (d *PodmanDriver) DeleteContainer(ctx context.Context, containerID string) error {
	log.Printf("[PodmanDriver] Deleting container: %s", containerID)
	_, err := containers.Remove(d.ctx, containerID, &containers.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete container %s: %w", containerID, err)
	}
	log.Printf("[PodmanDriver] Container deleted: %s", containerID)
	return nil
}

// GetContainerStatus retrieves the current status of a container
func (d *PodmanDriver) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	inspectData, err := containers.Inspect(d.ctx, containerID, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get container status for %s: %w", containerID, err)
	}
	return inspectData.State.Status, nil
}
