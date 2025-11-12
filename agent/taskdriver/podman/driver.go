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
	spec "github.com/opencontainers/runtime-spec/specs-go"
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

func (d *PodmanDriver) Run(ctx context.Context, job *pb.Job) error {
	err := d.pullImage(ctx, job.ContainerConfig.ImageName)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", job.ContainerConfig.ImageName, err)
	}

	err = d.createContainer(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", job.ContainerConfig.ImageName, err)
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

func (d *PodmanDriver) createContainer(ctx context.Context, job *pb.Job) error {
	s := specgen.NewSpecGenerator(job.ContainerConfig.ImageName, false)
	s.Terminal = true

	// Set command and args if specified
	if len(job.ContainerConfig.Entrypoint) > 0 {
		s.Command = job.ContainerConfig.Entrypoint
	}
	if len(job.ContainerConfig.Arguments) > 0 {
		s.Command = append(s.Command, job.ContainerConfig.Arguments...)
	}

	labels := make(map[string]string)
	labels["open-scheduler.managed"] = "true"
	labels["open-scheduler.job-name"] = job.JobName
	labels["open-scheduler.job-id"] = job.JobId

	s.Labels = labels

	if len(job.EnvironmentVariables) > 0 {
		envVars := make(map[string]string)
		for k, v := range job.EnvironmentVariables {
			envVars[k] = v
		}
		s.Env = envVars
	}

	// Set resource limits
	if job.ResourceRequirements != nil {
		// Memory limits (convert MB to bytes)
		if job.ResourceRequirements.MemoryLimitMb > 0 {
			memoryLimit := job.ResourceRequirements.MemoryLimitMb * 1024 * 1024
			s.ResourceLimits = &spec.LinuxResources{}
			s.ResourceLimits.Memory = &spec.LinuxMemory{
				Limit: &memoryLimit,
			}
			// Set memory reservation if specified
			if job.ResourceRequirements.MemoryReservedMb > 0 {
				memoryReserve := job.ResourceRequirements.MemoryReservedMb * 1024 * 1024
				s.ResourceLimits.Memory.Reservation = &memoryReserve
			}
		}

		// CPU limits
		if job.ResourceRequirements.CpuLimitCores > 0 {
			if s.ResourceLimits == nil {
				s.ResourceLimits = &spec.LinuxResources{}
			}
			// Convert CPU float to quota/period
			// Default period is 100000 microseconds (0.1s)
			period := uint64(100000)
			quota := int64(job.ResourceRequirements.CpuLimitCores * float32(period))
			s.ResourceLimits.CPU = &spec.LinuxCPU{
				Quota:  &quota,
				Period: &period,
			}
		}
	}

	// Set volume mounts
	if len(job.VolumeMounts) > 0 {
		mounts := make([]spec.Mount, 0, len(job.VolumeMounts))
		for _, vol := range job.VolumeMounts {
			mountType := "bind"
			mountOpts := []string{"rbind"}
			if vol.ReadOnly {
				mountOpts = append(mountOpts, "ro")
			} else {
				mountOpts = append(mountOpts, "rw")
			}

			mounts = append(mounts, spec.Mount{
				Type:        mountType,
				Source:      vol.SourcePath,
				Destination: vol.TargetPath,
				Options:     mountOpts,
			})
		}
		s.Mounts = mounts
	}

	// Create container with spec
	log.Printf("[PodmanDriver] Creating container for job %s with image: %s", job.JobId, job.ContainerConfig.ImageName)
	r, err := containers.CreateWithSpec(d.ctx, s, &containers.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Container start
	log.Printf("[PodmanDriver] Starting container: %s", r.ID)
	err = containers.Start(d.ctx, r.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container to be running
	log.Printf("[PodmanDriver] Waiting for container to be running: %s", r.ID)
	_, err = containers.Wait(d.ctx, r.ID, &containers.WaitOptions{
		Condition: []define.ContainerStatus{define.ContainerStateRunning},
	})
	if err != nil {
		return fmt.Errorf("container wait failed: %w", err)
	}

	log.Printf("[PodmanDriver] Container is running: %s", r.ID)
	return nil
}

// StopContainer stops a running container
func (d *PodmanDriver) StopContainer(ctx context.Context, containerID string) error {
	log.Printf("[PodmanDriver] Stopping container: %s", containerID)
	force := true
	err := containers.Stop(d.ctx, containerID, &containers.StopOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	containers.Remove(d.ctx, containerID, &containers.RemoveOptions{
		Force: &force,
	})
	log.Printf("[PodmanDriver] Container stopped: %s", containerID)
	return nil
}

// RestartContainer restarts a running container
func (d *PodmanDriver) RestartContainer(ctx context.Context, containerID string) error {
	log.Printf("[PodmanDriver] Restarting container: %s", containerID)
	err := containers.Restart(d.ctx, containerID, &containers.RestartOptions{})
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}
	log.Printf("[PodmanDriver] Container restarted: %s", containerID)
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

// InspectContainer inspects a container
func (d *PodmanDriver) InspectContainer(ctx context.Context, containerID string) (*pb.ContainerData, error) {
	inspectData, err := containers.Inspect(d.ctx, containerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	// Extract volume/mount information
	volumes := make([]string, 0, len(inspectData.Mounts))
	for _, mount := range inspectData.Mounts {
		volumes = append(volumes, mount.Destination)
	}

	// Extract port information
	ports := make([]string, 0)
	if inspectData.NetworkSettings != nil && len(inspectData.NetworkSettings.Ports) > 0 {
		for port := range inspectData.NetworkSettings.Ports {
			ports = append(ports, port)
		}
	}

	// Extract command from Config
	command := make([]string, 0)
	if inspectData.Config != nil {
		command = inspectData.Config.Cmd
	}

	// Extract labels from Config
	labels := make(map[string]string)
	if inspectData.Config != nil && inspectData.Config.Labels != nil {
		labels = inspectData.Config.Labels
	}

	return &pb.ContainerData{
		ContainerId:   inspectData.ID,
		ContainerName: inspectData.Name,
		Image:         inspectData.Image,
		ImageName:     inspectData.ImageName,
		Command:       command,
		Args:          inspectData.Args,
		Created:       inspectData.Created.Format("2006-01-02T15:04:05.999999999Z07:00"),
		StartedAt:     inspectData.State.StartedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		FinishedAt:    inspectData.State.FinishedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Status:        inspectData.State.Status,
		ExitCode:      int32(inspectData.State.ExitCode),
		Pid:           int32(inspectData.State.Pid),
		Labels:        labels,
		Ports:         ports,
		Volumes:       volumes,
	}, nil
}

func (d *PodmanDriver) ListContainers(ctx context.Context) ([]*pb.ContainerData, error) {
	log.Printf("[PodmanDriver] Listing all containers")

	allContainers := true
	containerList, err := containers.List(d.ctx, &containers.ListOptions{
		All: &allContainers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]*pb.ContainerData, 0)

	for _, container := range containerList {
		if container.Labels != nil {
			if managed, ok := container.Labels["open-scheduler.managed"]; ok && managed == "true" {
				inspectData, err := d.InspectContainer(ctx, container.ID)
				if err != nil {
					log.Printf("[PodmanDriver] Warning: failed to inspect container %s: %v", container.ID, err)
					continue
				}
				result = append(result, inspectData)
			}
		}
	}

	log.Printf("[PodmanDriver] Found %d managed containers", len(result))
	return result, nil
}
