package rest

import (
	"context"
	"encoding/json"
	"fmt"
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

	protected.HandleFunc("/deployments", s.handleListDeployments).Methods("GET")
	protected.HandleFunc("/deployments", s.handleSubmitDeployment).Methods("POST")
	protected.HandleFunc("/deployments/{id}", s.handleGetDeployment).Methods("GET")
	protected.HandleFunc("/deployments/{id}/status", s.handleGetDeploymentStatus).Methods("GET")
	protected.HandleFunc("/deployments/{id}/events", s.handleGetDeploymentEvents).Methods("GET")
	protected.HandleFunc("/instances", s.handleListInstances).Methods("GET")
	protected.HandleFunc("/instances/{id}", s.handleGetInstanceData).Methods("GET")

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
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"admin123"`
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

// handleListDeployments godoc
// @Summary List all deployments
// @Description Get a list of all deployments with optional status filter
// @Tags Deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (queued, pending, completed, failed)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /deployments [get]
func (s *APIServer) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	statusFilter := r.URL.Query().Get("status")

	response := make(map[string]interface{})

	// Queued deployments - waiting in queue to be picked up
	if statusFilter == "" || statusFilter == "queued" {
		queueLength, err := s.storage.GetQueueLength(ctx)
		if err != nil {
			log.Printf("[Centro REST] Failed to get queue length: %v", err)
		} else {
			response["queued_count"] = queueLength
		}
	}

	if statusFilter == "" || statusFilter == "queued" {
		queueDeployments, err := s.storage.GetQueueDeployments(ctx)
		if err != nil {
			log.Printf("[Centro REST] Failed to get queue deployments: %v", err)
		} else {
			// Format queued deployments to match expected structure (same format as active deployments)
			formattedQueue := make([]map[string]interface{}, 0, len(queueDeployments))
			for _, deployment := range queueDeployments {
				formattedQueue = append(formattedQueue, map[string]interface{}{
					"deployment_id": deployment.DeploymentId,
					"node_id":       "",
					"status":        "queued",
					"detail":        fmt.Sprintf("Deployment queued (attempt %d/%d)", deployment.RetryCount, deployment.MaxRetries),
					"updated_at":    nil,
					"claimed_at":    nil,
					"deployment":    deployment,
				})
			}
			response["queued_deployments"] = formattedQueue
		}
	}

	// Pending deployments - claimed by agent but not completed yet (assigned/running)
	if statusFilter == "" || statusFilter == "active" {
		activeDeployments, err := s.storage.GetAllActiveDeployments(ctx)
		if err != nil {
			log.Printf("[Centro REST] Failed to get active deployments: %v", err)
		} else {
			deployments := make([]map[string]interface{}, 0, len(activeDeployments))
			for deploymentID, status := range activeDeployments {
				deployments = append(deployments, map[string]interface{}{
					"deployment_id":     deploymentID,
					"node_id":    status.NodeID,
					"status":     status.Status,
					"detail":     status.Detail,
					"updated_at": status.UpdatedAt,
					"claimed_at": status.ClaimedAt,
					"deployment":        status.Deployment,
				})
			}
			response["active_deployments"] = deployments
		}
	}

	// Get all history to filter by status
	allHistory, err := s.storage.GetAllDeploymentHistory(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get deployment history: %v", err)
	} else {
		// Completed deployments - successfully finished
		if statusFilter == "" || statusFilter == "completed" {
			completedDeployments := make([]map[string]interface{}, 0)
			for deploymentID, status := range allHistory {
				if status.Status == "completed" {
					completedDeployments = append(completedDeployments, map[string]interface{}{
						"deployment_id":     deploymentID,
						"node_id":    status.NodeID,
						"status":     status.Status,
						"detail":     status.Detail,
						"updated_at": status.UpdatedAt,
						"claimed_at": status.ClaimedAt,
						"deployment":        status.Deployment,
					})
				}
			}
			response["completed_deployments"] = completedDeployments
			response["completed_count"] = len(completedDeployments)
		}

		// Failed deployments - permanently failed (exceeded retries)
		if statusFilter == "" || statusFilter == "failed" {
			failedDeployments := make([]map[string]interface{}, 0)
			for deploymentID, status := range allHistory {
				if status.Status == "failed" {
					failedDeployments = append(failedDeployments, map[string]interface{}{
						"deployment_id":     deploymentID,
						"node_id":    status.NodeID,
						"status":     "failed_permanent",
						"detail":     status.Detail,
						"updated_at": status.UpdatedAt,
						"claimed_at": status.ClaimedAt,
						"deployment":        status.Deployment,
					})
				}
			}

			// Also include deployments in failed queue (pending retry)
			failedQueueDeployments, err := s.storage.GetAllFailedDeployments(ctx)
			if err != nil {
				log.Printf("[Centro REST] Failed to get failed queue deployments: %v", err)
			} else {
				for _, deployment := range failedQueueDeployments {
					failedDeployments = append(failedDeployments, map[string]interface{}{
						"deployment_id":     deployment.DeploymentId,
						"node_id":    "",
						"status":     "failed_retrying",
						"detail":     fmt.Sprintf("Deployment failed, pending retry (attempt %d/%d)", deployment.RetryCount, deployment.MaxRetries),
						"updated_at": nil,
						"claimed_at": nil,
						"deployment":        deployment,
					})
				}
			}

			response["failed_deployments"] = failedDeployments
			response["failed_count"] = len(failedDeployments)
		}
	}

	respondWithJSON(w, http.StatusOK, response)
}

type SubmitDeploymentRequest struct {
	DeploymentId            string                `json:"deployment_id" example:"123"`
	DeploymentName          string                `json:"deployment_name" example:"web-server-deployment"`
	DeploymentType          string                `json:"deployment_type" example:"service"`
	SelectedClusters []string              `json:"selected_clusters" example:"dc1,dc2"`
	Meta             map[string]string     `json:"meta"`
	Driver           string                `json:"driver" example:"podman"`
	WorkloadType     string                `json:"workload_type" example:"container"`
	Command          string                `json:"command" example:"echo 'Hello World'"`
	CommandArray     []string              `json:"command_array,omitempty" example:"nginx,-g,daemon off;"`
	InstanceConfig   *InstanceSpecRequest  `json:"instance_config,omitempty"`
	Resources        *ResourcesRequest     `json:"resources,omitempty"`
	Volumes          []VolumeRequest       `json:"volumes,omitempty"`
	Replicas         *int32                `json:"replicas,omitempty" example:"2"`
	Placement        *PlacementRequest     `json:"placement,omitempty"`
	WorkingDir       string                `json:"working_dir,omitempty" example:"/usr/share/nginx/html"`
	Ports            []PortMappingRequest  `json:"ports,omitempty"`
	Security         *SecurityRequest      `json:"security,omitempty"`
	HealthCheck      *HealthCheckRequest   `json:"health_check,omitempty"`
	RestartPolicy    *RestartPolicyRequest `json:"restart_policy,omitempty"`
	Networks         []string              `json:"networks,omitempty" example:"backend-net"`
	InstanceType     string                `json:"instance_type,omitempty" example:"virtual-machine"`
}

type ResourcesRequest struct {
	MemoryMB        int64   `json:"memory_mb" example:"512"`
	MemoryReserveMB int64   `json:"memory_reserve_mb,omitempty" example:"256"`
	CPU             float32 `json:"cpu" example:"1.0"`
	CPUReserve      float32 `json:"cpu_reserve,omitempty" example:"0.5"`
}

type VolumeRequest struct {
	HostPath     string `json:"host_path" example:"/data/app"`
	InstancePath string `json:"instance_path" example:"/usr/share/nginx/html"`
	ReadOnly     bool   `json:"read_only,omitempty" example:"false"`
	Type         string `json:"type,omitempty" example:"bind"`
}

type InstanceSpecRequest struct {
	Image       string              `json:"image" example:"docker.io/library/alpine:latest"`
	Command     []string            `json:"command,omitempty" example:""`
	Args        []string            `json:"args,omitempty" example:""`
	Options     map[string]string   `json:"options,omitempty"`
	ImageSource *ImageSourceRequest `json:"image_source,omitempty"`
	UserData    string              `json:"user_data,omitempty"`
	Devices     []DeviceRequest     `json:"devices,omitempty"`
}

type PlacementRequest struct {
	Constraints []string `json:"constraints,omitempty" example:"node.driver in [podman, containerd]"`
	Strategy    string   `json:"strategy,omitempty" example:"spread"`
}

type PortMappingRequest struct {
	HostPort      int32  `json:"host_port" example:"8080"`
	ContainerPort int32  `json:"container_port" example:"80"`
	Protocol      string `json:"protocol" example:"tcp"`
}

type SecurityRequest struct {
	Privileged             bool     `json:"privileged,omitempty" example:"false"`
	CapabilitiesAdd        []string `json:"capabilities_add,omitempty" example:"NET_BIND_SERVICE"`
	CapabilitiesDrop       []string `json:"capabilities_drop,omitempty" example:"ALL"`
	ReadOnlyRootFilesystem bool     `json:"read_only_root_filesystem,omitempty" example:"true"`
}

type HealthCheckRequest struct {
	Test        []string `json:"test,omitempty" example:"CMD-SHELL,curl -f http://localhost/ || exit 1"`
	Interval    string   `json:"interval,omitempty" example:"30s"`
	Timeout     string   `json:"timeout,omitempty" example:"5s"`
	Retries     int32    `json:"retries,omitempty" example:"3"`
	StartPeriod string   `json:"start_period,omitempty" example:"5s"`
}

type RestartPolicyRequest struct {
	Condition   string `json:"condition" example:"on-failure"`
	MaxAttempts int32  `json:"max_attempts,omitempty" example:"3"`
}

type ImageSourceRequest struct {
	Alias  string `json:"alias,omitempty" example:"ubuntu/22.04"`
	Server string `json:"server,omitempty" example:"images.linuxcontainers.org"`
	Mode   string `json:"mode,omitempty" example:"pull"`
}

type DeviceRequest struct {
	Name       string            `json:"name" example:"eth0"`
	Type       string            `json:"type" example:"nic"`
	Properties map[string]string `json:"properties,omitempty"`
}

// handleSubmitDeployment godoc
// @Summary Submit a new deployment
// @Description Create and submit a new deployment to the scheduler
// @Tags Deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param deployment body SubmitDeploymentRequest true "Deployment details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /deployments [post]
func (s *APIServer) handleSubmitDeployment(w http.ResponseWriter, r *http.Request) {
	var req SubmitDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DeploymentName == "" {
		respondWithError(w, http.StatusBadRequest, "Deployment name is required")
		return
	}
	deploymentID := uuid.New().String()

	deployment := &pb.Deployment{
		DeploymentId:            deploymentID,
		DeploymentName:          req.DeploymentName,
		DeploymentType:          req.DeploymentType,
		SelectedClusters: req.SelectedClusters,
		DriverType:       req.Driver,
		WorkloadType:     req.WorkloadType,
		Command:          req.Command,
		RetryCount:       0,
		MaxRetries:       3, // Default: retry up to 3 times
		LastRetryTime:    0,
	}

	// Support command_array if provided
	if len(req.CommandArray) > 0 {
		deployment.CommandArray = req.CommandArray
	}

	// Replicas
	if req.Replicas != nil {
		deployment.Replicas = *req.Replicas
	} else {
		deployment.Replicas = 1 // Default to 1 replica
	}

	// Placement
	if req.Placement != nil {
		deployment.Placement = &pb.Placement{
			Constraints: req.Placement.Constraints,
			Strategy:    req.Placement.Strategy,
		}
	}

	// Working directory
	if req.WorkingDir != "" {
		deployment.WorkingDir = req.WorkingDir
	}

	// Ports
	if len(req.Ports) > 0 {
		deployment.Ports = make([]*pb.PortMapping, 0, len(req.Ports))
		for _, p := range req.Ports {
			deployment.Ports = append(deployment.Ports, &pb.PortMapping{
				HostPort:      p.HostPort,
				ContainerPort: p.ContainerPort,
				Protocol:      p.Protocol,
			})
		}
	}

	// Security settings
	if req.Security != nil {
		deployment.Security = &pb.SecuritySettings{
			Privileged:             req.Security.Privileged,
			CapabilitiesAdd:        req.Security.CapabilitiesAdd,
			CapabilitiesDrop:       req.Security.CapabilitiesDrop,
			ReadOnlyRootFilesystem: req.Security.ReadOnlyRootFilesystem,
		}
	}

	// Health check
	if req.HealthCheck != nil {
		deployment.HealthCheck = &pb.HealthCheck{
			Test:        req.HealthCheck.Test,
			Interval:    req.HealthCheck.Interval,
			Timeout:     req.HealthCheck.Timeout,
			Retries:     req.HealthCheck.Retries,
			StartPeriod: req.HealthCheck.StartPeriod,
		}
	}

	// Restart policy
	if req.RestartPolicy != nil {
		deployment.RestartPolicy = &pb.RestartPolicy{
			Condition:   req.RestartPolicy.Condition,
			MaxAttempts: req.RestartPolicy.MaxAttempts,
		}
	}

	// Networks
	if len(req.Networks) > 0 {
		deployment.Networks = make([]*pb.NetworkReference, 0, len(req.Networks))
		for _, netName := range req.Networks {
			deployment.Networks = append(deployment.Networks, &pb.NetworkReference{
				Name: netName,
			})
		}
	}

	// Instance type
	if req.InstanceType != "" {
		deployment.InstanceType = req.InstanceType
	}

	if req.InstanceConfig != nil {
		instSpec := &pb.InstanceSpec{
			ImageName:     req.InstanceConfig.Image,
			Entrypoint:    req.InstanceConfig.Command,
			Arguments:     req.InstanceConfig.Args,
			DriverOptions: req.InstanceConfig.Options,
		}

		// Image source
		if req.InstanceConfig.ImageSource != nil {
			instSpec.ImageSource = &pb.ImageSource{
				Alias:  req.InstanceConfig.ImageSource.Alias,
				Server: req.InstanceConfig.ImageSource.Server,
				Mode:   req.InstanceConfig.ImageSource.Mode,
			}
		}

		// User data
		if req.InstanceConfig.UserData != "" {
			instSpec.UserData = req.InstanceConfig.UserData
		}

		// Devices
		if len(req.InstanceConfig.Devices) > 0 {
			instSpec.Devices = make([]*pb.Device, 0, len(req.InstanceConfig.Devices))
			for _, d := range req.InstanceConfig.Devices {
				instSpec.Devices = append(instSpec.Devices, &pb.Device{
					Name:       d.Name,
					Type:       d.Type,
					Properties: d.Properties,
				})
			}
		}

		deployment.InstanceConfig = instSpec
	}
	if req.Resources != nil {
		deployment.ResourceRequirements = &pb.Resources{
			MemoryLimitMb:    req.Resources.MemoryMB,
			MemoryReservedMb: req.Resources.MemoryReserveMB,
			CpuLimitCores:    req.Resources.CPU,
			CpuReservedCores: req.Resources.CPUReserve,
		}
	}
	if len(req.Volumes) > 0 {
		deployment.VolumeMounts = make([]*pb.Volume, 0, len(req.Volumes))
		for _, v := range req.Volumes {
			vol := &pb.Volume{
				SourcePath: v.HostPath,
				TargetPath: v.InstancePath,
				ReadOnly:   v.ReadOnly,
			}
			if v.Type != "" {
				vol.Type = v.Type
			}
			deployment.VolumeMounts = append(deployment.VolumeMounts, vol)
		}
	}
	if req.Meta != nil {
		deployment.DeploymentMetadata = req.Meta
	}

	ctx := context.Background()
	if err := s.storage.EnqueueDeployment(ctx, deployment); err != nil {
		log.Printf("[Centro REST] Failed to enqueue deployment: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to submit deployment")
		return
	}

	log.Printf("[Centro REST] Deployment submitted: %s (%s)", deploymentID, req.DeploymentName)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"deployment_id":  deploymentID,
		"message": "Deployment submitted successfully",
		"deployment":     deployment,
	})
}

// handleGetDeployment godoc
// @Summary Get deployment details
// @Description Get detailed information about a specific deployment
// @Tags Deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deployment ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /deployments/{id} [get]
func (s *APIServer) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	ctx := context.Background()

	activeDeployment, err := s.storage.GetDeploymentActive(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get active deployment: %v", err)
	}

	events, err := s.storage.GetDeploymentEvents(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get deployment events: %v", err)
	}

	if activeDeployment != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deployment_id":     deploymentID,
			"status":     "active",
			"node_id":    activeDeployment.NodeID,
			"detail":     activeDeployment.Detail,
			"updated_at": activeDeployment.UpdatedAt,
			"claimed_at": activeDeployment.ClaimedAt,
			"deployment":        activeDeployment.Deployment,
			"events":     events,
		})
		return
	}

	queueDeployment, err := s.storage.GetQueueDeployment(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get queue deployment: %v", err)
	}

	if queueDeployment != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deployment_id":     deploymentID,
			"status":     "queued",
			"node_id":    "",
			"detail":     fmt.Sprintf("Deployment queued (attempt %d/%d)", queueDeployment.RetryCount, queueDeployment.MaxRetries),
			"updated_at": nil,
			"claimed_at": nil,
			"deployment":        queueDeployment,
			"events":     events,
		})
		return
	}

	failedDeployment, err := s.storage.GetFailedDeployment(ctx, deploymentID)
	log.Printf("[Centro REST] Failed deployment: %v", failedDeployment)
	if err != nil {
		log.Printf("[Centro REST] Failed to get failed deployment: %v", err)
	}

	if failedDeployment != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deployment_id":     deploymentID,
			"status":     "failed",
			"node_id":    "",
			"detail":     fmt.Sprintf("Deployment failed, pending retry (attempt %d/%d)", failedDeployment.RetryCount, failedDeployment.MaxRetries),
			"updated_at": nil,
			"claimed_at": nil,
			"deployment":        failedDeployment,
			"events":     events,
		})
		return
	}

	historyDeployment, err := s.storage.GetDeploymentHistory(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get deployment history: %v", err)
	}

	if historyDeployment != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deployment_id":     deploymentID,
			"status":     "completed",
			"node_id":    historyDeployment.NodeID,
			"detail":     historyDeployment.Detail,
			"updated_at": historyDeployment.UpdatedAt,
			"claimed_at": historyDeployment.ClaimedAt,
			"deployment":        historyDeployment.Deployment,
			"events":     events,
		})
		return
	}

	respondWithError(w, http.StatusNotFound, "Deployment not found")
}

// handleGetDeploymentStatus godoc
// @Summary Get deployment status
// @Description Get the current status of a specific deployment
// @Tags Deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deployment ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /deployments/{id}/status [get]
func (s *APIServer) handleGetDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	ctx := context.Background()

	activeDeployment, err := s.storage.GetDeploymentActive(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get active deployment: %v", err)
	}

	if activeDeployment != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deployment_id":     deploymentID,
			"status":     activeDeployment.Status,
			"node_id":    activeDeployment.NodeID,
			"detail":     activeDeployment.Detail,
			"updated_at": activeDeployment.UpdatedAt,
		})
		return
	}

	historyDeployment, err := s.storage.GetDeploymentHistory(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get deployment history: %v", err)
	}

	if historyDeployment != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deployment_id":     deploymentID,
			"status":     historyDeployment.Status,
			"node_id":    historyDeployment.NodeID,
			"detail":     historyDeployment.Detail,
			"updated_at": historyDeployment.UpdatedAt,
		})
		return
	}

	respondWithError(w, http.StatusNotFound, "Deployment not found")
}

// handleListInstances godoc
// @Summary List all instances
// @Description Get a list of all instances in the cluster
// @Tags Instances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /instances [get]
func (s *APIServer) handleListInstances(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	instances, err := s.storage.GetListOfInstances(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get instances: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve instances")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"instances": instances,
		"count":     len(instances),
	})
}

// handleGetDeploymentEvents godoc
// @Summary Get deployment events
// @Description Retrieve events for a specific deployment
// @Tags Deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Deployment ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /deployments/{id}/events [get]
func (s *APIServer) handleGetDeploymentEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	ctx := context.Background()
	events, err := s.storage.GetDeploymentEvents(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get deployment events: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve events")
		return
	}

	if events == nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"deployment_id": deploymentID,
			"events": []string{},
		})
		return
	}

	// Filter out consecutive duplicate events
	filteredEvents := filterConsecutiveDuplicates(events)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id": deploymentID,
		"events": filteredEvents,
	})
}

// filterConsecutiveDuplicates removes consecutive duplicate events from the list
func filterConsecutiveDuplicates(events []string) []string {
	if len(events) == 0 {
		return events
	}

	filtered := make([]string, 0, len(events))
	filtered = append(filtered, events[0])

	for i := 1; i < len(events); i++ {
		// Extract the message part after the timestamp (everything after "] ")
		currentMsg := extractEventMessage(events[i])
		previousMsg := extractEventMessage(events[i-1])

		// Only add if the message is different from the previous one
		if currentMsg != previousMsg {
			filtered = append(filtered, events[i])
		}
	}

	return filtered
}

// extractEventMessage extracts the message part from an event string
// Events format: "[timestamp] message"
func extractEventMessage(event string) string {
	// Find the position after the timestamp bracket
	idx := strings.Index(event, "] ")
	if idx == -1 {
		return event
	}
	return event[idx+2:]
}

// handleGetInstanceData godoc
// @Summary Get instance data for a deployment
// @Description Retrieves the instance data associated with a specific deployment
// @Tags Instances
// @Security BearerAuth
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /instances/{id} [get]
func (s *APIServer) handleGetInstanceData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	ctx := context.Background()
	instanceData, err := s.storage.GetInstanceData(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get instance data: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve instance data")
		return
	}

	if instanceData == nil {
		respondWithError(w, http.StatusNotFound, "Instance data not found for this deployment")
		return
	}

	events, err := s.storage.GetDeploymentEvents(ctx, deploymentID)
	if err != nil {
		log.Printf("[Centro REST] Failed to get deployment events: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve events")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id":        deploymentID,
		"instance_data": instanceData,
		"events":        events,
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

	activeCount, err := s.storage.GetActiveDeploymentCount(ctx)
	if err != nil {
		log.Printf("[Centro REST] Failed to get active deployment count: %v", err)
		activeCount = 0
	}

	completedCount, err := s.storage.GetDeploymentHistoryCount(ctx)
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
		"deployments": map[string]interface{}{
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
