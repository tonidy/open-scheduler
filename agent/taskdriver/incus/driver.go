package incus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	pb "github.com/open-scheduler/proto"
)

type IncusDriver struct {
	ctx context.Context
}

// IncusInstance represents an Incus instance from the list command
type IncusInstance struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Stateful    bool   `json:"stateful"`
	Location    string `json:"location"`
	Project     string `json:"project"`
	Description string `json:"description"`
}

// IncusInstanceDetail represents detailed instance information
type IncusInstanceDetail struct {
	Architecture string                 `json:"architecture"`
	CreatedAt    string                 `json:"created_at"`
	Description  string                 `json:"description"`
	Devices      map[string]interface{} `json:"devices"`
	Ephemeral    bool                   `json:"ephemeral"`
	Environment  map[string]string      `json:"environment"`
	ExpandedConfig map[string]string    `json:"expanded_config"`
	ExpandedDevices map[string]interface{} `json:"expanded_devices"`
	Hostname     string                 `json:"hostname"`
	LastUsedAt   string                 `json:"last_used_at"`
	Location     string                 `json:"location"`
	Name         string                 `json:"name"`
	Platform     string                 `json:"platform"`
	Profiles     []string               `json:"profiles"`
	Stateful     bool                   `json:"stateful"`
	Status       string                 `json:"status"`
	Type         string                 `json:"type"`
}

func NewIncusDriver() *IncusDriver {
	return &IncusDriver{
		ctx: context.Background(),
	}
}

func (d *IncusDriver) Run(ctx context.Context, job *pb.Job) (string, error) {
	// Apply job timeout if specified
	if job.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
		defer cancel()
		log.Printf("[IncusDriver] Job timeout set to %d seconds", job.TimeoutSeconds)
	}

	instanceName := job.JobId
	image := job.InstanceConfig.ImageName

	log.Printf("[IncusDriver] Creating instance for job %s with image: %s", job.JobId, image)

	// Create instance
	args := []string{"launch", image, instanceName}

	// Add command/entrypoint if specified
	if len(job.InstanceConfig.Entrypoint) > 0 {
		// Note: Incus executes commands differently than Podman
		// We'll set them as environment variables and document this limitation
		log.Printf("[IncusDriver] Warning: Incus driver doesn't support custom entrypoints like containers")
	}

	// Create the instance
	cmd := exec.CommandContext(ctx, "incus", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create instance: %w", err)
	}

	log.Printf("[IncusDriver] Instance created: %s", instanceName)

	// Wait for instance to start
	if err := d.waitForStatus(ctx, instanceName, "Running"); err != nil {
		log.Printf("[IncusDriver] Warning: Failed to wait for instance to start: %v", err)
		// Continue anyway, instance might be starting
	}

	log.Printf("[IncusDriver] Instance is running: %s", instanceName)
	return instanceName, nil
}

func (d *IncusDriver) waitForStatus(ctx context.Context, instanceName, expectedStatus string) error {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := d.GetInstanceStatus(ctx, instanceName)
		if err != nil {
			log.Printf("[IncusDriver] Error getting status: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if status == expectedStatus {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for instance to reach status %s", expectedStatus)
}

func (d *IncusDriver) StopInstance(ctx context.Context, instanceID string) error {
	log.Printf("[IncusDriver] Stopping instance: %s", instanceID)

	// Stop the instance
	cmd := exec.CommandContext(ctx, "incus", "stop", instanceID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop instance %s: %w", instanceID, err)
	}

	// Delete the instance
	cmd = exec.CommandContext(ctx, "incus", "delete", instanceID, "--force")
	if err := cmd.Run(); err != nil {
		log.Printf("[IncusDriver] Warning: Failed to delete instance %s: %v", instanceID, err)
		// Don't fail the operation, the instance is stopped
	}

	log.Printf("[IncusDriver] Instance stopped and cleaned up: %s", instanceID)
	return nil
}

func (d *IncusDriver) RestartInstance(ctx context.Context, instanceID string) error {
	log.Printf("[IncusDriver] Restarting instance: %s", instanceID)
	cmd := exec.CommandContext(ctx, "incus", "restart", instanceID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart instance %s: %w", instanceID, err)
	}

	log.Printf("[IncusDriver] Instance restarted: %s", instanceID)
	return nil
}

func (d *IncusDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	// Use incus list with JSON output
	cmd := exec.CommandContext(ctx, "incus", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []IncusInstance
	if err := json.Unmarshal(output, &instances); err != nil {
		return "", fmt.Errorf("failed to parse instance list: %w", err)
	}

	for _, instance := range instances {
		if instance.Name == instanceID {
			return instance.Status, nil
		}
	}

	return "", fmt.Errorf("instance not found: %s", instanceID)
}

func (d *IncusDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	// Get instance information using incus query
	cmd := exec.CommandContext(ctx, "incus", "query", "/1.0/instances/"+instanceID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect instance %s: %w", instanceID, err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse instance info: %w", err)
	}

	// Extract metadata
	metadata, ok := response["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid metadata in response")
	}

	name, _ := metadata["name"].(string)
	status, _ := metadata["status"].(string)
	created, _ := metadata["created_at"].(string)

	return &pb.InstanceData{
		InstanceId:   instanceID,
		InstanceName: name,
		Status:       status,
		Created:      created,
		Image:        "", // Incus uses different terminology for images
	}, nil
}

func (d *IncusDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	log.Printf("[IncusDriver] Listing all instances")

	// List instances with JSON output
	cmd := exec.CommandContext(ctx, "incus", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []IncusInstance
	if err := json.Unmarshal(output, &instances); err != nil {
		return nil, fmt.Errorf("failed to parse instance list: %w", err)
	}

	result := make([]*pb.InstanceData, 0)

	// Filter for instances that look like they were created by open-scheduler
	// (would need a labeling/naming convention to properly identify them)
	for _, instance := range instances {
		// For now, include all instances (in production, would use labels/naming convention)
		instanceData := &pb.InstanceData{
			InstanceId:   instance.Name,
			InstanceName: instance.Name,
			Status:       instance.Status,
			Type:         instance.Type,
		}
		result = append(result, instanceData)
	}

	log.Printf("[IncusDriver] Found %d instances", len(result))
	return result, nil
}
