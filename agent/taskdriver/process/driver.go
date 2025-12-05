package process

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	pb "github.com/open-scheduler/proto"
)

type ProcessInfo struct {
	cmd       *exec.Cmd
	jobID     string
	jobName   string
	startedAt time.Time
	status    string
	exitCode  int32
	pid       int32
	labels    map[string]string
	command   []string
	args      []string
	envVars   map[string]string
}

type ProcessDriver struct {
	processes map[string]*ProcessInfo
	mu        sync.RWMutex
}

func NewProcessDriver() *ProcessDriver {
	return &ProcessDriver{
		processes: make(map[string]*ProcessInfo),
	}
}

func (d *ProcessDriver) Run(ctx context.Context, deployment *pb.Deployment) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Generate instance ID from deployment ID
	instanceID := fmt.Sprintf("process-%s", deployment.DeploymentId)

	// Build command
	// Priority: command_array > InstanceConfig.Entrypoint > Command (legacy)
	var cmd *exec.Cmd
	if len(deployment.CommandArray) > 0 {
		cmd = exec.CommandContext(ctx, deployment.CommandArray[0], deployment.CommandArray[1:]...)
	} else if len(deployment.InstanceConfig.Entrypoint) > 0 {
		cmd = exec.CommandContext(ctx, deployment.InstanceConfig.Entrypoint[0], deployment.InstanceConfig.Entrypoint[1:]...)
		if len(deployment.InstanceConfig.Arguments) > 0 {
			cmd.Args = append(cmd.Args, deployment.InstanceConfig.Arguments...)
		}
	} else if deployment.Command != "" {
		// Fallback to command field if entrypoint is not set
		cmd = exec.CommandContext(ctx, "sh", "-c", deployment.Command)
	} else {
		return "", fmt.Errorf("no command or entrypoint specified")
	}

	// Set working directory
	if deployment.WorkingDir != "" {
		cmd.Dir = deployment.WorkingDir
	}

	// Set environment variables
	if len(deployment.EnvironmentVariables) > 0 {
		env := os.Environ()
		for k, v := range deployment.EnvironmentVariables {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Set working directory if volume mounts exist
	if len(deployment.VolumeMounts) > 0 {
		// Use first volume mount as working directory if it's a single mount
		// Otherwise, use current directory
		cmd.Dir = deployment.VolumeMounts[0].TargetPath
	}

	// Start the process
	log.Printf("[ProcessDriver] Starting process for deployment %s: %v", deployment.DeploymentId, cmd.Args)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start process: %w", err)
	}

	// Create process info
	info := &ProcessInfo{
		cmd:       cmd,
		jobID:     deployment.DeploymentId,
		jobName:   deployment.DeploymentName,
		startedAt: time.Now(),
		status:    "running",
		pid:       int32(cmd.Process.Pid),
		labels: map[string]string{
			"open-scheduler.managed":        "true",
			"open-scheduler.deployment-name": deployment.DeploymentName,
			"open-scheduler.deployment-id":  deployment.DeploymentId,
		},
		command: cmd.Args,
		args:    cmd.Args[1:],
		envVars: deployment.EnvironmentVariables,
	}

	d.processes[instanceID] = info

	// Monitor process completion in background
	go d.monitorProcess(instanceID, cmd)

	log.Printf("[ProcessDriver] Process started with PID %d: %s", cmd.Process.Pid, instanceID)
	return instanceID, nil
}

func (d *ProcessDriver) monitorProcess(instanceID string, cmd *exec.Cmd) {
	err := cmd.Wait()

	d.mu.Lock()
	defer d.mu.Unlock()

	info, exists := d.processes[instanceID]
	if !exists {
		return
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				info.exitCode = int32(status.ExitStatus())
			}
		}
		info.status = "exited"
		log.Printf("[ProcessDriver] Process %s exited with code %d", instanceID, info.exitCode)
	} else {
		info.exitCode = 0
		info.status = "exited"
		log.Printf("[ProcessDriver] Process %s completed successfully", instanceID)
	}
}

func (d *ProcessDriver) StopInstance(ctx context.Context, instanceID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	info, exists := d.processes[instanceID]
	if !exists {
		return fmt.Errorf("process %s not found", instanceID)
	}

	if info.status != "running" {
		log.Printf("[ProcessDriver] Process %s is not running (status: %s)", instanceID, info.status)
		return nil
	}

	log.Printf("[ProcessDriver] Stopping process %s (PID: %d)", instanceID, info.pid)

	// Kill the process
	if err := info.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process %s: %w", instanceID, err)
	}

	info.status = "stopped"
	log.Printf("[ProcessDriver] Process %s stopped", instanceID)
	return nil
}

func (d *ProcessDriver) RestartInstance(ctx context.Context, instanceID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	info, exists := d.processes[instanceID]
	if !exists {
		return fmt.Errorf("process %s not found", instanceID)
	}

	// Stop the current process if running
	if info.status == "running" {
		if err := info.cmd.Process.Kill(); err != nil {
			log.Printf("[ProcessDriver] Warning: failed to kill process during restart: %v", err)
		}
	}

	// Create new command
	var cmd *exec.Cmd
	if len(info.command) > 0 {
		cmd = exec.CommandContext(ctx, info.command[0], info.command[1:]...)
	} else {
		return fmt.Errorf("cannot restart: no command information available")
	}

	// Set environment variables
	if len(info.envVars) > 0 {
		env := os.Environ()
		for k, v := range info.envVars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Start the new process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart process: %w", err)
	}

	// Update process info
	info.cmd = cmd
	info.pid = int32(cmd.Process.Pid)
	info.startedAt = time.Now()
	info.status = "running"
	info.exitCode = 0

	// Monitor new process
	go d.monitorProcess(instanceID, cmd)

	log.Printf("[ProcessDriver] Process %s restarted with PID %d", instanceID, info.pid)
	return nil
}

func (d *ProcessDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	info, exists := d.processes[instanceID]
	if !exists {
		return "", fmt.Errorf("process %s not found", instanceID)
	}

	// Check if process is still alive
	if info.status == "running" {
		// Check if process is actually still running
		if err := info.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			// Process is dead
			info.status = "exited"
			return "exited", nil
		}
	}

	return info.status, nil
}

func (d *ProcessDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	info, exists := d.processes[instanceID]
	if !exists {
		return nil, fmt.Errorf("process %s not found", instanceID)
	}

	// Check if process is still alive
	status := info.status
	if status == "running" {
		if err := info.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			status = "exited"
		}
	}

	finishedAt := ""
	if status == "exited" || status == "stopped" {
		finishedAt = time.Now().Format(time.RFC3339Nano)
	}

	return &pb.InstanceData{
		InstanceId:   instanceID,
		InstanceName: info.jobName,
		Image:        "process",
		ImageName:    "process",
		Command:      info.command,
		Args:         info.args,
		Created:      info.startedAt.Format(time.RFC3339Nano),
		StartedAt:    info.startedAt.Format(time.RFC3339Nano),
		FinishedAt:   finishedAt,
		Status:       status,
		ExitCode:     info.exitCode,
		Pid:          info.pid,
		Labels:       info.labels,
		Ports:        []string{},
		Volumes:      []string{},
	}, nil
}

func (d *ProcessDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*pb.InstanceData, 0, len(d.processes))
	for instanceID := range d.processes {
		inspectData, err := d.InspectInstance(ctx, instanceID)
		if err != nil {
			log.Printf("[ProcessDriver] Warning: failed to inspect instance %s: %v", instanceID, err)
			continue
		}
		result = append(result, inspectData)
	}

	log.Printf("[ProcessDriver] Found %d managed processes", len(result))
	return result, nil
}
