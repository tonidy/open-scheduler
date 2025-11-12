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
	nodesPrefix        = "/centro/nodes/"
	jobQueuePrefix     = "/centro/jobs/queue/"
	failJobQueuePrefix = "/centro/jobs/fail-queue/"
	jobActivePrefix    = "/centro/jobs/active/"
	jobHistoryPrefix   = "/centro/jobs/history/"
	jobEventsPrefix    = "/centro/jobs/events/"
	instanceDataPrefix = "/centro/jobs/instance_data/"
)

type Storage struct {
	client *clientv3.Client
}

type NodeInfo struct {
	NodeID        string            `json:"node_id"`
	ClusterName   string            `json:"cluster_name"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	RamMB         float32           `json:"ram_mb"`
	CPUCores      float32           `json:"cpu_cores"`
	DiskMB        float32           `json:"disk_mb"`
	Metadata      map[string]string `json:"metadata"`
}

func (n *NodeInfo) IsHealthy() bool {
	return time.Since(n.LastHeartbeat) < 60*time.Second
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

func (s *Storage) EnqueueFailedJob(ctx context.Context, job *pb.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	key := fmt.Sprintf("%s%s", failJobQueuePrefix, job.JobId)
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to enqueue failed job: %w", err)
	}

	return nil
}

func (s *Storage) GetQueueJobs(ctx context.Context) ([]*pb.Job, error) {
	resp, err := s.client.Get(ctx, jobQueuePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get queue jobs: %w", err)
	}

	jobs := make([]*pb.Job, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var job pb.Job
		if err := json.Unmarshal(kv.Value, &job); err != nil {
			log.Printf("Failed to unmarshal job: %v", err)
			continue
		}
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

func (s *Storage) DeleteFailedJob(ctx context.Context, jobID string) error {
	// Use prefix delete to remove the failed job entry (which has a timestamp suffix)
	keyPrefix := fmt.Sprintf("%s%s", failJobQueuePrefix, jobID)
	_, err := s.client.Delete(ctx, keyPrefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to delete failed job: %w", err)
	}

	return nil
}

func (s *Storage) EnqueueJob(ctx context.Context, job *pb.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	key := fmt.Sprintf("%s%s", jobQueuePrefix, job.JobId)
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

func (s *Storage) GetListOfInstances(ctx context.Context) ([]InstanceItem, error) {
	resp, err := s.client.Get(ctx, instanceDataPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get all instances: %w", err)
	}

	instances := make([]InstanceItem, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var rawInstance pb.InstanceData
		if err := json.Unmarshal(kv.Value, &rawInstance); err != nil {
			log.Printf("Failed to unmarshal instance: %v", err)
			continue
		}

		cleanInstance := InstanceItem{
			JobID:        strings.TrimPrefix(string(kv.Key), instanceDataPrefix),
			InstanceName: rawInstance.InstanceName,
			Status:       rawInstance.Status,
			Created:      rawInstance.Created,
		}
		instances = append(instances, cleanInstance)
	}

	return instances, nil
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

func (s *Storage) GetAllJobHistory(ctx context.Context) (map[string]*JobStatus, error) {
	resp, err := s.client.Get(ctx, jobHistoryPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get job history: %w", err)
	}

	jobs := make(map[string]*JobStatus)
	for _, kv := range resp.Kvs {
		var status JobStatus
		if err := json.Unmarshal(kv.Value, &status); err != nil {
			log.Printf("Failed to unmarshal job history: %v", err)
			continue
		}
		jobID := strings.TrimPrefix(string(kv.Key), jobHistoryPrefix)
		jobs[jobID] = &status
	}

	return jobs, nil
}

func (s *Storage) GetJobHistory(ctx context.Context, jobID string) (*JobStatus, error) {
	key := jobHistoryPrefix + jobID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get job history: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var status JobStatus
	if err := json.Unmarshal(resp.Kvs[0].Value, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job history: %w", err)
	}

	return &status, nil
}

func (s *Storage) SaveJobEvent(ctx context.Context, jobID string, event string) error {
	// Remove timestamp (everything up to and including "] "), use the rest as the event message
	getMessage := func(ev string) string {
		if idx := strings.Index(ev, "] "); idx != -1 {
			return ev[idx+2:]
		}
		return ev
	}
	newMsg := getMessage(event)

	// Get the previous (most recent) event for this job, if any
	prefix := jobEventsPrefix + jobID + "/"
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend), clientv3.WithLimit(1))
	if err == nil && len(resp.Kvs) > 0 {
		prevMsg := getMessage(string(resp.Kvs[0].Value))
		if prevMsg == newMsg {
			return nil // skip duplicate event regardless of timestamp
		}
	}
	// if error above, fail open: allow event

	key := fmt.Sprintf("%s%s/%d", jobEventsPrefix, jobID, time.Now().UnixNano())
	_, err = s.client.Put(ctx, key, event)
	if err != nil {
		return fmt.Errorf("failed to save job event: %w", err)
	}
	return nil
}

func (s *Storage) GetJobEvents(ctx context.Context, jobID string) ([]string, error) {
	prefix := jobEventsPrefix + jobID + "/"
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return nil, fmt.Errorf("failed to get job events: %w", err)
	}

	events := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		events = append(events, string(kv.Value))
	}

	return events, nil
}

func (s *Storage) GetAllFailedJobs(ctx context.Context) (map[string]*pb.Job, error) {
	resp, err := s.client.Get(ctx, failJobQueuePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get all failed jobs: %w", err)
	}

	jobs := make(map[string]*pb.Job)
	for _, kv := range resp.Kvs {
		var job pb.Job
		if err := json.Unmarshal(kv.Value, &job); err != nil {
			log.Printf("Failed to unmarshal job: %v", err)
			continue
		}
		jobs[job.JobId] = &job
	}

	return jobs, nil
}

func (s *Storage) SaveInstanceData(ctx context.Context, jobID string, instanceData *pb.InstanceData) error {
	data, err := json.Marshal(instanceData)
	if err != nil {
		return fmt.Errorf("failed to marshal instance data: %w", err)
	}

	key := instanceDataPrefix + jobID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save instance data: %w", err)
	}
	return nil
}

func (s *Storage) GetInstanceData(ctx context.Context, jobID string) (*pb.InstanceData, error) {
	key := instanceDataPrefix + jobID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance data: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var instanceData pb.InstanceData
	if err := json.Unmarshal(resp.Kvs[0].Value, &instanceData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance data: %w", err)
	}

	return &instanceData, nil
}


func (s *Storage) GetQueueJob(ctx context.Context, jobID string) (*pb.Job, error) {
	key := jobQueuePrefix + jobID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue job: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var job pb.Job
	if err := json.Unmarshal(resp.Kvs[0].Value, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

func (s *Storage) GetFailedJob(ctx context.Context, jobID string) (*pb.Job, error) {
	key := failJobQueuePrefix + jobID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed job: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var job pb.Job
	if err := json.Unmarshal(resp.Kvs[0].Value, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}