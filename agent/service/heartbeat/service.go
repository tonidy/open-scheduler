package heartbeat

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"syscall"

	sharedgrpc "github.com/open-scheduler/agent/grpc"
)

type HeartbeatService struct {
	grpcClient *sharedgrpc.SharedClient
}

func NewHeartbeatService(grpcClient *sharedgrpc.SharedClient) (*HeartbeatService, error) {
	if grpcClient == nil {
		return nil, fmt.Errorf("gRPC client cannot be nil")
	}

	return &HeartbeatService{
		grpcClient: grpcClient,
	}, nil
}

func getAvailableMemoryMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	totalMemMB := float64(m.Sys) / 1024 / 1024
	usedMemMB := float64(m.Alloc) / 1024 / 1024
	return totalMemMB - usedMemMB
}

func getCPUUsagePercent() float64 {
	return float64(runtime.NumCPU()) * 100.0 / float64(runtime.NumCPU())
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
	log.Printf("[HeartbeatService] Sending heartbeat for node: %s", nodeID)

	metadata := map[string]string{
		"version": "1.0.0",
		"region":  "us-west-1",
	}

	ramAvailable := getAvailableMemoryMB()
	cpuAvailable := getCPUUsagePercent()
	diskAvailable := getAvailableDiskMB()

	log.Printf("[HeartbeatService] System metrics - RAM: %.2f MB, CPU: %.2f%%, Disk: %.2f MB",
		ramAvailable, cpuAvailable, diskAvailable)

	resp, err := h.grpcClient.SendHeartbeat(
		ctx,
		nodeID,
		token,
		float32(ramAvailable),
		float32(cpuAvailable),
		float32(diskAvailable),
		metadata,
	)

	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	if !resp.Ok {
		return fmt.Errorf("heartbeat rejected: %s", resp.Message)
	}

	log.Printf("[HeartbeatService] Heartbeat successful: %s", resp.Message)
	return nil
}
