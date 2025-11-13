package process

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	pb "github.com/open-scheduler/proto"
)

type ProcessDriver struct {
	processes map[string]*os.Process
	mu        sync.RWMutex
}

func NewProcessDriver() *ProcessDriver {
	log.Printf("[ProcessDriver] Initializing process driver for direct shell commands")
	return &ProcessDriver{
		processes: make(map[string]*os.Process),
	}
}

func (d *ProcessDriver) Run(ctx context.Context, job *pb.Job) (string, error) {
	log.Printf("[ProcessDriver] Running job: %s (ID: %s)", job.JobName, job.JobId)

	// Build command
	var cmd *exec.Cmd
	if len(job.InstanceConfig.Entrypoint) == 0 {
		return "", fmt.Errorf("no command specified for job %s", job.JobName)
	}

	// Create command with args
	if len(job.InstanceConfig.Arguments) > 0 {
		cmdArgs := append(job.InstanceConfig.Entrypoint[1:], job.InstanceConfig.Arguments...)
		cmd = exec.CommandContext(ctx, job.InstanceConfig.Entrypoint[0], cmdArgs...)
	} else if len(job.InstanceConfig.Entrypoint) > 1 {
		cmd = exec.CommandContext(ctx, job.InstanceConfig.Entrypoint[0], job.InstanceConfig.Entrypoint[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, job.InstanceConfig.Entrypoint[0])
	}

	// Set environment variables
	if len(job.EnvironmentVariables) > 0 {
		env := os.Environ()
		for k, v := range job.EnvironmentVariables {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Set up stdio
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the command
	log.Printf("[ProcessDriver] Starting command: %v", job.InstanceConfig.Entrypoint)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Store process using job ID as key
	d.mu.Lock()
	d.processes[job.JobId] = cmd.Process
	d.mu.Unlock()

	log.Printf("[ProcessDriver] Command started with PID: %d for job %s", cmd.Process.Pid, job.JobId)

	// Wait for command to complete in a goroutine
	go func() {
		err := cmd.Wait()
		d.mu.Lock()
		delete(d.processes, job.JobId)
		d.mu.Unlock()

		if err != nil {
			log.Printf("[ProcessDriver] Command failed for job %s: %v", job.JobId, err)
		} else {
			log.Printf("[ProcessDriver] Command completed successfully for job %s", job.JobId)
		}
	}()

	return fmt.Sprintf("%d", cmd.Process.Pid), nil
}

// StopInstance is a mock implementation for the Driver interface
func (d *ProcessDriver) StopInstance(ctx context.Context, instanceID string) error {
	log.Printf("[ProcessDriver] StopInstance called for: %s (not implemented)", instanceID)
	return fmt.Errorf("StopInstance not implemented for process driver")
}

// RestartInstance is a mock implementation for the Driver interface
func (d *ProcessDriver) RestartInstance(ctx context.Context, instanceID string) error {
	log.Printf("[ProcessDriver] RestartInstance called for: %s (not implemented)", instanceID)
	return fmt.Errorf("RestartInstance not implemented for process driver")
}

// GetInstanceStatus is a mock implementation for the Driver interface
func (d *ProcessDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	log.Printf("[ProcessDriver] GetInstanceStatus called for: %s (not implemented)", instanceID)
	return "unknown", nil
}

func (d *ProcessDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	log.Printf("[ProcessDriver] InspectInstance called for: %s (not implemented)", instanceID)
	return nil, fmt.Errorf("InspectInstance not implemented for process driver")
}

func (d *ProcessDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	log.Printf("[ProcessDriver] ListInstances called (not implemented)")
	return []*pb.InstanceData{}, nil
}
