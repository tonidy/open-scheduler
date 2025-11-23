package rest

import (
	"testing"
)

func TestValidateJobRequest_ValidJob(t *testing.T) {
	req := &SubmitJobRequest{
		JobName:   "test-job",
		JobType:   "single",
		Driver:    "podman",
		WorkloadType: "container",
	}

	err := ValidateJobRequest(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestValidateJobRequest_MissingJobName(t *testing.T) {
	req := &SubmitJobRequest{
		JobName: "",
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for missing job name, got nil")
	}
	if err.Error() != "job_name is required" {
		t.Fatalf("Expected 'job_name is required', got: %v", err)
	}
}

func TestValidateJobRequest_JobNameTooLong(t *testing.T) {
	// Create a string longer than 255 characters
	longName := ""
	for i := 0; i < 256; i++ {
		longName += "a"
	}

	req := &SubmitJobRequest{
		JobName: longName,
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for long job name, got nil")
	}
	if err.Error() != "job_name must be 255 characters or less" {
		t.Fatalf("Expected 'job_name must be 255 characters or less', got: %v", err)
	}
}

func TestValidateJobRequest_InvalidJobName(t *testing.T) {
	req := &SubmitJobRequest{
		JobName: "job@#name", // Invalid characters
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid job name, got nil")
	}
	if err.Error() != "job_name contains invalid characters (use alphanumeric, dash, underscore)" {
		t.Fatalf("Unexpected error message: %v", err)
	}
}

func TestValidateJobRequest_InvalidJobType(t *testing.T) {
	req := &SubmitJobRequest{
		JobName: "test-job",
		JobType: "invalid_type",
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid job type, got nil")
	}
}

func TestValidateJobRequest_ValidJobType(t *testing.T) {
	validTypes := []string{"single", "service", "batch"}

	for _, jobType := range validTypes {
		req := &SubmitJobRequest{
			JobName: "test-job",
			JobType: jobType,
		}

		err := ValidateJobRequest(req)
		if err != nil {
			t.Fatalf("Expected no error for job type %s, got: %v", jobType, err)
		}
	}
}

func TestValidateJobRequest_InvalidDriver(t *testing.T) {
	req := &SubmitJobRequest{
		JobName: "test-job",
		Driver:  "invalid_driver",
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid driver, got nil")
	}
}

func TestValidateJobRequest_ValidDriver(t *testing.T) {
	validDrivers := []string{"podman", "incus", "process"}

	for _, driver := range validDrivers {
		req := &SubmitJobRequest{
			JobName: "test-job",
			Driver:  driver,
		}

		err := ValidateJobRequest(req)
		if err != nil {
			t.Fatalf("Expected no error for driver %s, got: %v", driver, err)
		}
	}
}

func TestValidateJobRequest_InvalidWorkloadType(t *testing.T) {
	req := &SubmitJobRequest{
		JobName:      "test-job",
		WorkloadType: "invalid_type",
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid workload type, got nil")
	}
}

func TestValidateJobRequest_NegativeTimeout(t *testing.T) {
	req := &SubmitJobRequest{
		JobName:        "test-job",
		TimeoutSeconds: -10,
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for negative timeout, got nil")
	}
	if err.Error() != "timeout_seconds cannot be negative" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateJobRequest_TimeoutTooSmall(t *testing.T) {
	req := &SubmitJobRequest{
		JobName:        "test-job",
		TimeoutSeconds: 5,
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for timeout < 10s, got nil")
	}
	if err.Error() != "timeout_seconds must be at least 10 seconds if specified" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateJobRequest_TimeoutTooLarge(t *testing.T) {
	req := &SubmitJobRequest{
		JobName:        "test-job",
		TimeoutSeconds: 86401, // 24 hours + 1 second
	}

	err := ValidateJobRequest(req)
	if err == nil {
		t.Fatal("Expected error for timeout > 24h, got nil")
	}
	if err.Error() != "timeout_seconds exceeds maximum (24 hours)" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateJobRequest_ValidTimeout(t *testing.T) {
	validTimeouts := []int64{0, 10, 3600, 86400}

	for _, timeout := range validTimeouts {
		req := &SubmitJobRequest{
			JobName:        "test-job",
			TimeoutSeconds: timeout,
		}

		err := ValidateJobRequest(req)
		if err != nil {
			t.Fatalf("Expected no error for timeout %d, got: %v", timeout, err)
		}
	}
}

func TestValidateResources_NegativeMemory(t *testing.T) {
	res := &ResourcesRequest{
		MemoryMB: -100,
	}

	err := ValidateResources(res)
	if err == nil {
		t.Fatal("Expected error for negative memory, got nil")
	}
	if err.Error() != "memory_mb cannot be negative" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateResources_MemoryTooLarge(t *testing.T) {
	res := &ResourcesRequest{
		MemoryMB: 1048577, // > 1TB
	}

	err := ValidateResources(res)
	if err == nil {
		t.Fatal("Expected error for memory > 1TB, got nil")
	}
}

func TestValidateResources_ReserveExceedsLimit(t *testing.T) {
	res := &ResourcesRequest{
		MemoryMB:      100,
		MemoryReserveMB: 200, // Reserve > limit
	}

	err := ValidateResources(res)
	if err == nil {
		t.Fatal("Expected error for reserve > limit, got nil")
	}
	if err.Error() != "memory_reserve_mb cannot exceed memory_mb" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateResources_NegativeCPU(t *testing.T) {
	res := &ResourcesRequest{
		CPU: -2.5,
	}

	err := ValidateResources(res)
	if err == nil {
		t.Fatal("Expected error for negative CPU, got nil")
	}
	if err.Error() != "cpu cannot be negative" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateResources_CPUTooLarge(t *testing.T) {
	res := &ResourcesRequest{
		CPU: 257, // > 256 cores
	}

	err := ValidateResources(res)
	if err == nil {
		t.Fatal("Expected error for CPU > 256 cores, got nil")
	}
}

func TestValidateResources_ValidResources(t *testing.T) {
	res := &ResourcesRequest{
		MemoryMB:        512,
		MemoryReserveMB: 256,
		CPU:             2.0,
		CPUReserve:      1.0,
	}

	err := ValidateResources(res)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestValidateVolume_MissingHostPath(t *testing.T) {
	vol := VolumeRequest{
		HostPath:     "",
		InstancePath: "/data",
	}

	err := ValidateVolume(vol, 0)
	if err == nil {
		t.Fatal("Expected error for missing host_path, got nil")
	}
	if err.Error() != "host_path is required" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateVolume_MissingInstancePath(t *testing.T) {
	vol := VolumeRequest{
		HostPath:     "/host/data",
		InstancePath: "",
	}

	err := ValidateVolume(vol, 0)
	if err == nil {
		t.Fatal("Expected error for missing instance_path, got nil")
	}
	if err.Error() != "instance_path is required" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateVolume_PathTraversal(t *testing.T) {
	tests := []struct {
		name        string
		hostPath    string
		instancePath string
	}{
		{"host path traversal", "/data/../etc/passwd", "/data"},
		{"instance path traversal", "/data", "/var/../etc/password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vol := VolumeRequest{
				HostPath:     tt.hostPath,
				InstancePath: tt.instancePath,
			}

			err := ValidateVolume(vol, 0)
			if err == nil {
				t.Fatal("Expected error for path traversal, got nil")
			}
		})
	}
}

func TestValidateVolume_ValidPath(t *testing.T) {
	vol := VolumeRequest{
		HostPath:     "/host/data",
		InstancePath: "/data",
	}

	err := ValidateVolume(vol, 0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestValidateInstanceSpec_MissingImage(t *testing.T) {
	spec := &InstanceSpecRequest{
		Image: "",
	}

	err := ValidateInstanceSpec(spec)
	if err == nil {
		t.Fatal("Expected error for missing image, got nil")
	}
	if err.Error() != "image is required when instance_config is provided" {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateInstanceSpec_ImageTooLong(t *testing.T) {
	longImage := ""
	for i := 0; i < 513; i++ {
		longImage += "a"
	}

	spec := &InstanceSpecRequest{
		Image: longImage,
	}

	err := ValidateInstanceSpec(spec)
	if err == nil {
		t.Fatal("Expected error for long image name, got nil")
	}
}

func TestValidateInstanceSpec_InvalidImageFormat(t *testing.T) {
	tests := []string{
		"image && echo pwned",
		"image; rm -rf /",
		"image | cat /etc/passwd",
		"image`whoami`",
	}

	for _, image := range tests {
		spec := &InstanceSpecRequest{
			Image: image,
		}

		err := ValidateInstanceSpec(spec)
		if err == nil {
			t.Fatalf("Expected error for malicious image: %s, got nil", image)
		}
	}
}

func TestValidateInstanceSpec_ValidImage(t *testing.T) {
	validImages := []string{
		"nginx:latest",
		"ubuntu:20.04",
		"registry.example.com/myapp:v1.0",
		"localhost/local-image",
	}

	for _, image := range validImages {
		spec := &InstanceSpecRequest{
			Image: image,
		}

		err := ValidateInstanceSpec(spec)
		if err != nil {
			t.Fatalf("Expected no error for image %s, got: %v", image, err)
		}
	}
}

// Benchmark tests for performance-critical functions

func BenchmarkValidateJobRequest(b *testing.B) {
	req := &SubmitJobRequest{
		JobName:   "bench-job",
		JobType:   "single",
		Driver:    "podman",
		WorkloadType: "container",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateJobRequest(req)
	}
}

func BenchmarkValidateResources(b *testing.B) {
	res := &ResourcesRequest{
		MemoryMB:        512,
		MemoryReserveMB: 256,
		CPU:             2.0,
		CPUReserve:      1.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateResources(res)
	}
}

func BenchmarkValidateVolume(b *testing.B) {
	vol := VolumeRequest{
		HostPath:     "/host/data",
		InstancePath: "/data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateVolume(vol, 0)
	}
}
