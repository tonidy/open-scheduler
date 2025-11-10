package exec

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/open-scheduler/agent/taskdriver/model"
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

func (d *ExecDriver) Run(ctx context.Context, task *pb.Task) error {
	log.Printf("[ExecDriver] Running task: %s", task.TaskName)

	// Build command
	var cmd *exec.Cmd
	if len(task.ContainerConfig.Entrypoint) == 0 {
		return fmt.Errorf("no command specified for task %s", task.TaskName)
	}

	// Create command with args
	if len(task.ContainerConfig.Arguments) > 0 {
		cmdArgs := append(task.ContainerConfig.Entrypoint[1:], task.ContainerConfig.Arguments...)
		cmd = exec.CommandContext(ctx, task.ContainerConfig.Entrypoint[0], cmdArgs...)
	} else if len(task.ContainerConfig.Entrypoint) > 1 {
		cmd = exec.CommandContext(ctx, task.ContainerConfig.Entrypoint[0], task.ContainerConfig.Entrypoint[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, task.ContainerConfig.Entrypoint[0])
	}

	// Set environment variables
	if len(task.EnvironmentVariables) > 0 {
		env := os.Environ()
		for k, v := range task.EnvironmentVariables {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Set up stdio
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the command
	log.Printf("[ExecDriver] Starting command: %v", task.ContainerConfig.Entrypoint)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Store process
	d.mu.Lock()
	d.processes[task.TaskName] = cmd.Process
	d.mu.Unlock()

	log.Printf("[ExecDriver] Command started with PID: %d", cmd.Process.Pid)

	// Wait for command to complete in a goroutine
	go func() {
		err := cmd.Wait()
		d.mu.Lock()
		delete(d.processes, task.TaskName)
		d.mu.Unlock()

		if err != nil {
			log.Printf("[ExecDriver] Command failed for task %s: %v", task.TaskName, err)
		} else {
			log.Printf("[ExecDriver] Command completed successfully for task %s", task.TaskName)
		}
	}()

	return nil
}

// StopContainer is a mock implementation for the Driver interface
func (d *ExecDriver) StopContainer(ctx context.Context, containerID string) error {
	log.Printf("[ExecDriver] StopContainer called for: %s (not implemented)", containerID)
	return nil
}

// GetContainerStatus is a mock implementation for the Driver interface
func (d *ExecDriver) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	log.Printf("[ExecDriver] GetContainerStatus called for: %s (not implemented)", containerID)
	return "unknown", nil
}

func (d *ExecDriver) InspectContainer(ctx context.Context, containerID string) (model.ContainerInspect, error) {
	log.Printf("[ExecDriver] InspectContainer called for: %s (not implemented)", containerID)
	return model.ContainerInspect{}, fmt.Errorf("InspectContainer not implemented for exec driver")
}

// ListContainers lists all containers (not applicable for exec driver)
func (d *ExecDriver) ListContainers(ctx context.Context) ([]model.ContainerInspect, error) {
	// Exec driver doesn't manage containers
	return []model.ContainerInspect{}, nil
}
