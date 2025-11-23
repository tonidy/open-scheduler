package cleanup

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/open-scheduler/proto"
)

// MockDriver is a mock implementation of the taskdriver.Driver interface
type MockDriver struct {
	instances map[string]*pb.InstanceData
	stopCalls []string
}

func NewMockDriver() *MockDriver {
	return &MockDriver{
		instances: make(map[string]*pb.InstanceData),
		stopCalls: make([]string, 0),
	}
}

func (m *MockDriver) AddInstance(id string, status string) {
	m.instances[id] = &pb.InstanceData{
		InstanceId:   id,
		InstanceName: id,
		Status:       status,
	}
}

func (m *MockDriver) Run(ctx context.Context, job *pb.Job) (string, error) {
	return "", fmt.Errorf("not implemented for mock")
}

func (m *MockDriver) StopInstance(ctx context.Context, instanceID string) error {
	m.stopCalls = append(m.stopCalls, instanceID)
	delete(m.instances, instanceID)
	return nil
}

func (m *MockDriver) StopInstanceError(ctx context.Context, instanceID string) error {
	m.stopCalls = append(m.stopCalls, instanceID)
	return fmt.Errorf("failed to stop instance")
}

func (m *MockDriver) RestartInstance(ctx context.Context, instanceID string) error {
	return fmt.Errorf("not implemented for mock")
}

func (m *MockDriver) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	if instance, ok := m.instances[instanceID]; ok {
		return instance.Status, nil
	}
	return "", fmt.Errorf("instance not found")
}

func (m *MockDriver) InspectInstance(ctx context.Context, instanceID string) (*pb.InstanceData, error) {
	if instance, ok := m.instances[instanceID]; ok {
		return instance, nil
	}
	return nil, fmt.Errorf("instance not found")
}

func (m *MockDriver) ListInstances(ctx context.Context) ([]*pb.InstanceData, error) {
	result := make([]*pb.InstanceData, 0, len(m.instances))
	for _, instance := range m.instances {
		result = append(result, instance)
	}
	return result, nil
}

func TestNewCleanupService(t *testing.T) {
	driver := NewMockDriver()
	svc, err := NewCleanupService(driver, "test-node")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if svc == nil {
		t.Fatal("Expected service, got nil")
	}
}

func TestNewCleanupService_NilDriver(t *testing.T) {
	_, err := NewCleanupService(nil, "test-node")
	if err == nil {
		t.Fatal("Expected error for nil driver, got nil")
	}
	if err.Error() != "driver cannot be nil" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCleanupService_NoInstances(t *testing.T) {
	driver := NewMockDriver()
	svc, err := NewCleanupService(driver, "test-node")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	err = svc.Execute(ctx, "test-node", "test-token")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have no stop calls
	if len(driver.stopCalls) != 0 {
		t.Fatalf("Expected 0 stop calls, got %d", len(driver.stopCalls))
	}
}

func TestCleanupService_StopsRunningInstances(t *testing.T) {
	driver := NewMockDriver()
	driver.AddInstance("running-1", "running")
	driver.AddInstance("running-2", "running")

	svc, err := NewCleanupService(driver, "test-node")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	err = svc.Execute(ctx, "test-node", "test-token")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have stopped both running instances
	if len(driver.stopCalls) != 2 {
		t.Fatalf("Expected 2 stop calls, got %d", len(driver.stopCalls))
	}

	// Verify the running instances were stopped
	if _, ok := driver.instances["running-1"]; ok {
		t.Fatal("Expected running-1 to be stopped")
	}
	if _, ok := driver.instances["running-2"]; ok {
		t.Fatal("Expected running-2 to be stopped")
	}
}

func TestCleanupService_CleansStoppedInstances(t *testing.T) {
	driver := NewMockDriver()
	driver.AddInstance("stopped-1", "stopped")
	driver.AddInstance("exited-1", "exited")

	svc, err := NewCleanupService(driver, "test-node")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	err = svc.Execute(ctx, "test-node", "test-token")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have cleaned both stopped instances
	if len(driver.stopCalls) != 2 {
		t.Fatalf("Expected 2 stop calls, got %d", len(driver.stopCalls))
	}
}

func TestCleanupService_MixedInstances(t *testing.T) {
	driver := NewMockDriver()
	driver.AddInstance("running-1", "running")
	driver.AddInstance("running-2", "running")
	driver.AddInstance("stopped-1", "stopped")
	driver.AddInstance("exited-1", "exited")

	svc, err := NewCleanupService(driver, "test-node")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	err = svc.Execute(ctx, "test-node", "test-token")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have cleaned all 4 instances
	if len(driver.stopCalls) != 4 {
		t.Fatalf("Expected 4 stop calls, got %d", len(driver.stopCalls))
	}

	// All instances should be cleaned
	instances, err := driver.ListInstances(ctx)
	if err != nil {
		t.Fatalf("Failed to list instances: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("Expected 0 remaining instances, got %d", len(instances))
	}
}

func TestCleanupService_ContextCancellation(t *testing.T) {
	driver := NewMockDriver()
	driver.AddInstance("running-1", "running")

	svc, err := NewCleanupService(driver, "test-node")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// This should not crash, but may return an error from context
	err = svc.Execute(ctx, "test-node", "test-token")
	// Execution might succeed or fail depending on timing
	// The important thing is that it doesn't crash
	t.Logf("Execute result: %v (no crash expected)", err)
}

// Benchmark tests

func BenchmarkCleanupService_NoInstances(b *testing.B) {
	driver := NewMockDriver()
	svc, _ := NewCleanupService(driver, "test-node")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Execute(ctx, "test-node", "test-token")
	}
}

func BenchmarkCleanupService_TenInstances(b *testing.B) {
	driver := NewMockDriver()
	for i := 0; i < 10; i++ {
		driver.AddInstance(fmt.Sprintf("instance-%d", i), "running")
	}

	svc, _ := NewCleanupService(driver, "test-node")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Recreate instances for each iteration
		for j := 0; j < 10; j++ {
			driver.AddInstance(fmt.Sprintf("instance-%d", j), "running")
		}
		svc.Execute(ctx, "test-node", "test-token")
	}
}
