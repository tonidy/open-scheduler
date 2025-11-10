package grpc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/open-scheduler/proto"
)

type GrpcClient struct {
	serverAddr string
	conn       *grpc.ClientConn
	client     pb.CentroSchedulerServiceClient
	mu         sync.RWMutex
}

func NewGrpcClient(serverAddr string) (*GrpcClient, error) {
	if serverAddr == "" {
		return nil, fmt.Errorf("server address cannot be empty")
	}

	return &GrpcClient{
		serverAddr: serverAddr,
	}, nil
}

func (c *GrpcClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.client != nil {
		log.Printf("[GrpcClient] Already connected to server")
		return nil
	}

	log.Printf("[GrpcClient] Connecting to server at %s", c.serverAddr)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	conn, err := grpc.DialContext(ctx, c.serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	c.conn = conn
	c.client = pb.NewCentroSchedulerServiceClient(conn)
	log.Printf("[GrpcClient] Successfully connected to server")

	return nil
}

func (c *GrpcClient) SendHeartbeat(ctx context.Context, nodeID string, token string, ramMB, cpuCores, diskMB float32, clusterName string, meta map[string]string) (*pb.HeartbeatResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("gRPC client is not connected")
	}

	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", token),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &pb.HeartbeatRequest{
		NodeId:            nodeID,
		Timestamp:         time.Now().Unix(),
		AvailableMemoryMb: ramMB,
		AvailableCpuCores: cpuCores,
		AvailableDiskMb:   diskMB,
		NodeMetadata:      meta,
		ClusterName:       clusterName,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.Heartbeat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("heartbeat RPC failed: %w", err)
	}

	// Response logged by HeartbeatService
	return resp, nil
}

func (c *GrpcClient) GetJob(ctx context.Context, nodeID string, token string) (*pb.GetJobResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("gRPC client is not connected")
	}

	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", token),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &pb.GetJobRequest{
		NodeId: nodeID,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.GetJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("GetJob RPC failed: %w", err)
	}

	// Job details logged by GetJobService
	return resp, nil
}

func (c *GrpcClient) UpdateStatus(ctx context.Context, nodeID string, token string, jobID string, status string, detail string, timestamp int64) (*pb.UpdateStatusResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("gRPC client is not connected")
	}

	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", token),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &pb.UpdateStatusRequest{
		NodeId:        nodeID,
		JobId:         jobID,
		JobStatus:     status,
		StatusMessage: detail,
		Timestamp:     timestamp,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.UpdateStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("UpdateStatus RPC failed: %w", err)
	}

	log.Printf("[GrpcClient] UpdateStatus response: acknowledged=%v, message=%s", resp.Acknowledged, resp.ResponseMessage)

	return resp, nil
}

func (c *GrpcClient) SetContainerData(ctx context.Context, nodeID string, token string, jobID string, containerData *pb.ContainerData, timestamp int64) (*pb.SetContainerDataResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("gRPC client is not connected")
	}

	md := metadata.New(map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", token),
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	req := &pb.SetContainerDataRequest{
		NodeId:        nodeID,
		JobId:         jobID,
		ContainerData: containerData,
		Timestamp:     timestamp,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.SetContainerData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("SetContainerData RPC failed: %w", err)
	}

	log.Printf("[GrpcClient] SetContainerData response: acknowledged=%v, message=%s", resp.Acknowledged, resp.ResponseMessage)

	return resp, nil
}

func (c *GrpcClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		log.Printf("[GrpcClient] Closing connection")
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}

func (c *GrpcClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && c.client != nil
}
