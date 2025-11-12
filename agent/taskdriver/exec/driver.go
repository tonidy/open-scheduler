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
	if len(job.ContainerConfig.Entrypoint) == 0 {
		return fmt.Errorf("no command specified for job %s", job.JobName)
	}

	// Create command with args
	if len(job.ContainerConfig.Arguments) > 0 {
		cmdArgs := append(job.ContainerConfig.Entrypoint[1:], job.ContainerConfig.Arguments...)
		cmd = exec.CommandContext(ctx, job.ContainerConfig.Entrypoint[0], cmdArgs...)
	} else if len(job.ContainerConfig.Entrypoint) > 1 {
		cmd = exec.CommandContext(ctx, job.ContainerConfig.Entrypoint[0], job.ContainerConfig.Entrypoint[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, job.ContainerConfig.Entrypoint[0])
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
	log.Printf("[ExecDriver] Starting command: %v", job.ContainerConfig.Entrypoint)
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

// StopContainer is a mock implementation for the Driver interface
func (d *ExecDriver) StopContainer(ctx context.Context, containerID string) error {
	log.Printf("[ExecDriver] StopContainer called for: %s (not implemented)", containerID)
	return nil
}

// RestartContainer is a mock implementation for the Driver interface
func (d *ExecDriver) RestartContainer(ctx context.Context, containerID string) error {
	log.Printf("[ExecDriver] RestartContainer called for: %s (not implemented)", containerID)
	return nil
}

// GetContainerStatus is a mock implementation for the Driver interface
func (d *ExecDriver) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	log.Printf("[ExecDriver] GetContainerStatus called for: %s (not implemented)", containerID)
	return "unknown", nil
}

func (d *ExecDriver) InspectContainer(ctx context.Context, containerID string) (*pb.ContainerData, error) {
	log.Printf("[ExecDriver] InspectContainer called for: %s (not implemented)", containerID)
	return nil, fmt.Errorf("InspectContainer not implemented for exec driver")
}

func (d *ExecDriver) ListContainers(ctx context.Context) ([]*pb.ContainerData, error) {
	return []*pb.ContainerData{}, nil
}

