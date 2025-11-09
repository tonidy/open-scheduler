package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	pb "github.com/open-scheduler/proto"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	nodesPrefix       = "/centro/nodes/"
	jobQueuePrefix    = "/centro/jobs/queue/"
	jobActivePrefix   = "/centro/jobs/active/"
	jobHistoryPrefix  = "/centro/jobs/history/"
)

type Storage struct {
	client *clientv3.Client
}

type NodeInfo struct {
	NodeID        string            `json:"node_id"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	RamMB         float32           `json:"ram_mb"`
	CPUPercent    float32           `json:"cpu_percent"`
	DiskMB        float32           `json:"disk_mb"`
	Metadata      map[string]string `json:"metadata"`
}

type JobStatus struct {
	Job       *pb.Job   `json:"job"`
	NodeID    string    `json:"node_id"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	UpdatedAt time.Time `json:"updated_at"`
	ClaimedAt time.Time `json:"claimed_at"`
}

func NewStorage(endpoints []string) (*Storage, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &Storage{client: cli}, nil
}

func (s *Storage) Close() error {
	return s.client.Close()
}

func (s *Storage) SaveNode(ctx context.Context, node *NodeInfo) error {
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	key := nodesPrefix + node.NodeID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save node to etcd: %w", err)
	}

	return nil
}

func (s *Storage) GetNode(ctx context.Context, nodeID string) (*NodeInfo, error) {
	key := nodesPrefix + nodeID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get node from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var node NodeInfo
	if err := json.Unmarshal(resp.Kvs[0].Value, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	return &node, nil
}

func (s *Storage) GetAllNodes(ctx context.Context) (map[string]*NodeInfo, error) {
	resp, err := s.client.Get(ctx, nodesPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes from etcd: %w", err)
	}

	nodes := make(map[string]*NodeInfo)
	for _, kv := range resp.Kvs {
		var node NodeInfo
		if err := json.Unmarshal(kv.Value, &node); err != nil {
			log.Printf("Failed to unmarshal node: %v", err)
			continue
		}
		nodes[node.NodeID] = &node
	}

	return nodes, nil
}

func (s *Storage) EnqueueJob(ctx context.Context, job *pb.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	key := fmt.Sprintf("%s%s-%d", jobQueuePrefix, job.JobId, time.Now().UnixNano())
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

func (s *Storage) DequeueJob(ctx context.Context) (*pb.Job, error) {
	resp, err := s.client.Get(ctx, jobQueuePrefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend), clientv3.WithLimit(1))
	if err != nil {
		return nil, fmt.Errorf("failed to get job from queue: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var job pb.Job
	if err := json.Unmarshal(resp.Kvs[0].Value, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	_, err = s.client.Delete(ctx, string(resp.Kvs[0].Key))
	if err != nil {
		return nil, fmt.Errorf("failed to delete job from queue: %w", err)
	}

	return &job, nil
}

func (s *Storage) GetQueueLength(ctx context.Context) (int, error) {
	resp, err := s.client.Get(ctx, jobQueuePrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}

	return int(resp.Count), nil
}

func (s *Storage) SaveJobActive(ctx context.Context, jobID string, status *JobStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal job status: %w", err)
	}

	key := jobActivePrefix + jobID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save active job: %w", err)
	}

	return nil
}

func (s *Storage) GetJobActive(ctx context.Context, jobID string) (*JobStatus, error) {
	key := jobActivePrefix + jobID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get active job: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var status JobStatus
	if err := json.Unmarshal(resp.Kvs[0].Value, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job status: %w", err)
	}

	return &status, nil
}

func (s *Storage) DeleteJobActive(ctx context.Context, jobID string) error {
	key := jobActivePrefix + jobID
	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete active job: %w", err)
	}

	return nil
}

func (s *Storage) GetAllActiveJobs(ctx context.Context) (map[string]*JobStatus, error) {
	resp, err := s.client.Get(ctx, jobActivePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get active jobs: %w", err)
	}

	jobs := make(map[string]*JobStatus)
	for _, kv := range resp.Kvs {
		var status JobStatus
		if err := json.Unmarshal(kv.Value, &status); err != nil {
			log.Printf("Failed to unmarshal job status: %v", err)
			continue
		}
		jobID := strings.TrimPrefix(string(kv.Key), jobActivePrefix)
		jobs[jobID] = &status
	}

	return jobs, nil
}

func (s *Storage) SaveJobHistory(ctx context.Context, jobID string, status *JobStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal job status: %w", err)
	}

	key := jobHistoryPrefix + jobID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save job history: %w", err)
	}

	return nil
}

func (s *Storage) GetJobHistoryCount(ctx context.Context) (int, error) {
	resp, err := s.client.Get(ctx, jobHistoryPrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, fmt.Errorf("failed to get history count: %w", err)
	}

	return int(resp.Count), nil
}

func (s *Storage) GetActiveJobCount(ctx context.Context) (int, error) {
	resp, err := s.client.Get(ctx, jobActivePrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, fmt.Errorf("failed to get active job count: %w", err)
	}

	return int(resp.Count), nil
}

