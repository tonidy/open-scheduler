package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	etcdstorage "github.com/open-scheduler/centro/storage/etcd"
	pb "github.com/open-scheduler/proto"
)

type CentroServer struct {
	pb.UnimplementedNodeAgentServiceServer
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

func (s *CentroServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if req.NodeId == "" {
		return &pb.HeartbeatResponse{
			Ok:      false,
			Message: "node_id is required",
		}, nil
	}

	node, err := s.storage.GetNode(ctx, req.NodeId)
	if err != nil {
		log.Printf("[Centro] Failed to get node: %v", err)
		return &pb.HeartbeatResponse{
			Ok:      false,
			Message: "Failed to get node info",
		}, nil
	}

	if node == nil {
		log.Printf("[Centro] New node registered: %s", req.NodeId)
		node = &etcdstorage.NodeInfo{NodeID: req.NodeId}
	}

	node.LastHeartbeat = time.Now()
	node.RamMB = req.RamMb
	node.CPUPercent = req.CpuPercent
	node.DiskMB = req.DiskMb
	node.Metadata = req.Metadata

	if err := s.storage.SaveNode(ctx, node); err != nil {
		log.Printf("[Centro] Failed to save node: %v", err)
		return &pb.HeartbeatResponse{
			Ok:      false,
			Message: "Failed to save node info",
		}, nil
	}

	log.Printf("[Centro] Heartbeat from node %s - CPU: %.2f%%, RAM: %.2fMB, Disk: %.2fMB",
		req.NodeId, req.CpuPercent, req.RamMb, req.DiskMb)

	return &pb.HeartbeatResponse{
		Ok:      true,
		Message: "Heartbeat received",
	}, nil
}

func (s *CentroServer) GetJob(ctx context.Context, req *pb.GetJobRequest) (*pb.GetJobResponse, error) {
	if req.NodeId == "" {
		return &pb.GetJobResponse{
			HasJob:  false,
			Message: "node_id is required",
		}, nil
	}

	node, err := s.storage.GetNode(ctx, req.NodeId)
	if err != nil {
		log.Printf("[Centro] Failed to get node: %v", err)
		return &pb.GetJobResponse{
			HasJob:  false,
			Message: "Failed to get node info",
		}, nil
	}

	if node == nil {
		return &pb.GetJobResponse{
			HasJob:  false,
			Message: "Node not registered. Send a heartbeat first.",
		}, nil
	}

	job, err := s.storage.DequeueJob(ctx)
	if err != nil {
		log.Printf("[Centro] Failed to dequeue job: %v", err)
		return &pb.GetJobResponse{
			HasJob:  false,
			Message: "Failed to get job from queue",
		}, nil
	}

	if job == nil {
		return &pb.GetJobResponse{
			HasJob:  false,
			Message: "No jobs available",
		}, nil
	}

	log.Printf("[Centro] Assigning job %s to node %s", job.JobId, req.NodeId)

	return &pb.GetJobResponse{
		HasJob:  true,
		Job:     job,
		Message: fmt.Sprintf("Job %s assigned", job.JobId),
	}, nil
}

func (s *CentroServer) ClaimJob(ctx context.Context, req *pb.ClaimJobRequest) (*pb.ClaimJobResponse, error) {
	if req.NodeId == "" {
		return &pb.ClaimJobResponse{
			Ok:      false,
			Message: "node_id is required",
		}, nil
	}

	if req.JobId == "" {
		return &pb.ClaimJobResponse{
			Ok:      false,
			Message: "job_id is required",
		}, nil
	}

	existing, err := s.storage.GetJobActive(ctx, req.JobId)
	if err != nil {
		log.Printf("[Centro] Failed to get active job: %v", err)
		return &pb.ClaimJobResponse{
			Ok:      false,
			Message: "Failed to check job status",
		}, nil
	}

	if existing != nil {
		if existing.NodeID == req.NodeId {
			return &pb.ClaimJobResponse{
				Ok:      true,
				Message: "Job already claimed by this node",
			}, nil
		}
		return &pb.ClaimJobResponse{
			Ok:      false,
			Message: fmt.Sprintf("Job already claimed by node %s", existing.NodeID),
		}, nil
	}

	jobStatus := &etcdstorage.JobStatus{
		NodeID:    req.NodeId,
		Status:    "claimed",
		ClaimedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.storage.SaveJobActive(ctx, req.JobId, jobStatus); err != nil {
		log.Printf("[Centro] Failed to save job claim: %v", err)
		return &pb.ClaimJobResponse{
			Ok:      false,
			Message: "Failed to claim job",
		}, nil
	}

	log.Printf("[Centro] Job %s claimed by node %s", req.JobId, req.NodeId)

	return &pb.ClaimJobResponse{
		Ok:      true,
		Message: "Job claimed successfully",
	}, nil
}

func (s *CentroServer) UpdateStatus(ctx context.Context, req *pb.UpdateStatusRequest) (*pb.UpdateStatusResponse, error) {
	if req.NodeId == "" {
		return &pb.UpdateStatusResponse{
			Ok:      false,
			Message: "node_id is required",
		}, nil
	}

	if req.JobId == "" {
		return &pb.UpdateStatusResponse{
			Ok:      false,
			Message: "job_id is required",
		}, nil
	}

	jobStatus, err := s.storage.GetJobActive(ctx, req.JobId)
	if err != nil {
		log.Printf("[Centro] Failed to get active job: %v", err)
		return &pb.UpdateStatusResponse{
			Ok:      false,
			Message: "Failed to get job status",
		}, nil
	}

	if jobStatus == nil {
		jobStatus = &etcdstorage.JobStatus{
			NodeID: req.NodeId,
		}
	}

	jobStatus.Status = req.Status
	jobStatus.Detail = req.Detail
	jobStatus.UpdatedAt = time.Now()

	log.Printf("[Centro] Job %s status update from node %s: %s - %s",
		req.JobId, req.NodeId, req.Status, req.Detail)

	if req.Status == "completed" || req.Status == "failed" {
		if err := s.storage.SaveJobHistory(ctx, req.JobId, jobStatus); err != nil {
			log.Printf("[Centro] Failed to save job history: %v", err)
			return &pb.UpdateStatusResponse{
				Ok:      false,
				Message: "Failed to save job history",
			}, nil
		}

		if err := s.storage.DeleteJobActive(ctx, req.JobId); err != nil {
			log.Printf("[Centro] Failed to delete active job: %v", err)
		}

		log.Printf("[Centro] Job %s finished with status: %s", req.JobId, req.Status)
	} else {
		if err := s.storage.SaveJobActive(ctx, req.JobId, jobStatus); err != nil {
			log.Printf("[Centro] Failed to save job status: %v", err)
			return &pb.UpdateStatusResponse{
				Ok:      false,
				Message: "Failed to save job status",
			}, nil
		}
	}

	return &pb.UpdateStatusResponse{
		Ok:      true,
		Message: "Status updated successfully",
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
