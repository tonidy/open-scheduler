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

// calculateJobResourceRequirements calculates the resource requirements for the job
// Returns: (cpuCores, ramMB, diskMB)
func calculateJobResourceRequirements(job *pb.Job) (float32, float32, float32) {
	var totalCPU float32
	var totalRAM float32
	var totalDisk float32 = 0 // Disk requirements are not typically specified, but we'll keep this for consistency

	if job.ResourceRequirements != nil {
		// Use cpu_limit_cores if set, otherwise use cpu_reserved_cores
		if job.ResourceRequirements.CpuLimitCores > 0 {
			totalCPU = float32(job.ResourceRequirements.CpuLimitCores)
		} else if job.ResourceRequirements.CpuReservedCores > 0 {
			totalCPU = float32(job.ResourceRequirements.CpuReservedCores)
		}

		// Use memory_limit_mb if set, otherwise use memory_reserved_mb
		if job.ResourceRequirements.MemoryLimitMb > 0 {
			totalRAM = float32(job.ResourceRequirements.MemoryLimitMb)
		} else if job.ResourceRequirements.MemoryReservedMb > 0 {
			totalRAM = float32(job.ResourceRequirements.MemoryReservedMb)
		}
	}

	return totalCPU, totalRAM, totalDisk
}

// nodeHasSufficientResources checks if a node has enough available resources for a job
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

// handleJobRejection handles when a job is rejected by a node, checks if any other nodes could take it,
// and saves detailed rejection events if no nodes are suitable
func (s *CentroServer) handleJobRejection(ctx context.Context, job *pb.Job, rejectedByNodeID, rejectionReason string, requiredCPU, requiredRAM, requiredDisk float32) {
	// Get all nodes to check if any could potentially take this job
	allNodes, err := s.storage.GetAllNodes(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get all nodes: %v", err)
		// Re-queue the job anyway
		if err := s.storage.EnqueueJob(ctx, job); err != nil {
			log.Printf("[Centro] Failed to re-queue job: %v", err)
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
		if len(job.SelectedClusters) > 0 {
			clusterMatches := false
			for _, cluster := range job.SelectedClusters {
				if cluster == node.ClusterName {
					clusterMatches = true
					break
				}
			}
			if !clusterMatches {
				rejectionReasons[nodeID] = fmt.Sprintf("Cluster mismatch: job requires %v, node is in '%s'",
					job.SelectedClusters, node.ClusterName)
				continue
			}
		}

		// Check resources
		if !nodeHasSufficientResources(node, requiredCPU, requiredRAM, requiredDisk) {
			rejectionReasons[nodeID] = fmt.Sprintf("Insufficient resources: job needs CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB; node has CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB",
				requiredCPU, requiredRAM, requiredDisk, node.CPUCores, node.RamMB, node.DiskMB)
			continue
		}

		// This node could potentially take the job
		hasHealthyMatchingNode = true
		break
	}

	// If no nodes can take this job, save a detailed event
	if !hasHealthyMatchingNode {
		var eventMessage strings.Builder
		eventMessage.WriteString(fmt.Sprintf("[%s] No matching nodes available for job %s\n",
			time.Now().Format(time.RFC3339), job.JobId))
		eventMessage.WriteString(fmt.Sprintf("Job requirements: CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB",
			requiredCPU, requiredRAM, requiredDisk))
		if len(job.SelectedClusters) > 0 {
			eventMessage.WriteString(fmt.Sprintf(", Clusters=%v", job.SelectedClusters))
		}
		eventMessage.WriteString("\n\nRejection reasons by node:\n")

		for nodeID, reason := range rejectionReasons {
			eventMessage.WriteString(fmt.Sprintf("  - Node '%s': %s\n", nodeID, reason))
		}

		if err := s.storage.SaveJobEvent(ctx, job.JobId, eventMessage.String()); err != nil {
			log.Printf("[Centro] Failed to save 'no matching nodes' event: %v", err)
		}

		log.Printf("[Centro] Job %s has no matching nodes. Rejection reasons saved to events.", job.JobId)
		if err := s.storage.EnqueueFailedJob(ctx, job); err != nil {
			log.Printf("[Centro] Failed to enqueue failed job: %v", err)
		}
	} else {

		if err := s.storage.EnqueueJob(ctx, job); err != nil {
			log.Printf("[Centro] Failed to re-queue job: %v", err)
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

func (s *CentroServer) GetJob(ctx context.Context, req *pb.GetJobRequest) (*pb.GetJobResponse, error) {
	if req.NodeId == "" {
		return &pb.GetJobResponse{
			JobAvailable:    false,
			ResponseMessage: "node_id is required",
		}, nil
	}

	node, err := s.storage.GetNode(ctx, req.NodeId)
	if err != nil {
		log.Printf("[Centro] Failed to get node: %v", err)
		return &pb.GetJobResponse{
			JobAvailable:    false,
			ResponseMessage: "Failed to get node info",
		}, nil
	}

	if node == nil {
		return &pb.GetJobResponse{
			JobAvailable:    false,
			ResponseMessage: "Node not registered. Send a heartbeat first.",
		}, nil
	}

	// Check if node is healthy based on last heartbeat
	if !node.IsHealthy() {
		log.Printf("[Centro] Node %s is not healthy (last heartbeat: %v) - rejecting job request",
			req.NodeId, node.LastHeartbeat)
		return &pb.GetJobResponse{
			JobAvailable:    false,
			ResponseMessage: fmt.Sprintf("Node is not healthy. Last heartbeat: %v", node.LastHeartbeat),
		}, nil
	}

	job, err := s.storage.DequeueJob(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to dequeue job: %v", err)
		return &pb.GetJobResponse{
			JobAvailable:    false,
			ResponseMessage: "Failed to get job from queue",
		}, nil
	}

	if job == nil {
		return &pb.GetJobResponse{
			JobAvailable:    false,
			ResponseMessage: "No jobs available",
		}, nil
	}

	// Calculate job resource requirements
	requiredCPU, requiredRAM, requiredDisk := calculateJobResourceRequirements(job)

	// Check if job has cluster requirements and if node matches
	var rejectionReason string
	if len(job.SelectedClusters) > 0 {
		log.Printf("[Centro] Job %s has cluster requirements: %v", job.JobId, job.SelectedClusters)
		clusterMatches := false
		for _, cluster := range job.SelectedClusters {
			if cluster == node.ClusterName {
				clusterMatches = true
				break
			}
		}
		// log.Printf("[Centro] Cluster matches: %v, node cluster: %s", clusterMatches, node.ClusterName)
		if !clusterMatches {
			rejectionReason = fmt.Sprintf("Cluster mismatch: job requires %v, node is in '%s'",
				job.SelectedClusters, node.ClusterName)
			log.Printf("[Centro] Job %s rejected by node %s: %s", job.JobId, req.NodeId, rejectionReason)

			// Check if any other nodes could potentially take this job
			s.handleJobRejection(ctx, job, req.NodeId, rejectionReason, requiredCPU, requiredRAM, requiredDisk)

			return &pb.GetJobResponse{
				JobAvailable:    false,
				ResponseMessage: fmt.Sprintf("No matching jobs for cluster: %s", node.ClusterName),
			}, nil
		}
	}

	// Check if node has sufficient resources for the job
	if !nodeHasSufficientResources(node, requiredCPU, requiredRAM, requiredDisk) {
		rejectionReason = fmt.Sprintf("Insufficient resources: job needs CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB; node has CPU=%.2f cores, RAM=%.2fMB, Disk=%.2fMB",
			requiredCPU, requiredRAM, requiredDisk, node.CPUCores, node.RamMB, node.DiskMB)
		log.Printf("[Centro] Job %s rejected by node %s: %s", job.JobId, req.NodeId, rejectionReason)

		// Check if any other nodes could potentially take this job
		s.handleJobRejection(ctx, job, req.NodeId, rejectionReason, requiredCPU, requiredRAM, requiredDisk)

		return &pb.GetJobResponse{
			JobAvailable:    false,
			ResponseMessage: "Insufficient resources on node for available jobs",
		}, nil
	}

	log.Printf("[Centro] Assigning job %s to node %s (cluster: %s) - Job requires CPU: %.2f cores, RAM: %.2fMB, Disk: %.2fMB",
		job.JobId, req.NodeId, node.ClusterName, requiredCPU, requiredRAM, requiredDisk)

	jobStatus := &etcdstorage.JobStatus{
		Job:       job,
		NodeID:    req.NodeId,
		Status:    "assigned",
		ClaimedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.storage.SaveJobActive(ctx, job.JobId, jobStatus); err != nil {
		log.Printf("[Centro] Failed to save job assignment: %v", err)
	}

	if err := s.storage.SaveJobEvent(ctx, job.JobId, fmt.Sprintf("[%s] Job assigned to node %s", time.Now().Format(time.RFC3339), req.NodeId)); err != nil {
		log.Printf("[Centro] Failed to save job event: %v", err)
	}

	return &pb.GetJobResponse{
		JobAvailable:    true,
		Job:             job,
		ResponseMessage: fmt.Sprintf("Job %s assigned", job.JobId),
	}, nil
}

func (s *CentroServer) UpdateStatus(ctx context.Context, req *pb.UpdateStatusRequest) (*pb.UpdateStatusResponse, error) {
	if req.NodeId == "" {
		return &pb.UpdateStatusResponse{
			Acknowledged:    false,
			ResponseMessage: "node_id is required",
		}, nil
	}

	if req.JobId == "" {
		return &pb.UpdateStatusResponse{
			Acknowledged:    false,
			ResponseMessage: "job_id is required",
		}, nil
	}

	jobStatus, err := s.storage.GetJobActive(ctx, req.JobId)
	if err != nil {
		log.Printf("[Centro] Failed to get active job: %v", err)
		return &pb.UpdateStatusResponse{
			Acknowledged:    false,
			ResponseMessage: "Failed to get job status",
		}, nil
	}

	if jobStatus == nil {
		jobStatus = &etcdstorage.JobStatus{
			NodeID:    req.NodeId,
			ClaimedAt: time.Now(),
		}
	}

	jobStatus.Status = req.JobStatus
	jobStatus.Detail = req.StatusMessage
	jobStatus.UpdatedAt = time.Now()

	if err := s.storage.SaveJobEvent(ctx, req.JobId, fmt.Sprintf("[%s] Status: %s - %s", time.Now().Format(time.RFC3339), req.JobStatus, req.StatusMessage)); err != nil {
		log.Printf("[Centro] Failed to save job event: %v", err)
	}

	log.Printf("[Centro] Job %s status update from node %s: %s - %s",
		req.JobId, req.NodeId, req.JobStatus, req.StatusMessage)

	if req.JobStatus == "completed" || req.JobStatus == "failed" {
		if err := s.storage.SaveJobHistory(ctx, req.JobId, jobStatus); err != nil {
			log.Printf("[Centro] Failed to save job history: %v", err)
			return &pb.UpdateStatusResponse{
				Acknowledged:    false,
				ResponseMessage: "Failed to save job history",
			}, nil
		}

		if err := s.storage.DeleteJobActive(ctx, req.JobId); err != nil {
			log.Printf("[Centro] Failed to delete active job: %v", err)
		}

		log.Printf("[Centro] Job %s finished with status: %s", req.JobId, req.JobStatus)
	} else {
		if err := s.storage.SaveJobActive(ctx, req.JobId, jobStatus); err != nil {
			log.Printf("[Centro] Failed to save job status: %v", err)
			return &pb.UpdateStatusResponse{
				Acknowledged:    false,
				ResponseMessage: "Failed to save job status",
			}, nil
		}
	}

	return &pb.UpdateStatusResponse{
		Acknowledged:    true,
		ResponseMessage: "Status updated successfully",
	}, nil
}

func (s *CentroServer) SetContainerData(ctx context.Context, req *pb.SetContainerDataRequest) (*pb.SetContainerDataResponse, error) {
	if req.NodeId == "" {
		return &pb.SetContainerDataResponse{
			Acknowledged:    false,
			ResponseMessage: "node_id is required",
		}, nil
	}

	if req.JobId == "" {
		return &pb.SetContainerDataResponse{
			Acknowledged:    false,
			ResponseMessage: "job_id is required",
		}, nil
	}

	if req.ContainerData == nil {
		return &pb.SetContainerDataResponse{
			Acknowledged:    false,
			ResponseMessage: "container_data is required",
		}, nil
	}

	// Log container data for monitoring
	log.Printf("[Centro] Received container data for job %s from node %s: container=%s, status=%s, pid=%d",
		req.JobId, req.NodeId, req.ContainerData.ContainerId, req.ContainerData.Status, req.ContainerData.Pid)

	// Save container data using the dedicated container_data key
	if err := s.storage.SaveContainerData(ctx, req.JobId, req.ContainerData); err != nil {
		log.Printf("[Centro] Failed to save container data: %v", err)
		return &pb.SetContainerDataResponse{
			Acknowledged:    false,
			ResponseMessage: fmt.Sprintf("Failed to save container data: %v", err),
		}, nil
	}

	return &pb.SetContainerDataResponse{
		Acknowledged:    true,
		ResponseMessage: "Container data received successfully",
	}, nil
}

func (s *CentroServer) AddJob(job *pb.Job) {
	ctx := context.Background()
	if err := s.storage.EnqueueJob(ctx, job); err != nil {
		log.Printf("[Centro] Failed to enqueue job: %v", err)
		return
	}

	queueLength, err := s.storage.GetQueueLength(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get queue length: %v", err)
		queueLength = 0
	}

	log.Printf("[Centro] Added job %s to queue (total queued: %d)", job.JobId, queueLength)
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

func (s *CentroServer) GetJobStats() (queued, active, completed int) {
	ctx := context.Background()

	queued, err := s.storage.GetQueueLength(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get queue length: %v", err)
		queued = 0
	}

	active, err = s.storage.GetActiveJobCount(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get active job count: %v", err)
		active = 0
	}

	completed, err = s.storage.GetJobHistoryCount(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to get history count: %v", err)
		completed = 0
	}

	return
}
