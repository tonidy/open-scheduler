package grpc

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	etcdstorage "github.com/open-scheduler/centro/storage/etcd"
	pb "github.com/open-scheduler/proto"
)

type CentroServer struct {
	pb.UnimplementedCentroSchedulerServiceServer
	storage *etcdstorage.Storage
}

func NewCentroServer(storage *etcdstorage.Storage) *CentroServer {
	server := &CentroServer{
		storage: storage,
	}

	go server.monitorNodes()

	return server
}

func (s *CentroServer) monitorNodes() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		nodes, err := s.storage.GetAllNodes(ctx)
		if err != nil {
			log.Printf("[Centro] Failed to get nodes for monitoring: %v", err)
			continue
		}

		now := time.Now()
		for nodeID, node := range nodes {
			if now.Sub(node.LastHeartbeat) > 60*time.Second {
				log.Printf("[Centro] Node %s appears to be offline (last heartbeat: %v)",
					nodeID, node.LastHeartbeat)
			}
		}
	}
}

// calculateDeploymentResourceRequirements calculates the resource requirements for the deployment
// Returns: (cpuCores, ramMB, diskMB)
func calculateDeploymentResourceRequirements(deployment *pb.Deployment) (float32, float32, float32) {
	var totalCPU float32
	var totalRAM float32
	var totalDisk float32 = 0 // Disk requirements are not typically specified, but we'll keep this for consistency

	if deployment.ResourceRequirements != nil {
		// Use cpu_limit_cores if set, otherwise use cpu_reserved_cores
		if deployment.ResourceRequirements.CpuLimitCores > 0 {
			totalCPU = float32(deployment.ResourceRequirements.CpuLimitCores)
		} else if deployment.ResourceRequirements.CpuReservedCores > 0 {
			totalCPU = float32(deployment.ResourceRequirements.CpuReservedCores)
		}

		// Use memory_limit_mb if set, otherwise use memory_reserved_mb
		if deployment.ResourceRequirements.MemoryLimitMb > 0 {
			totalRAM = float32(deployment.ResourceRequirements.MemoryLimitMb)
		} else if deployment.ResourceRequirements.MemoryReservedMb > 0 {
			totalRAM = float32(deployment.ResourceRequirements.MemoryReservedMb)
		}
	}

	return totalCPU, totalRAM, totalDisk
}

// nodeHasSufficientResources checks if a node has enough available resources for a deployment
// Note: Node.CPUCores is available CPU in cores, Node.RamMB and Node.DiskMB are available amounts
func nodeHasSufficientResources(node *etcdstorage.NodeInfo, requiredCPUCores, requiredRAMMB, requiredDiskMB float32) bool {
	// Check CPU availability (direct comparison in cores)
	if requiredCPUCores > node.CPUCores {
		log.Printf("[Centro] Insufficient CPU: required %.2f cores, available %.2f cores",
			requiredCPUCores, node.CPUCores)
		return false
	}

	// Check RAM availability
	if requiredRAMMB > node.RamMB {
		log.Printf("[Centro] Insufficient RAM: required %.2fMB, available %.2fMB",
			requiredRAMMB, node.RamMB)
		return false
	}

	// Check Disk availability (only if required)
	if requiredDiskMB > 0 && requiredDiskMB > node.DiskMB {
		log.Printf("[Centro] Insufficient Disk: required %.2fMB, available %.2fMB",
			requiredDiskMB, node.DiskMB)
		return false
	}

	return true
}

// handleDeploymentRejection handles when a deployment is rejected by a node, checks if any other nodes could take it,
// and saves detailed rejection events if no nodes are suitable
func (s *CentroServer) handleDeploymentRejection(ctx context.Context, deployment *pb.Deployment, rejectedByNodeID, rejectionReason string, requiredCPU, requiredRAM, requiredDisk float32) {
	// Get all nodes to check if any could potentially take this deployment
	allNodes, err := s.storage.GetAllNodes(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get all nodes: %v", err)
		// Re-queue the deployment anyway
		if err := s.storage.EnqueueDeployment(ctx, deployment); err != nil {
			log.Printf("[Centro] Failed to re-queue deployment: %v", err)
		}
		return
	}

	// Build a list of rejection reasons for all nodes
	rejectionReasons := make(map[string]string)
	rejectionReasons[rejectedByNodeID] = rejectionReason

	hasHealthyMatchingNode := false
	for nodeID, node := range allNodes {
		// Skip the node that just rejected it (already have its reason)
		if nodeID == rejectedByNodeID {
			continue
		}

		// Check if node is healthy
		if !node.IsHealthy() {
			rejectionReasons[nodeID] = fmt.Sprintf("Node unhealthy (last heartbeat: %v)", node.LastHeartbeat)
			continue
		}

		// Check cluster match
		if len(deployment.SelectedClusters) > 0 {
			clusterMatches := false
			for _, cluster := range deployment.SelectedClusters {
				if cluster == node.ClusterName {
					clusterMatches = true
					break
				}
			}
			if !clusterMatches {
				rejectionReasons[nodeID] = fmt.Sprintf("Cluster mismatch: deployment requires %v, node is in '%s'",
					deployment.SelectedClusters, node.ClusterName)
				continue
			}
		}

		// Check resources
		if !nodeHasSufficientResources(node, requiredCPU, requiredRAM, requiredDisk) {
			rejectionReasons[nodeID] = fmt.Sprintf("Insufficient resources: deployment needs CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB; node has CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB",
				requiredCPU, requiredRAM, requiredDisk, node.CPUCores, node.RamMB, node.DiskMB)
			continue
		}

		// This node could potentially take the deployment
		hasHealthyMatchingNode = true
		break
	}

	// If no nodes can take this deployment, save a detailed event
	if !hasHealthyMatchingNode {
		var eventMessage strings.Builder
		eventMessage.WriteString(fmt.Sprintf("[%s] No matching nodes available for deployment %s\n",
			time.Now().Format(time.RFC3339), deployment.DeploymentId))
		eventMessage.WriteString(fmt.Sprintf("Deployment requirements: CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB",
			requiredCPU, requiredRAM, requiredDisk))
		if len(deployment.SelectedClusters) > 0 {
			eventMessage.WriteString(fmt.Sprintf(", Clusters=%v", deployment.SelectedClusters))
		}
		eventMessage.WriteString("\n\nRejection reasons by node:\n")

		for nodeID, reason := range rejectionReasons {
			eventMessage.WriteString(fmt.Sprintf("  - Node '%s': %s\n", nodeID, reason))
		}

		if err := s.storage.SaveDeploymentEvent(ctx, deployment.DeploymentId, eventMessage.String()); err != nil {
			log.Printf("[Centro] Failed to save 'no matching nodes' event: %v", err)
		}

		log.Printf("[Centro] Deployment %s (retry %d/%d) has no matching nodes. Moving to failed queue.",
			deployment.DeploymentId, deployment.RetryCount, deployment.MaxRetries)

		if err := s.storage.EnqueueFailedDeployment(ctx, deployment); err != nil {
			log.Printf("[Centro] Failed to enqueue failed deployment %s: %v", deployment.DeploymentId, err)
		} else {
			log.Printf("[Centro] Deployment %s moved to failed queue for later retry", deployment.DeploymentId)
		}
	} else {

		if err := s.storage.EnqueueDeployment(ctx, deployment); err != nil {
			log.Printf("[Centro] Failed to re-queue deployment: %v", err)
		}
	}
}

func (s *CentroServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if req.NodeId == "" {
		return &pb.HeartbeatResponse{
			Acknowledged:    false,
			ResponseMessage: "node_id is required",
		}, nil
	}

	node, err := s.storage.GetNode(ctx, req.NodeId)
	if err != nil {
		log.Printf("[Centro] Failed to get node: %v", err)
		return &pb.HeartbeatResponse{
			Acknowledged:    false,
			ResponseMessage: "Failed to get node info",
		}, nil
	}

	if node == nil {
		log.Printf("[Centro] New node registered: %s (cluster: %s)", req.NodeId, req.ClusterName)
		node = &etcdstorage.NodeInfo{
			NodeID:      req.NodeId,
			ClusterName: req.ClusterName,
		}
	}

	node.LastHeartbeat = time.Now()
	node.ClusterName = req.ClusterName
	node.RamMB = req.AvailableMemoryMb
	node.CPUCores = req.AvailableCpuCores
	node.DiskMB = req.AvailableDiskMb
	node.Metadata = req.NodeMetadata

	if err := s.storage.SaveNode(ctx, node); err != nil {
		log.Printf("[Centro] Failed to save node: %v", err)
		return &pb.HeartbeatResponse{
			Acknowledged:    false,
			ResponseMessage: "Failed to save node info",
		}, nil
	}

	log.Printf("[Centro] Heartbeat from node %s - CPU: %.2f cores, RAM: %.2fMB, Disk: %.2fMB",
		req.NodeId, req.AvailableCpuCores, req.AvailableMemoryMb, req.AvailableDiskMb)

	return &pb.HeartbeatResponse{
		Acknowledged:    true,
		ResponseMessage: "Heartbeat received",
	}, nil
}

func (s *CentroServer) GetDeployment(ctx context.Context, req *pb.GetDeploymentRequest) (*pb.GetDeploymentResponse, error) {
	if req.NodeId == "" {
		return &pb.GetDeploymentResponse{
			DeploymentAvailable:    false,
			ResponseMessage: "node_id is required",
		}, nil
	}

	node, err := s.storage.GetNode(ctx, req.NodeId)
	if err != nil {
		log.Printf("[Centro] Failed to get node: %v", err)
		return &pb.GetDeploymentResponse{
			DeploymentAvailable:    false,
			ResponseMessage: "Failed to get node info",
		}, nil
	}

	if node == nil {
		return &pb.GetDeploymentResponse{
			DeploymentAvailable:    false,
			ResponseMessage: "Node not registered. Send a heartbeat first.",
		}, nil
	}

	// Check if node is healthy based on last heartbeat
	if !node.IsHealthy() {
		log.Printf("[Centro] Node %s is not healthy (last heartbeat: %v) - rejecting deployment request",
			req.NodeId, node.LastHeartbeat)
		return &pb.GetDeploymentResponse{
			DeploymentAvailable:    false,
			ResponseMessage: fmt.Sprintf("Node is not healthy. Last heartbeat: %v", node.LastHeartbeat),
		}, nil
	}

	deployment, err := s.storage.DequeueDeployment(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to dequeue deployment: %v", err)
		return &pb.GetDeploymentResponse{
			DeploymentAvailable:    false,
			ResponseMessage: "Failed to get deployment from queue",
		}, nil
	}

	if deployment == nil {
		return &pb.GetDeploymentResponse{
			DeploymentAvailable:    false,
			ResponseMessage: "No deployments available",
		}, nil
	}

	// Calculate deployment resource requirements
	requiredCPU, requiredRAM, requiredDisk := calculateDeploymentResourceRequirements(deployment)

	// Check if deployment has cluster requirements and if node matches
	var rejectionReason string
	if len(deployment.SelectedClusters) > 0 {
		log.Printf("[Centro] Deployment %s has cluster requirements: %v", deployment.DeploymentId, deployment.SelectedClusters)
		clusterMatches := false
		for _, cluster := range deployment.SelectedClusters {
			if cluster == node.ClusterName {
				clusterMatches = true
				break
			}
		}
		// log.Printf("[Centro] Cluster matches: %v, node cluster: %s", clusterMatches, node.ClusterName)
		if !clusterMatches {
			rejectionReason = fmt.Sprintf("Cluster mismatch: deployment requires %v, node is in '%s'",
				deployment.SelectedClusters, node.ClusterName)
			log.Printf("[Centro] Deployment %s rejected by node %s: %s", deployment.DeploymentId, req.NodeId, rejectionReason)

			// Check if any other nodes could potentially take this deployment
			s.handleDeploymentRejection(ctx, deployment, req.NodeId, rejectionReason, requiredCPU, requiredRAM, requiredDisk)

			return &pb.GetDeploymentResponse{
				DeploymentAvailable:    false,
				ResponseMessage: fmt.Sprintf("No matching deployments for cluster: %s", node.ClusterName),
			}, nil
		}
	}

	// Check if node has sufficient resources for the deployment
	if !nodeHasSufficientResources(node, requiredCPU, requiredRAM, requiredDisk) {
		rejectionReason = fmt.Sprintf("Insufficient resources: deployment needs CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB; node has CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB",
			requiredCPU, requiredRAM, requiredDisk, node.CPUCores, node.RamMB, node.DiskMB)
		log.Printf("[Centro] Deployment %s rejected by node %s: %s", deployment.DeploymentId, req.NodeId, rejectionReason)

		// Check if any other nodes could potentially take this deployment
		s.handleDeploymentRejection(ctx, deployment, req.NodeId, rejectionReason, requiredCPU, requiredRAM, requiredDisk)

		return &pb.GetDeploymentResponse{
			DeploymentAvailable:    false,
			ResponseMessage: "Insufficient resources on node for available deployments",
		}, nil
	}

	log.Printf("[Centro] Assigning deployment %s to node %s (cluster: %s) - Deployment requires CPU: %.2f cores, RAM: %.2fMB, Disk: %.2fMB",
		deployment.DeploymentId, req.NodeId, node.ClusterName, requiredCPU, requiredRAM, requiredDisk)

	deploymentStatus := &etcdstorage.DeploymentStatus{
		Deployment: deployment,
		NodeID:     req.NodeId,
		Status:     "assigned",
		ClaimedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.storage.SaveDeploymentActive(ctx, deployment.DeploymentId, deploymentStatus); err != nil {
		log.Printf("[Centro] Failed to save deployment assignment: %v", err)
	}

	if err := s.storage.SaveDeploymentEvent(ctx, deployment.DeploymentId, fmt.Sprintf("[%s] Deployment assigned to node %s", time.Now().Format(time.RFC3339), req.NodeId)); err != nil {
		log.Printf("[Centro] Failed to save deployment event: %v", err)
	}

	return &pb.GetDeploymentResponse{
		DeploymentAvailable:    true,
		Deployment:             deployment,
		ResponseMessage: fmt.Sprintf("Deployment %s assigned", deployment.DeploymentId),
	}, nil
}

func (s *CentroServer) UpdateStatus(ctx context.Context, req *pb.UpdateStatusRequest) (*pb.UpdateStatusResponse, error) {
	if req.NodeId == "" {
		return &pb.UpdateStatusResponse{
			Acknowledged:    false,
			ResponseMessage: "node_id is required",
		}, nil
	}

	if req.DeploymentId == "" {
		return &pb.UpdateStatusResponse{
			Acknowledged:    false,
			ResponseMessage: "deployment_id is required",
		}, nil
	}

	deploymentStatus, err := s.storage.GetDeploymentActive(ctx, req.DeploymentId)
	if err != nil {
		log.Printf("[Centro] Failed to get active deployment: %v", err)
		return &pb.UpdateStatusResponse{
			Acknowledged:    false,
			ResponseMessage: "Failed to get deployment status",
		}, nil
	}

	if deploymentStatus == nil {
		deploymentStatus = &etcdstorage.DeploymentStatus{
			NodeID:    req.NodeId,
			ClaimedAt: time.Now(),
		}
	}

	deploymentStatus.Status = req.DeploymentStatus
	deploymentStatus.Detail = req.StatusMessage
	deploymentStatus.UpdatedAt = time.Now()

	if err := s.storage.SaveDeploymentEvent(ctx, req.DeploymentId, fmt.Sprintf("[%s] Status: %s - %s", time.Now().Format(time.RFC3339), req.DeploymentStatus, req.StatusMessage)); err != nil {
		log.Printf("[Centro] Failed to save deployment event: %v", err)
	}

	log.Printf("[Centro] Deployment %s status update from node %s: %s - %s",
		req.DeploymentId, req.NodeId, req.DeploymentStatus, req.StatusMessage)

	if req.DeploymentStatus == "completed" || req.DeploymentStatus == "failed" {
		if err := s.storage.SaveDeploymentHistory(ctx, req.DeploymentId, deploymentStatus); err != nil {
			log.Printf("[Centro] Failed to save deployment history: %v", err)
			return &pb.UpdateStatusResponse{
				Acknowledged:    false,
				ResponseMessage: "Failed to save deployment history",
			}, nil
		}

		if err := s.storage.DeleteDeploymentActive(ctx, req.DeploymentId); err != nil {
			log.Printf("[Centro] Failed to delete active deployment: %v", err)
		}

		log.Printf("[Centro] Deployment %s finished with status: %s", req.DeploymentId, req.DeploymentStatus)
	} else {
		if err := s.storage.SaveDeploymentActive(ctx, req.DeploymentId, deploymentStatus); err != nil {
			log.Printf("[Centro] Failed to save deployment status: %v", err)
			return &pb.UpdateStatusResponse{
				Acknowledged:    false,
				ResponseMessage: "Failed to save deployment status",
			}, nil
		}
	}

	return &pb.UpdateStatusResponse{
		Acknowledged:    true,
		ResponseMessage: "Status updated successfully",
	}, nil
}

func (s *CentroServer) SetInstanceData(ctx context.Context, req *pb.SetInstanceDataRequest) (*pb.SetInstanceDataResponse, error) {
	if req.NodeId == "" {
		return &pb.SetInstanceDataResponse{
			Acknowledged:    false,
			ResponseMessage: "node_id is required",
		}, nil
	}

	if req.DeploymentId == "" {
		return &pb.SetInstanceDataResponse{
			Acknowledged:    false,
			ResponseMessage: "deployment_id is required",
		}, nil
	}

	if req.InstanceData == nil {
		return &pb.SetInstanceDataResponse{
			Acknowledged:    false,
			ResponseMessage: "instance_data is required",
		}, nil
	}

	// Log instance data for monitoring
	log.Printf("[Centro] Received instance data for deployment %s from node %s: instance=%s, status=%s, pid=%d",
		req.DeploymentId, req.NodeId, req.InstanceData.InstanceId, req.InstanceData.Status, req.InstanceData.Pid)

	// Save instance data using the dedicated instance_data key
	if err := s.storage.SaveInstanceData(ctx, req.DeploymentId, req.InstanceData); err != nil {
		log.Printf("[Centro] Failed to save instance data: %v", err)
		return &pb.SetInstanceDataResponse{
			Acknowledged:    false,
			ResponseMessage: fmt.Sprintf("Failed to save instance data: %v", err),
		}, nil
	}

	return &pb.SetInstanceDataResponse{
		Acknowledged:    true,
		ResponseMessage: "Instance data received successfully",
	}, nil
}

func (s *CentroServer) AddDeployment(deployment *pb.Deployment) {
	ctx := context.Background()
	if err := s.storage.EnqueueDeployment(ctx, deployment); err != nil {
		log.Printf("[Centro] Failed to enqueue deployment: %v", err)
		return
	}

	queueLength, err := s.storage.GetQueueLength(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get queue length: %v", err)
		queueLength = 0
	}

	log.Printf("[Centro] Added deployment %s to queue (total queued: %d)", deployment.DeploymentId, queueLength)
}

func (s *CentroServer) GetNodeCount() int {
	ctx := context.Background()
	nodes, err := s.storage.GetAllNodes(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get node count: %v", err)
		return 0
	}
	return len(nodes)
}

func (s *CentroServer) GetDeploymentStats() (queued, active, completed int) {
	ctx := context.Background()

	queued, err := s.storage.GetQueueLength(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get queue length: %v", err)
		queued = 0
	}

	active, err = s.storage.GetActiveDeploymentCount(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get active deployment count: %v", err)
		active = 0
	}

	completed, err = s.storage.GetDeploymentHistoryCount(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get history count: %v", err)
		completed = 0
	}

	return
}
