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

func (d *ExecDriver) Run(ctx context.Context, task *pb.Task) error {
	log.Printf("[ExecDriver] Running task: %s", task.Name)

	// Build command
	var cmd *exec.Cmd
	if len(task.Config.Command) == 0 {
		return fmt.Errorf("no command specified for task %s", task.Name)
	}

	// Create command with args
	if len(task.Config.Args) > 0 {
		cmdArgs := append(task.Config.Command[1:], task.Config.Args...)
		cmd = exec.CommandContext(ctx, task.Config.Command[0], cmdArgs...)
	} else if len(task.Config.Command) > 1 {
		cmd = exec.CommandContext(ctx, task.Config.Command[0], task.Config.Command[1:]...)
	} else {
		cmd = exec.CommandContext(ctx, task.Config.Command[0])
	}

	// Set environment variables
	if len(task.Env) > 0 {
		env := os.Environ()
		for k, v := range task.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Set up stdio
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the command
	log.Printf("[ExecDriver] Starting command: %v", task.Config.Command)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Store process
	d.mu.Lock()
	d.processes[task.Name] = cmd.Process
	d.mu.Unlock()

	log.Printf("[ExecDriver] Command started with PID: %d", cmd.Process.Pid)

	// Wait for command to complete in a goroutine
	go func() {
		err := cmd.Wait()
		d.mu.Lock()
		delete(d.processes, task.Name)
		d.mu.Unlock()

		if err != nil {
			log.Printf("[ExecDriver] Command failed for task %s: %v", task.Name, err)
		} else {
			log.Printf("[ExecDriver] Command completed successfully for task %s", task.Name)
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
