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
	nodesPrefix              = "/centro/nodes/"
	deploymentQueuePrefix     = "/centro/deployments/queue/"
	failDeploymentQueuePrefix = "/centro/deployments/fail-queue/"
	deploymentActivePrefix    = "/centro/deployments/active/"
	deploymentHistoryPrefix   = "/centro/deployments/history/"
	deploymentEventsPrefix    = "/centro/deployments/events/"
	instanceDataPrefix        = "/centro/deployments/instance_data/"
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

type DeploymentStatus struct {
	Deployment *pb.Deployment `json:"deployment"`
	NodeID     string         `json:"node_id"`
	Status     string         `json:"status"`
	Detail     string         `json:"detail"`
	UpdatedAt  time.Time      `json:"updated_at"`
	ClaimedAt  time.Time      `json:"claimed_at"`
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

func (s *Storage) EnqueueFailedDeployment(ctx context.Context, deployment *pb.Deployment) error {
	data, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	key := fmt.Sprintf("%s%s", failDeploymentQueuePrefix, deployment.DeploymentId)
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to enqueue failed deployment: %w", err)
	}

	return nil
}

func (s *Storage) GetQueueDeployments(ctx context.Context) ([]*pb.Deployment, error) {
	resp, err := s.client.Get(ctx, deploymentQueuePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get queue deployments: %w", err)
	}

	deployments := make([]*pb.Deployment, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var deployment pb.Deployment
		if err := json.Unmarshal(kv.Value, &deployment); err != nil {
			log.Printf("Failed to unmarshal deployment: %v", err)
			continue
		}
		deployments = append(deployments, &deployment)
	}
	return deployments, nil
}

func (s *Storage) DeleteFailedDeployment(ctx context.Context, deploymentID string) error {
	// Use prefix delete to remove the failed deployment entry (which has a timestamp suffix)
	keyPrefix := fmt.Sprintf("%s%s", failDeploymentQueuePrefix, deploymentID)
	_, err := s.client.Delete(ctx, keyPrefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to delete failed deployment: %w", err)
	}

	return nil
}

func (s *Storage) EnqueueDeployment(ctx context.Context, deployment *pb.Deployment) error {
	data, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	key := fmt.Sprintf("%s%s", deploymentQueuePrefix, deployment.DeploymentId)
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to enqueue deployment: %w", err)
	}

	return nil
}

func (s *Storage) DequeueDeployment(ctx context.Context) (*pb.Deployment, error) {
	resp, err := s.client.Get(ctx, deploymentQueuePrefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend), clientv3.WithLimit(1))
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment from queue: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var deployment pb.Deployment
	if err := json.Unmarshal(resp.Kvs[0].Value, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	_, err = s.client.Delete(ctx, string(resp.Kvs[0].Key))
	if err != nil {
		return nil, fmt.Errorf("failed to delete deployment from queue: %w", err)
	}

	return &deployment, nil
}

func (s *Storage) GetQueueLength(ctx context.Context) (int, error) {
	resp, err := s.client.Get(ctx, deploymentQueuePrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}

	return int(resp.Count), nil
}

func (s *Storage) SaveDeploymentActive(ctx context.Context, deploymentID string, status *DeploymentStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment status: %w", err)
	}

	key := deploymentActivePrefix + deploymentID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save active deployment: %w", err)
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
			DeploymentID: strings.TrimPrefix(string(kv.Key), instanceDataPrefix),
			InstanceName: rawInstance.InstanceName,
			Status:       rawInstance.Status,
			Created:      rawInstance.Created,
		}
		instances = append(instances, cleanInstance)
	}

	return instances, nil
}
func (s *Storage) GetDeploymentActive(ctx context.Context, deploymentID string) (*DeploymentStatus, error) {
	key := deploymentActivePrefix + deploymentID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get active deployment: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var status DeploymentStatus
	if err := json.Unmarshal(resp.Kvs[0].Value, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment status: %w", err)
	}

	return &status, nil
}

func (s *Storage) DeleteDeploymentActive(ctx context.Context, deploymentID string) error {
	key := deploymentActivePrefix + deploymentID
	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete active deployment: %w", err)
	}

	return nil
}

func (s *Storage) GetAllActiveDeployments(ctx context.Context) (map[string]*DeploymentStatus, error) {
	resp, err := s.client.Get(ctx, deploymentActivePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get active deployments: %w", err)
	}

	deployments := make(map[string]*DeploymentStatus)
	for _, kv := range resp.Kvs {
		var status DeploymentStatus
		if err := json.Unmarshal(kv.Value, &status); err != nil {
			log.Printf("Failed to unmarshal deployment status: %v", err)
			continue
		}
		deploymentID := strings.TrimPrefix(string(kv.Key), deploymentActivePrefix)
		deployments[deploymentID] = &status
	}

	return deployments, nil
}

func (s *Storage) SaveDeploymentHistory(ctx context.Context, deploymentID string, status *DeploymentStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment status: %w", err)
	}

	key := deploymentHistoryPrefix + deploymentID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save deployment history: %w", err)
	}

	return nil
}

func (s *Storage) GetDeploymentHistoryCount(ctx context.Context) (int, error) {
	resp, err := s.client.Get(ctx, deploymentHistoryPrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, fmt.Errorf("failed to get history count: %w", err)
	}

	return int(resp.Count), nil
}

func (s *Storage) GetActiveDeploymentCount(ctx context.Context) (int, error) {
	resp, err := s.client.Get(ctx, deploymentActivePrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, fmt.Errorf("failed to get active deployment count: %w", err)
	}

	return int(resp.Count), nil
}

func (s *Storage) GetAllDeploymentHistory(ctx context.Context) (map[string]*DeploymentStatus, error) {
	resp, err := s.client.Get(ctx, deploymentHistoryPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	deployments := make(map[string]*DeploymentStatus)
	for _, kv := range resp.Kvs {
		var status DeploymentStatus
		if err := json.Unmarshal(kv.Value, &status); err != nil {
			log.Printf("Failed to unmarshal deployment history: %v", err)
			continue
		}
		deploymentID := strings.TrimPrefix(string(kv.Key), deploymentHistoryPrefix)
		deployments[deploymentID] = &status
	}

	return deployments, nil
}

func (s *Storage) GetDeploymentHistory(ctx context.Context, deploymentID string) (*DeploymentStatus, error) {
	key := deploymentHistoryPrefix + deploymentID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var status DeploymentStatus
	if err := json.Unmarshal(resp.Kvs[0].Value, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment history: %w", err)
	}

	return &status, nil
}

func (s *Storage) SaveDeploymentEvent(ctx context.Context, deploymentID string, event string) error {
	// Remove timestamp (everything up to and including "] "), use the rest as the event message
	getMessage := func(ev string) string {
		if idx := strings.Index(ev, "] "); idx != -1 {
			return ev[idx+2:]
		}
		return ev
	}
	newMsg := getMessage(event)

	// Get the previous (most recent) event for this deployment, if any
	prefix := deploymentEventsPrefix + deploymentID + "/"
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend), clientv3.WithLimit(1))
	if err == nil && len(resp.Kvs) > 0 {
		prevMsg := getMessage(string(resp.Kvs[0].Value))
		if prevMsg == newMsg {
			return nil // skip duplicate event regardless of timestamp
		}
	}
	// if error above, fail open: allow event

	key := fmt.Sprintf("%s%s/%d", deploymentEventsPrefix, deploymentID, time.Now().UnixNano())
	_, err = s.client.Put(ctx, key, event)
	if err != nil {
		return fmt.Errorf("failed to save deployment event: %w", err)
	}
	return nil
}

func (s *Storage) GetDeploymentEvents(ctx context.Context, deploymentID string) ([]string, error) {
	prefix := deploymentEventsPrefix + deploymentID + "/"
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment events: %w", err)
	}

	events := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		events = append(events, string(kv.Value))
	}

	return events, nil
}

func (s *Storage) GetAllFailedDeployments(ctx context.Context) (map[string]*pb.Deployment, error) {
	resp, err := s.client.Get(ctx, failDeploymentQueuePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get all failed deployments: %w", err)
	}

	deployments := make(map[string]*pb.Deployment)
	for _, kv := range resp.Kvs {
		var deployment pb.Deployment
		if err := json.Unmarshal(kv.Value, &deployment); err != nil {
			log.Printf("Failed to unmarshal deployment: %v", err)
			continue
		}
		deployments[deployment.DeploymentId] = &deployment
	}

	return deployments, nil
}

func (s *Storage) SaveInstanceData(ctx context.Context, deploymentID string, instanceData *pb.InstanceData) error {
	data, err := json.Marshal(instanceData)
	if err != nil {
		return fmt.Errorf("failed to marshal instance data: %w", err)
	}

	key := instanceDataPrefix + deploymentID
	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to save instance data: %w", err)
	}
	return nil
}

func (s *Storage) GetInstanceData(ctx context.Context, deploymentID string) (*pb.InstanceData, error) {
	key := instanceDataPrefix + deploymentID
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


func (s *Storage) GetQueueDeployment(ctx context.Context, deploymentID string) (*pb.Deployment, error) {
	key := deploymentQueuePrefix + deploymentID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue deployment: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var deployment pb.Deployment
	if err := json.Unmarshal(resp.Kvs[0].Value, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return &deployment, nil
}

func (s *Storage) GetFailedDeployment(ctx context.Context, deploymentID string) (*pb.Deployment, error) {
	key := failDeploymentQueuePrefix + deploymentID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed deployment: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var deployment pb.Deployment
	if err := json.Unmarshal(resp.Kvs[0].Value, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return &deployment, nil
}