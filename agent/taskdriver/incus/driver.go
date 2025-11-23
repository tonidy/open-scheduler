package incus

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	incusclient "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
	pb "github.com/open-scheduler/proto"
)

type IncusDriver struct {
	server incusclient.InstanceServer
	ctx    context.Context
}

func NewIncusDriver() *IncusDriver {
	// Get Incus socket path
	// Order of precedence:
	// 1. INCUS_SOCKET env var
	// 2. Default paths: /var/snap/lxd/common/lxd/unix.socket (snap) or /var/lib/lxd/unix.socket
	// 3. XDG_RUNTIME_DIR/lxd/unix.socket (user mode)

	var socketPath string

	if addr := os.Getenv("INCUS_SOCKET"); addr != "" {
		socketPath = addr
		log.Printf("[IncusDriver] Using INCUS_SOCKET: %s", socketPath)
	} else if xdgDir := os.Getenv("XDG_RUNTIME_DIR"); xdgDir != "" {
		socketPath = filepath.Join(xdgDir, "lxd", "unix.socket")
		log.Printf("[IncusDriver] Using XDG_RUNTIME_DIR socket: %s", socketPath)
	} else {
		// Try default paths
		defaultPaths := []string{
			"/var/snap/lxd/common/lxd/unix.socket", // Snap installation
			"/var/lib/lxd/unix.socket",             // System installation
			"/var/run/lxd.socket",                  // Alternative path
		}
		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				socketPath = path
				log.Printf("[IncusDriver] Using default socket: %s", socketPath)
				break
			}
		}
	}

	if socketPath == "" {
		log.Printf("[IncusDriver] No Incus socket found. Please set INCUS_SOCKET or ensure Incus is running")
		return nil
	}

	// Connect to Incus
	log.Printf("[IncusDriver] Connecting to Incus at: %s", socketPath)
	server, err := incusclient.ConnectIncusUnix(socketPath, nil)
	if err != nil {
		log.Printf("[IncusDriver] Failed to connect to Incus: %v", err)
		log.Printf("[IncusDriver] Make sure Incus is running")
		return nil
	}

	log.Printf("[IncusDriver] Successfully connected to Incus!")
	return &IncusDriver{
		server: server,
		ctx:    context.Background(),
	}
}

func (d *IncusDriver) Run(ctx context.Context, job *pb.Job) (string, error) {
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

func (d *IncusDriver) pullImage(ctx context.Context, image string) error {
	// Incus will automatically pull the image if it's not available when creating an instance
	// So we can skip explicit image pulling, or try to ensure it exists
	log.Printf("[IncusDriver] Checking/ensuring image availability: %s", image)

	// Try to get the image - if it doesn't exist, Incus will pull it automatically during instance creation
	_, _, err := d.server.GetImageAlias(image)
	if err != nil {
		// Image alias doesn't exist, but that's OK - Incus will pull it during instance creation
		log.Printf("[IncusDriver] Image alias not found, will be pulled during instance creation: %s", image)
		return nil
	}

	log.Printf("[IncusDriver] Image alias found: %s", image)
	return nil
}

func (d *IncusDriver) createInstance(ctx context.Context, job *pb.Job) (string, error) {
	// Generate container name from job ID
	containerName := fmt.Sprintf("open-scheduler-%s", job.JobId)

	// Prepare instance creation request
	req := api.InstancesPost{
		Name: containerName,
		Type: api.InstanceTypeContainer,
		Source: api.InstanceSource{
			Type:  "image",
			Alias: job.InstanceConfig.ImageName,
		},
	}

	// Set instance configuration
	instanceConfig := make(map[string]string)
	instanceConfig["user.open-scheduler.managed"] = "true"
	instanceConfig["user.open-scheduler.job-name"] = job.JobName
	instanceConfig["user.open-scheduler.job-id"] = job.JobId

	// Set resource limits
	if job.ResourceRequirements != nil {
		if job.ResourceRequirements.MemoryLimitMb > 0 {
			instanceConfig["limits.memory"] = fmt.Sprintf("%dMB", job.ResourceRequirements.MemoryLimitMb)
		}
		if job.ResourceRequirements.CpuLimitCores > 0 {
			instanceConfig["limits.cpu"] = fmt.Sprintf("%.2f", job.ResourceRequirements.CpuLimitCores)
		}
	}

	req.Config = instanceConfig

	// Set environment variables
	if len(job.EnvironmentVariables) > 0 {
		envVars := ""
		for k, v := range job.EnvironmentVariables {
			if envVars != "" {
				envVars += " "
			}
			envVars += fmt.Sprintf("%s=%s", k, v)
		}
		instanceConfig["environment"] = envVars
	}

	// Set command and arguments
	if len(job.InstanceConfig.Entrypoint) > 0 {
		cmd := ""
		for i, arg := range job.InstanceConfig.Entrypoint {
			if i > 0 {
				cmd += " "
			}
			cmd += arg
		}
		if len(job.InstanceConfig.Arguments) > 0 {
			for _, arg := range job.InstanceConfig.Arguments {
				cmd += " " + arg
			}
		}
		instanceConfig["raw.lxc"] = fmt.Sprintf("lxc.init.cmd=%s", cmd)
	}

	// Create instance
	log.Printf("[IncusDriver] Creating instance for job %s with image: %s", job.JobId, job.InstanceConfig.ImageName)
	op, err := d.server.CreateInstance(req)
	if err != nil {
		return "", fmt.Errorf("failed to create instance: %w", err)
	}

	// Wait for instance to be created
	err = op.Wait()
	if err != nil {
		return "", fmt.Errorf("failed to wait for instance creation: %w", err)
	}

	// Add volume mounts
	if len(job.VolumeMounts) > 0 {
		instance, _, err := d.server.GetInstance(containerName)
		if err != nil {
			return "", fmt.Errorf("failed to get instance: %w", err)
		}

		deviceMap := make(map[string]map[string]string)
		for i, vol := range job.VolumeMounts {
			deviceName := fmt.Sprintf("volume%d", i)
			deviceConfig := map[string]string{
				"type":     "disk",
				"source":   vol.SourcePath,
				"path":     vol.TargetPath,
				"readonly": fmt.Sprintf("%v", vol.ReadOnly),
			}
			deviceMap[deviceName] = deviceConfig
		}

		instance.Devices = deviceMap
		op, err = d.server.UpdateInstance(containerName, instance.Writable(), "")
		if err != nil {
			return "", fmt.Errorf("failed to add volume mounts: %w", err)
		}
		err = op.Wait()
		if err != nil {
			return "", fmt.Errorf("failed to wait for volume mount update: %w", err)
		}
	}

	// Start instance
	log.Printf("[IncusDriver] Starting instance: %s", containerName)
	reqState := api.InstanceStatePut{
		Action:  "start",
		Timeout: -1,
	}
	op, err = d.server.UpdateInstanceState(containerName, reqState, "")
	if err != nil {
		return "", fmt.Errorf("failed to start instance: %w", err)
	}

	err = op.Wait()
	if err != nil {
		return "", fmt.Errorf("failed to wait for instance start: %w", err)
	}

	log.Printf("[IncusDriver] Instance is running: %s", containerName)
	return containerName, nil
}

func (d *IncusDriver) StopInstance(ctx context.Context, instanceID string) error {
	log.Printf("[IncusDriver] Stopping instance: %s", instanceID)
	reqState := api.InstanceStatePut{
		Action:  "stop",
		Timeout: -1,
		Force:   true,
	}
	op, err := d.server.UpdateInstanceState(instanceID, reqState, "")
	if err != nil {
		return fmt.Errorf("failed to stop instance %s: %w", instanceID, err)
	}

	err = op.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for instance stop: %w", err)
	}

	// Delete instance
	delOp, err := d.server.DeleteInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance %s: %w", instanceID, err)
	}

	err = delOp.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for instance deletion: %w", err)
	}

	log.Printf("[IncusDriver] Instance stopped: %s", instanceID)
	return nil
}

func (d *IncusDriver) RestartInstance(ctx context.Context, instanceID string) error {
	log.Printf("[IncusDriver] Restarting instance: %s", instanceID)
	reqState := api.InstanceStatePut{
		Action:  "restart",
		Timeout: -1,
		Force:   true,
	}
	op, err := d.server.UpdateInstanceState(instanceID, reqState, "")
	if err != nil {
		return fmt.Errorf("failed to restart instance %s: %w", instanceID, err)
	}

	err = op.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for instance restart: %w", err)
	}

	log.Printf("[IncusDriver] Instance restarted: %s", instanceID)
	return nil
}

func (d *IncusDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	state, _, err := d.server.GetInstanceState(instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get instance status for %s: %w", instanceID, err)
	}

	// Map Incus status to standard status
	switch state.Status {
	case "Running":
		return "running", nil
	case "Stopped":
		return "stopped", nil
	case "Starting":
		return "starting", nil
	case "Stopping":
		return "stopping", nil
	case "Aborting":
		return "aborting", nil
	case "Freezing":
		return "freezing", nil
	case "Frozen":
		return "frozen", nil
	case "Thawed":
		return "thawed", nil
	case "Error":
		return "error", nil
	default:
		return state.Status, nil
	}
}

func (d *IncusDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	instance, _, err := d.server.GetInstance(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect instance %s: %w", instanceID, err)
	}

	state, _, err := d.server.GetInstanceState(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance state %s: %w", instanceID, err)
	}

	// Extract labels
	labels := make(map[string]string)
	for k, v := range instance.Config {
		if len(k) > 5 && k[:5] == "user." {
			labels[k[5:]] = v
		}
	}

	// Extract volumes
	volumes := make([]string, 0)
	for _, device := range instance.Devices {
		if device["type"] == "disk" {
			if path, ok := device["path"]; ok {
				volumes = append(volumes, path)
			}
		}
	}

	// Extract command from config
	command := make([]string, 0)
	if len(instance.Config["raw.lxc"]) > 0 {
		// Parse command from raw.lxc if available
		// This is a simplified parser - in production you might want more robust parsing
		if len(instance.Config["raw.lxc"]) > 13 && instance.Config["raw.lxc"][:13] == "lxc.init.cmd=" {
			cmdStr := instance.Config["raw.lxc"][13:]
			// Simple split - in production use proper shell parsing
			command = []string{cmdStr}
		}
	}

	// Get image name
	imageName := instance.Config["image.description"]
	if imageName == "" {
		imageName = instance.Config["volatile.base_image"]
	}
	if imageName == "" {
		imageName = "unknown"
	}

	// Format timestamps
	created := instance.CreatedAt.Format(time.RFC3339Nano)
	startedAt := ""
	finishedAt := ""
	if state.Pid > 0 && state.Status == "Running" {
		// If instance is running, use creation time as start time (or current time)
		startedAt = instance.CreatedAt.Format(time.RFC3339Nano)
	}
	if state.Status == "Stopped" && state.StatusCode != 0 {
		finishedAt = time.Now().Format(time.RFC3339Nano)
	}

	// Map status
	status := "unknown"
	switch state.Status {
	case "Running":
		status = "running"
	case "Stopped":
		status = "stopped"
	default:
		status = state.Status
	}

	return &pb.InstanceData{
		InstanceId:   instanceID,
		InstanceName: instance.Name,
		Image:        imageName,
		ImageName:    imageName,
		Command:      command,
		Args:         []string{},
		Created:      created,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		Status:       status,
		ExitCode:     int32(state.StatusCode),
		Pid:          int32(state.Pid),
		Labels:       labels,
		Ports:        []string{},
		Volumes:      volumes,
	}, nil
}

func (d *IncusDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	log.Printf("[IncusDriver] Listing all instances")

	instances, err := d.server.GetInstances(api.InstanceTypeAny)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	result := make([]*pb.InstanceData, 0)

	for _, instance := range instances {
		// Check if instance is managed by open-scheduler
		if managed, ok := instance.Config["user.open-scheduler.managed"]; ok && managed == "true" {
			inspectData, err := d.InspectInstance(ctx, instance.Name)
			if err != nil {
				log.Printf("[IncusDriver] Warning: failed to inspect instance %s: %v", instance.Name, err)
				continue
			}
			result = append(result, inspectData)
		}
	}

	log.Printf("[IncusDriver] Found %d managed instances", len(result))
	return result, nil
}
