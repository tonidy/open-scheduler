package heartbeat

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
)

type HeartbeatService struct {
	grpcClient *sharedgrpc.GrpcClient
}

func NewHeartbeatService(grpcClient *sharedgrpc.GrpcClient) (*HeartbeatService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &HeartbeatService{
		grpcClient: grpcClient,
	}, nil
}

func getAvailableMemoryMB() float64 {
	// Try to get actual system memory (macOS/BSD)
	if runtime.GOOS == "darwin" {
		// Get total physical memory
		totalBytes, err := syscall.Sysctl("hw.memsize")
		if err == nil && len(totalBytes) > 0 {
			// Parse the bytes as uint64
			var total uint64
			for i := 0; i < 8 && i < len(totalBytes); i++ {
				total |= uint64(totalBytes[i]) << (uint(i) * 8)
			}

			// Get VM statistics for free memory
			// Note: This is a simplified approach - on macOS, "available" memory
			// is complex (includes cached, compressed, etc.)
			// For production, consider using vm_stat or a library like gopsutil

			// For now, we'll use a heuristic: return a reasonable portion of total memory
			// as "available" since getting exact free memory on macOS requires more complex syscalls
			totalMB := float64(total) / 1024 / 1024

			// Estimate available as 20% of total (conservative estimate)
			// In a real system, you'd want to parse vm_stat output or use proper syscalls
			return totalMB * 0.2
		}
	}

	// Fallback for Linux - read /proc/meminfo
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			lines := string(data)
			// Parse MemAvailable if present (more accurate than MemFree)
			for _, line := range strings.Split(lines, "\n") {
				if strings.HasPrefix(line, "MemAvailable:") {
					var available uint64
					fmt.Sscanf(line, "MemAvailable: %d kB", &available)
					return float64(available) / 1024 // Convert KB to MB
				}
			}
		}
	}

	// Fallback to runtime stats (not accurate for system memory, but better than nothing)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	totalMemMB := float64(m.Sys) / 1024 / 1024
	usedMemMB := float64(m.Alloc) / 1024 / 1024
	return totalMemMB - usedMemMB
}

func getAvailableCPUCores() float64 {
	return float64(runtime.NumCPU())
}

func getAvailableDiskMB() float64 {
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return 0
	}

	err = syscall.Statfs(wd, &stat)
	if err != nil {
		return 0
	}

	availableBytes := stat.Bavail * uint64(stat.Bsize)
	availableMB := float64(availableBytes) / 1024 / 1024
	return availableMB
}

func (h *HeartbeatService) Execute(ctx context.Context, nodeID string, token string) error {
	// Get cluster name from environment variable, default to "default"
	clusterName := os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		clusterName = "default"
	}

	metadata := map[string]string{
		"version": "1.0.0",
		"region":  "us-west-1",
	}

	ramAvailable := getAvailableMemoryMB()
	cpuAvailable := getAvailableCPUCores()
	diskAvailable := getAvailableDiskMB()

	resp, err := h.grpcClient.SendHeartbeat(
		ctx,
		nodeID,
		token,
		float32(ramAvailable),
		float32(cpuAvailable),
		float32(diskAvailable),
		clusterName,
		metadata,
	)

	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	if !resp.Acknowledged {
		return fmt.Errorf("heartbeat rejected: %s", resp.ResponseMessage)
	}

	// Only log errors, success is assumed
	return nil
}
