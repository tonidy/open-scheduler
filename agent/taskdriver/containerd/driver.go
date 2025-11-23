package containerd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	pb "github.com/open-scheduler/proto"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type ContainerdDriver struct {
	client *containerd.Client
	ctx    context.Context
}

func NewContainerdDriver() *ContainerdDriver {
	// Get containerd socket path
	// Order of precedence:
	// 1. CONTAINERD_ADDRESS env var
	// 2. Default paths: /run/containerd/containerd.sock (Linux) or /var/run/containerd/containerd.sock
	// 3. XDG_RUNTIME_DIR/containerd/containerd.sock (Linux)

	var socketPath string

	if addr := os.Getenv("CONTAINERD_ADDRESS"); addr != "" {
		socketPath = addr
		log.Printf("[ContainerdDriver] Using CONTAINERD_ADDRESS: %s", socketPath)
	} else if xdgDir := os.Getenv("XDG_RUNTIME_DIR"); xdgDir != "" {
		socketPath = filepath.Join(xdgDir, "containerd", "containerd.sock")
		log.Printf("[ContainerdDriver] Using XDG_RUNTIME_DIR socket: %s", socketPath)
	} else {
		// Try default paths
		defaultPaths := []string{
			"/run/containerd/containerd.sock",
			"/var/run/containerd/containerd.sock",
		}
		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				socketPath = path
				log.Printf("[ContainerdDriver] Using default socket: %s", socketPath)
				break
			}
		}
	}

	if socketPath == "" {
		log.Printf("[ContainerdDriver] No containerd socket found. Please set CONTAINERD_ADDRESS or ensure containerd is running")
		return nil
	}

	// Connect to containerd
	log.Printf("[ContainerdDriver] Connecting to containerd at: %s", socketPath)
	client, err := containerd.New(socketPath)
	if err != nil {
		log.Printf("[ContainerdDriver] Failed to connect to containerd: %v", err)
		log.Printf("[ContainerdDriver] Make sure containerd is running")
		return nil
	}

	log.Printf("[ContainerdDriver] Successfully connected to containerd!")
	return &ContainerdDriver{
		client: client,
		ctx:    namespaces.WithNamespace(context.Background(), "default"),
	}
}

func (d *ContainerdDriver) Run(ctx context.Context, job *pb.Job) (string, error) {
	err := d.pullImage(ctx, job.InstanceConfig.ImageName)
	if err != nil {
		return "", fmt.Errorf("failed to pull image %s: %w", job.InstanceConfig.ImageName, err)
	}

	id, err := d.createInstance(ctx, job)
	if err != nil {
		return "", fmt.Errorf("failed to create instance %s: %w", job.InstanceConfig.ImageName, err)
	}

	return id, nil
}

func (d *ContainerdDriver) pullImage(ctx context.Context, image string) error {
	log.Printf("[ContainerdDriver] Pulling image: %s", image)
	img, err := d.client.Pull(ctx, image, containerd.WithPullUnpack)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	log.Printf("[ContainerdDriver] Successfully pulled image: %s (digest: %s)", image, img.Target().Digest)
	return nil
}

func (d *ContainerdDriver) createInstance(ctx context.Context, job *pb.Job) (string, error) {
	// Get image
	image, err := d.client.GetImage(ctx, job.InstanceConfig.ImageName)
	if err != nil {
		return "", fmt.Errorf("failed to get image %s: %w", job.InstanceConfig.ImageName, err)
	}

	// Generate container ID from job ID
	containerID := fmt.Sprintf("open-scheduler-%s", job.JobId)

	// Create container spec
	opts := []oci.SpecOpts{
		oci.WithImageConfig(image),
		oci.WithTTY,
	}

	// Set command and args if specified
	if len(job.InstanceConfig.Entrypoint) > 0 {
		cmd := job.InstanceConfig.Entrypoint
		if len(job.InstanceConfig.Arguments) > 0 {
			cmd = append(cmd, job.InstanceConfig.Arguments...)
		}
		opts = append(opts, oci.WithProcessArgs(cmd...))
	}

	// Set environment variables
	if len(job.EnvironmentVariables) > 0 {
		envVars := make([]string, 0, len(job.EnvironmentVariables))
		for k, v := range job.EnvironmentVariables {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		opts = append(opts, oci.WithEnv(envVars))
	}

	// Set resource limits
	if job.ResourceRequirements != nil {
		// Memory limits (convert MB to bytes)
		if job.ResourceRequirements.MemoryLimitMb > 0 {
			memoryLimit := int64(job.ResourceRequirements.MemoryLimitMb * 1024 * 1024)
			opts = append(opts, oci.WithMemoryLimit(uint64(memoryLimit)))
			// Set memory reservation if specified
			if job.ResourceRequirements.MemoryReservedMb > 0 {
				memoryReserve := int64(job.ResourceRequirements.MemoryReservedMb * 1024 * 1024)
				opts = append(opts, func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
					if s.Linux == nil {
						s.Linux = &specs.Linux{}
					}
					if s.Linux.Resources == nil {
						s.Linux.Resources = &specs.LinuxResources{}
					}
					if s.Linux.Resources.Memory == nil {
						s.Linux.Resources.Memory = &specs.LinuxMemory{}
					}
					s.Linux.Resources.Memory.Reservation = &memoryReserve
					return nil
				})
			}
		}

		// CPU limits
		if job.ResourceRequirements.CpuLimitCores > 0 {
			// Convert CPU cores to quota/period
			// Default period is 100000 microseconds (0.1s)
			period := uint64(100000)
			quota := int64(job.ResourceRequirements.CpuLimitCores * float32(period))
			opts = append(opts, func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
				if s.Linux == nil {
					s.Linux = &specs.Linux{}
				}
				if s.Linux.Resources == nil {
					s.Linux.Resources = &specs.LinuxResources{}
				}
				if s.Linux.Resources.CPU == nil {
					s.Linux.Resources.CPU = &specs.LinuxCPU{}
				}
				s.Linux.Resources.CPU.Quota = &quota
				s.Linux.Resources.CPU.Period = &period
				return nil
			})
		}
	}

	// Set volume mounts
	if len(job.VolumeMounts) > 0 {
		for _, vol := range job.VolumeMounts {
			mountOpts := []string{"rbind"}
			if vol.ReadOnly {
				mountOpts = append(mountOpts, "ro")
			} else {
				mountOpts = append(mountOpts, "rw")
			}
			opts = append(opts, oci.WithMounts([]specs.Mount{
				{
					Type:        "bind",
					Source:      vol.SourcePath,
					Destination: vol.TargetPath,
					Options:     mountOpts,
				},
			}))
		}
	}

	// Create container spec
	specOpts := append(opts, oci.WithImageConfig(image))
	spec, err := oci.GenerateSpec(d.ctx, nil, &containers.Container{
		ID: containerID,
	}, specOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to generate spec: %w", err)
	}

	// Add labels
	if spec.Annotations == nil {
		spec.Annotations = make(map[string]string)
	}
	spec.Annotations["open-scheduler.managed"] = "true"
	spec.Annotations["open-scheduler.job-name"] = job.JobName
	spec.Annotations["open-scheduler.job-id"] = job.JobId

	// Create container
	log.Printf("[ContainerdDriver] Creating container for job %s with image: %s", job.JobId, job.InstanceConfig.ImageName)
	container, err := d.client.NewContainer(
		ctx,
		containerID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(containerID+"-snapshot", image),
		containerd.WithNewSpec(specOpts...),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Create task
	log.Printf("[ContainerdDriver] Creating task for container: %s", containerID)
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		container.Delete(ctx)
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	// Start task
	log.Printf("[ContainerdDriver] Starting task: %s", containerID)
	if err := task.Start(ctx); err != nil {
		task.Delete(ctx)
		container.Delete(ctx)
		return "", fmt.Errorf("failed to start task: %w", err)
	}

	log.Printf("[ContainerdDriver] Container is running: %s", containerID)
	return containerID, nil
}

func (d *ContainerdDriver) StopInstance(ctx context.Context, instanceID string) error {
	log.Printf("[ContainerdDriver] Stopping instance: %s", instanceID)
	container, err := d.client.LoadContainer(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", instanceID, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		// Task might not exist, try to delete container directly
		log.Printf("[ContainerdDriver] Task not found, deleting container: %s", instanceID)
		if err := container.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete container %s: %w", instanceID, err)
		}
		log.Printf("[ContainerdDriver] Instance stopped: %s", instanceID)
		return nil
	}

	// Stop task
	if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
		log.Printf("[ContainerdDriver] Warning: failed to kill task: %v", err)
	}

	// Wait for task to exit
	status, err := task.Wait(ctx)
	if err == nil {
		<-status
	}

	// Delete task
	if _, err := task.Delete(ctx); err != nil {
		log.Printf("[ContainerdDriver] Warning: failed to delete task: %v", err)
	}

	// Delete container
	if err := container.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete container %s: %w", instanceID, err)
	}

	log.Printf("[ContainerdDriver] Instance stopped: %s", instanceID)
	return nil
}

func (d *ContainerdDriver) RestartInstance(ctx context.Context, instanceID string) error {
	log.Printf("[ContainerdDriver] Restarting instance: %s", instanceID)
	container, err := d.client.LoadContainer(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", instanceID, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get task for container %s: %w", instanceID, err)
	}

	// Kill existing task
	if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to kill task: %w", err)
	}

	// Wait for task to exit
	status, err := task.Wait(ctx)
	if err == nil {
		<-status
	}

	// Delete old task
	if _, err := task.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete old task: %w", err)
	}

	// Create new task
	newTask, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return fmt.Errorf("failed to create new task: %w", err)
	}

	// Start new task
	if err := newTask.Start(ctx); err != nil {
		newTask.Delete(ctx)
		return fmt.Errorf("failed to start new task: %w", err)
	}

	log.Printf("[ContainerdDriver] Instance restarted: %s", instanceID)
	return nil
}

func (d *ContainerdDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	container, err := d.client.LoadContainer(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to load container %s: %w", instanceID, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		// Task doesn't exist, container is stopped
		return "stopped", nil
	}

	status, err := task.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get task status: %w", err)
	}

	switch status.Status {
	case containerd.Running:
		return "running", nil
	case containerd.Created:
		return "created", nil
	case containerd.Stopped:
		return "stopped", nil
	case containerd.Paused:
		return "paused", nil
	case containerd.Pausing:
		return "pausing", nil
	default:
		return string(status.Status), nil
	}
}

func (d *ContainerdDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	container, err := d.client.LoadContainer(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load container %s: %w", instanceID, err)
	}

	info, err := container.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	// Get task status
	task, err := container.Task(ctx, nil)
	var status string
	var pid uint32
	var exitCode uint32
	var startedAt, finishedAt time.Time

	if err != nil {
		status = "stopped"
	} else {
		taskStatus, err := task.Status(ctx)
		if err != nil {
			status = "unknown"
		} else {
			status = string(taskStatus.Status)
			// Get PID from task
			pid = task.Pid()
		}
	}

	// Get labels
	labels := make(map[string]string)
	if info.Labels != nil {
		labels = info.Labels
	}

	// Extract command and volumes from spec
	var command []string
	volumes := make([]string, 0)

	spec, err := container.Spec(ctx)
	if err == nil {
		if spec.Process != nil {
			command = spec.Process.Args
		}
		for _, mount := range spec.Mounts {
			volumes = append(volumes, mount.Destination)
		}
	}

	// Get image name
	imageName := info.Image
	if imageName == "" {
		imageName = "unknown"
	}

	return &pb.InstanceData{
		InstanceId:   instanceID,
		InstanceName: info.ID,
		Image:        imageName,
		ImageName:    imageName,
		Command:      command,
		Args:         []string{},
		Created:      info.CreatedAt.Format(time.RFC3339Nano),
		StartedAt:    startedAt.Format(time.RFC3339Nano),
		FinishedAt:   finishedAt.Format(time.RFC3339Nano),
		Status:       status,
		ExitCode:     int32(exitCode),
		Pid:          int32(pid),
		Labels:       labels,
		Ports:        []string{},
		Volumes:      volumes,
	}, nil
}

func (d *ContainerdDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	log.Printf("[ContainerdDriver] Listing all instances")

	containerList, err := d.client.Containers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]*pb.InstanceData, 0)

	for _, container := range containerList {
		info, err := container.Info(ctx)
		if err != nil {
			log.Printf("[ContainerdDriver] Warning: failed to get info for container %s: %v", container.ID(), err)
			continue
		}

		// Check if container is managed by open-scheduler
		if info.Labels != nil {
			if managed, ok := info.Labels["open-scheduler.managed"]; ok && managed == "true" {
				inspectData, err := d.InspectInstance(ctx, container.ID())
				if err != nil {
					log.Printf("[ContainerdDriver] Warning: failed to inspect instance %s: %v", container.ID(), err)
					continue
				}
				result = append(result, inspectData)
			}
		}
	}

	log.Printf("[ContainerdDriver] Found %d managed instances", len(result))
	return result, nil
}

func (d *ContainerdDriver) DeleteInstance(ctx context.Context, instanceID string) error {
	log.Printf("[ContainerdDriver] Deleting instance: %s", instanceID)
	container, err := d.client.LoadContainer(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", instanceID, err)
	}

	// Try to stop task if it exists
	task, err := container.Task(ctx, nil)
	if err == nil {
		task.Kill(ctx, syscall.SIGKILL)
		task.Delete(ctx)
	}

	// Delete container
	if err := container.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete container %s: %w", instanceID, err)
	}

	log.Printf("[ContainerdDriver] Instance deleted: %s", instanceID)
	return nil
}
