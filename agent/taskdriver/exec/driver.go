package exec

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	pb "github.com/open-scheduler/proto"
)

type ExecDriver struct {
	processes map[string]*os.Process
	mu        sync.RWMutex
}

func NewExecDriver() *ExecDriver {
	log.Printf("[ExecDriver] Initializing exec driver for direct shell commands")
	return &ExecDriver{
		processes: make(map[string]*os.Process),
	}
}

func (d *ExecDriver) Run(ctx context.Context, job *pb.Job) error {
	log.Printf("[ExecDriver] Running job: %s (ID: %s)", job.JobName, job.JobId)

	// Build command
	var cmd *exec.Cmd
	if len(job.InstanceConfig.Entrypoint) == 0 {
		return fmt.Errorf("no command specified for job %s", job.JobName)
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
	log.Printf("[ExecDriver] Starting command: %v", job.InstanceConfig.Entrypoint)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Store process using job ID as key
	d.mu.Lock()
	d.processes[job.JobId] = cmd.Process
	d.mu.Unlock()

	log.Printf("[ExecDriver] Command started with PID: %d for job %s", cmd.Process.Pid, job.JobId)

	// Wait for command to complete in a goroutine
	go func() {
		err := cmd.Wait()
		d.mu.Lock()
		delete(d.processes, job.JobId)
		d.mu.Unlock()

		if err != nil {
			log.Printf("[ExecDriver] Command failed for job %s: %v", job.JobId, err)
		} else {
			log.Printf("[ExecDriver] Command completed successfully for job %s", job.JobId)
		}
	}()

	return nil
}

// StopInstance is a mock implementation for the Driver interface
func (d *ExecDriver) StopInstance(ctx context.Context, instanceID string) error {
	log.Printf("[ExecDriver] StopInstance called for: %s (not implemented)", instanceID)
	return nil
}

// RestartInstance is a mock implementation for the Driver interface
func (d *ExecDriver) RestartInstance(ctx context.Context, instanceID string) error {
	log.Printf("[ExecDriver] RestartInstance called for: %s (not implemented)", instanceID)
	return nil
}

// GetInstanceStatus is a mock implementation for the Driver interface
func (d *ExecDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	log.Printf("[ExecDriver] GetInstanceStatus called for: %s (not implemented)", instanceID)
	return "unknown", nil
}

func (d *ExecDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	log.Printf("[ExecDriver] InspectInstance called for: %s (not implemented)", instanceID)
	return nil, fmt.Errorf("InspectInstance not implemented for exec driver")
}

func (d *ExecDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	return []*pb.InstanceData{}, nil
}

