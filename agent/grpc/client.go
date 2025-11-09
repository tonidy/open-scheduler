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

type SharedClient struct {
	serverAddr string
	conn       *grpc.ClientConn
	client     pb.NodeAgentServiceClient
	mu         sync.RWMutex
}

func NewSharedClient(serverAddr string) (*SharedClient, error) {
	if serverAddr == "" {
		return nil, fmt.Errorf("server address cannot be empty")
	}

	return &SharedClient{
		serverAddr: serverAddr,
	}, nil
}

func (c *SharedClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.client != nil {
		log.Printf("[SharedClient] Already connected to server")
		return nil
	}

	log.Printf("[SharedClient] Connecting to server at %s", c.serverAddr)

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
	c.client = pb.NewNodeAgentServiceClient(conn)
	log.Printf("[SharedClient] Successfully connected to server")

	return nil
}

func (c *SharedClient) SendHeartbeat(ctx context.Context, nodeID string, token string, ramMB, cpuPercent, diskMB float32, meta map[string]string) (*pb.HeartbeatResponse, error) {
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
		NodeId:     nodeID,
		Timestamp:  time.Now().Unix(),
		RamMb:      ramMB,
		CpuPercent: cpuPercent,
		DiskMb:     diskMB,
		Metadata:   meta,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.Heartbeat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("heartbeat RPC failed: %w", err)
	}

	log.Printf("[SharedClient] Heartbeat response: ok=%v, message=%s", resp.Ok, resp.Message)

	return resp, nil
}

func (c *SharedClient) GetJob(ctx context.Context, nodeID string, token string) (*pb.GetJobResponse, error) {
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

	if resp.HasJob {
		log.Printf("[SharedClient] Received job: job_id=%s, name=%s, type=%s",
			resp.Job.JobId, resp.Job.Name, resp.Job.Type)
	} else {
		log.Printf("[SharedClient] No job available: %s", resp.Message)
	}

	return resp, nil
}

func (c *SharedClient) ClaimJob(ctx context.Context, nodeID string, jobID string, token string) (*pb.ClaimJobResponse, error) {
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

	req := &pb.ClaimJobRequest{
		NodeId: nodeID,
		JobId: jobID,
	}

	resp, err := client.ClaimJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ClaimJob RPC failed: %w", err)
	}

	log.Printf("[SharedClient] ClaimJob response: ok=%v, message=%s", resp.Ok, resp.Message)

	return resp, nil
}

func (c *SharedClient) UpdateStatus(ctx context.Context, nodeID string, token string, jobID string, status string, detail string, timestamp int64) (*pb.UpdateStatusResponse, error) {
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
		NodeId:    nodeID,
		JobId:     jobID,
		Status:    status,
		Detail:    detail,
		Timestamp: timestamp,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.UpdateStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("UpdateStatus RPC failed: %w", err)
	}

	log.Printf("[SharedClient] UpdateStatus response: ok=%v, message=%s", resp.Ok, resp.Message)

	return resp, nil
}

func (c *SharedClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		log.Printf("[SharedClient] Closing connection")
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}

func (c *SharedClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && c.client != nil
}
