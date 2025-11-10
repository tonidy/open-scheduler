package rest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	etcdstorage "github.com/open-scheduler/centro/storage/etcd"
	pb "github.com/open-scheduler/proto"
	httpSwagger "github.com/swaggo/http-swagger"
)

type APIServer struct {
	storage *etcdstorage.Storage
	router  *mux.Router
}

func NewAPIServer(storage *etcdstorage.Storage) *APIServer {
	server := &APIServer{
		storage: storage,
		router:  mux.NewRouter(),
	}

	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	// Swagger documentation
	s.router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	api := s.router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/auth/login", s.handleLogin).Methods("POST", "OPTIONS")

	protected := api.PathPrefix("").Subrouter()
	protected.Use(JWTAuthMiddleware)

	protected.HandleFunc("/jobs", s.handleListJobs).Methods("GET")
	protected.HandleFunc("/jobs", s.handleSubmitJob).Methods("POST")
	protected.HandleFunc("/jobs/{id}", s.handleGetJob).Methods("GET")
	protected.HandleFunc("/jobs/{id}/status", s.handleGetJobStatus).Methods("GET")
	protected.HandleFunc("/jobs/{id}/events", s.handleGetJobEvents).Methods("GET")

	protected.HandleFunc("/nodes", s.handleListNodes).Methods("GET")
	protected.HandleFunc("/nodes/{id}", s.handleGetNode).Methods("GET")
	protected.HandleFunc("/nodes/{id}/health", s.handleNodeHealth).Methods("GET")

	protected.HandleFunc("/stats", s.handleStats).Methods("GET")

	s.router.Use(LoggingMiddleware)
	s.router.Use(CORSMiddleware)
}

func (s *APIServer) GetRouter() *mux.Router {
	return s.router
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
	Username  string `json:"username"`
}

// handleLogin godoc
// @Summary Login to get JWT token
// @Description Authenticate with username and password to receive a JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	if req.Username == "admin" && req.Password == "admin123" {
		token, err := GenerateToken(req.Username, "admin")
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		respondWithJSON(w, http.StatusOK, LoginResponse{
			Token:     token,
			ExpiresIn: 86400,
			Username:  req.Username,
		})
		return
	}

	respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
}

// handleListJobs godoc
// @Summary List all jobs
// @Description Get a list of all jobs with optional status filter
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (queued, active, completed)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /jobs [get]
func (s *APIServer) handleListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	statusFilter := r.URL.Query().Get("status")

	response := make(map[string]interface{})

	if statusFilter == "" || statusFilter == "queued" {
		queueLength, err := s.storage.GetQueueLength(ctx)
		if err != nil {
			log.Printf("[Centro REST] Failed to get queue length: %v", err)
		} else {
			response["queued_count"] = queueLength
		}
	}

	if statusFilter == "" || statusFilter == "active" {
		activeJobs, err := s.storage.GetAllActiveJobs(ctx)
		if err != nil {
			log.Printf("[Centro REST] Failed to get active jobs: %v", err)
		} else {
			jobs := make([]map[string]interface{}, 0, len(activeJobs))
			for jobID, status := range activeJobs {
				jobs = append(jobs, map[string]interface{}{
					"job_id":     jobID,
					"node_id":    status.NodeID,
					"status":     status.Status,
					"detail":     status.Detail,
					"updated_at": status.UpdatedAt,
					"claimed_at": status.ClaimedAt,
					"job":        status.Job,
				})
			}
			response["active_jobs"] = jobs
		}
	}

	if statusFilter == "" || statusFilter == "completed" {
		completedCount, err := s.storage.GetJobHistoryCount(ctx)
		if err != nil {
			log.Printf("[Centro REST] Failed to get history count: %v", err)
		} else {
			response["completed_count"] = completedCount
		}

		completedJobs, err := s.storage.GetAllJobHistory(ctx)
		if err != nil {
			log.Printf("[Centro REST] Failed to get job history: %v", err)
		} else {
			jobs := make([]map[string]interface{}, 0, len(completedJobs))
			for jobID, status := range completedJobs {
				jobs = append(jobs, map[string]interface{}{
					"job_id":     jobID,
					"node_id":    status.NodeID,
					"status":     status.Status,
					"detail":     status.Detail,
					"updated_at": status.UpdatedAt,
					"claimed_at": status.ClaimedAt,
					"job":        status.Job,
				})
			}
			response["completed_jobs"] = jobs
		}
	}

	respondWithJSON(w, http.StatusOK, response)
}

type SubmitJobRequest struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Datacenters string            `json:"datacenters"`
	Tasks       []TaskRequest     `json:"tasks"`
	Meta        map[string]string `json:"meta"`
}

type TaskRequest struct {
	Name      string               `json:"name"`
	Driver    string               `json:"driver"`
	Kind      string               `json:"kind,omitempty"`
	Config    ContainerSpecRequest `json:"config"`
	Env       map[string]string    `json:"env"`
	Resources *ResourcesRequest    `json:"resources,omitempty"`
	Volumes   []VolumeRequest      `json:"volumes,omitempty"`
}

type ResourcesRequest struct {
	MemoryMB        int64   `json:"memory_mb"`
	MemoryReserveMB int64   `json:"memory_reserve_mb,omitempty"`
	CPU             float32 `json:"cpu"`
	CPUReserve      float32 `json:"cpu_reserve,omitempty"`
}

type VolumeRequest struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only,omitempty"`
}

type ContainerSpecRequest struct {
	Image   string            `json:"image"`
	Command []string          `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

// handleSubmitJob godoc
// @Summary Submit a new job
// @Description Create and submit a new job to the scheduler
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param job body SubmitJobRequest true "Job details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /jobs [post]
func (s *APIServer) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	var req SubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Job name is required")
		return
	}

	if len(req.Tasks) == 0 {
		respondWithError(w, http.StatusBadRequest, "At least one task is required")
		return
	}

	jobID := uuid.New().String()

	tasks := make([]*pb.Task, 0, len(req.Tasks))
	for _, t := range req.Tasks {
		task := &pb.Task{
			TaskName:     t.Name,
			DriverType:   t.Driver,
			WorkloadType: t.Kind,
			ContainerConfig: &pb.ContainerSpec{
				ImageName:     t.Config.Image,
				Entrypoint:    t.Config.Command,
				Arguments:     t.Config.Args,
				DriverOptions: t.Config.Options,
			},
			EnvironmentVariables: t.Env,
		}

		// Add resources if specified
		if t.Resources != nil {
			task.ResourceRequirements = &pb.Resources{
				MemoryLimitMb:    t.Resources.MemoryMB,
				MemoryReservedMb: t.Resources.MemoryReserveMB,
				CpuLimitCores:    t.Resources.CPU,
				CpuReservedCores: t.Resources.CPUReserve,
			}
		}

		// Add volumes if specified
		if len(t.Volumes) > 0 {
			task.VolumeMounts = make([]*pb.Volume, 0, len(t.Volumes))
			for _, v := range t.Volumes {
				task.VolumeMounts = append(task.VolumeMounts, &pb.Volume{
					SourcePath: v.HostPath,
					TargetPath: v.ContainerPath,
					ReadOnly:   v.ReadOnly,
				})
			}
		}

		tasks = append(tasks, task)
	}

	// Split comma-separated clusters string into array
	var selectedClusters []string
	if req.Datacenters != "" {
		// Support comma-separated list
		for _, cluster := range strings.Split(req.Datacenters, ",") {
			cluster = strings.TrimSpace(cluster)
			if cluster != "" {
				selectedClusters = append(selectedClusters, cluster)
			}
		}
	}

	job := &pb.Job{
		JobId:            jobID,
		JobName:          req.Name,
		JobType:          req.Type,
		SelectedClusters: selectedClusters,
		Tasks:            tasks,
		JobMetadata:      req.Meta,
	}

	ctx := context.Background()
	if err := s.storage.EnqueueJob(ctx, job); err != nil {
		log.Printf("[Centro REST] Failed to enqueue job: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to submit job")
		return
	}

	log.Printf("[Centro REST] Job submitted: %s (%s)", jobID, req.Name)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"job_id":  jobID,
		"message": "Job submitted successfully",
		"job":     job,
	})
}

// handleGetJob godoc
// @Summary Get job details
// @Description Get detailed information about a specific job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /jobs/{id} [get]
func (s *APIServer) handleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	ctx := context.Background()

	activeJob, err := s.storage.GetJobActive(ctx, jobID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get active job: %v", err)
	}

	if activeJob != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"job_id":     jobID,
			"status":     "active",
			"node_id":    activeJob.NodeID,
			"detail":     activeJob.Detail,
			"updated_at": activeJob.UpdatedAt,
			"claimed_at": activeJob.ClaimedAt,
			"job":        activeJob.Job,
		})
		return
	}

	historyJob, err := s.storage.GetJobHistory(ctx, jobID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get job history: %v", err)
	}

	if historyJob != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"job_id":     jobID,
			"status":     "completed",
			"node_id":    historyJob.NodeID,
			"detail":     historyJob.Detail,
			"updated_at": historyJob.UpdatedAt,
			"claimed_at": historyJob.ClaimedAt,
			"job":        historyJob.Job,
		})
		return
	}

	respondWithError(w, http.StatusNotFound, "Job not found")
}

// handleGetJobStatus godoc
// @Summary Get job status
// @Description Get the current status of a specific job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /jobs/{id}/status [get]
func (s *APIServer) handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	ctx := context.Background()

	activeJob, err := s.storage.GetJobActive(ctx, jobID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get active job: %v", err)
	}

	if activeJob != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"job_id":     jobID,
			"status":     activeJob.Status,
			"node_id":    activeJob.NodeID,
			"detail":     activeJob.Detail,
			"updated_at": activeJob.UpdatedAt,
		})
		return
	}

	historyJob, err := s.storage.GetJobHistory(ctx, jobID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get job history: %v", err)
	}

	if historyJob != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"job_id":     jobID,
			"status":     historyJob.Status,
			"node_id":    historyJob.NodeID,
			"detail":     historyJob.Detail,
			"updated_at": historyJob.UpdatedAt,
		})
		return
	}

	respondWithError(w, http.StatusNotFound, "Job not found")
}

// handleGetJobEvents godoc
// @Summary Get job events
// @Description Retrieve events for a specific job
// @Tags Jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /jobs/{id}/events [get]
func (s *APIServer) handleGetJobEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	ctx := context.Background()
	events, err := s.storage.GetJobEvents(ctx, jobID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get job events: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve events")
		return
	}

	if events == nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"job_id": jobID,
			"events": []string{},
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"job_id": jobID,
		"events": events,
	})
}

// handleListNodes godoc
// @Summary List all nodes
// @Description Get a list of all registered nodes in the cluster
// @Tags Nodes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /nodes [get]
func (s *APIServer) handleListNodes(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	nodes, err := s.storage.GetAllNodes(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get nodes: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve nodes")
		return
	}

	nodesList := make([]map[string]interface{}, 0, len(nodes))
	for _, node := range nodes {
		nodesList = append(nodesList, map[string]interface{}{
			"node_id":        node.NodeID,
			"last_heartbeat": node.LastHeartbeat,
			"ram_mb":         node.RamMB,
			"cpu_cores":      node.CPUCores,
			"disk_mb":        node.DiskMB,
			"metadata":       node.Metadata,
		})
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodesList,
		"count": len(nodesList),
	})
}

// handleGetNode godoc
// @Summary Get node details
// @Description Get detailed information about a specific node
// @Tags Nodes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Node ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /nodes/{id} [get]
func (s *APIServer) handleGetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	ctx := context.Background()
	node, err := s.storage.GetNode(ctx, nodeID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get node: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve node")
		return
	}

	if node == nil {
		respondWithError(w, http.StatusNotFound, "Node not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"node_id":        node.NodeID,
		"last_heartbeat": node.LastHeartbeat,
		"ram_mb":         node.RamMB,
		"cpu_cores":      node.CPUCores,
		"disk_mb":        node.DiskMB,
		"metadata":       node.Metadata,
	})
}

// handleNodeHealth godoc
// @Summary Check node health
// @Description Check the health status of a specific node
// @Tags Nodes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Node ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /nodes/{id}/health [get]
func (s *APIServer) handleNodeHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	ctx := context.Background()
	node, err := s.storage.GetNode(ctx, nodeID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get node: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve node")
		return
	}

	if node == nil {
		respondWithError(w, http.StatusNotFound, "Node not found")
		return
	}

	healthy := node.IsHealthy()
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"node_id":        node.NodeID,
		"healthy":        healthy,
		"status":         status,
		"last_heartbeat": node.LastHeartbeat,
	})
}

// handleStats godoc
// @Summary Get system statistics
// @Description Get overall system statistics including node and job counts
// @Tags Statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /stats [get]
func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	nodes, err := s.storage.GetAllNodes(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get nodes: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve stats")
		return
	}

	queueLength, err := s.storage.GetQueueLength(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get queue length: %v", err)
		queueLength = 0
	}

	activeCount, err := s.storage.GetActiveJobCount(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get active job count: %v", err)
		activeCount = 0
	}

	completedCount, err := s.storage.GetJobHistoryCount(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get history count: %v", err)
		completedCount = 0
	}

	healthyNodes := 0
	for _, node := range nodes {
		if node.IsHealthy() {
			healthyNodes++
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": map[string]interface{}{
			"total":   len(nodes),
			"healthy": healthyNodes,
		},
		"jobs": map[string]interface{}{
			"queued":    queueLength,
			"active":    activeCount,
			"completed": completedCount,
		},
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}
