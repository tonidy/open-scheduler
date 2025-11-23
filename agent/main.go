package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-scheduler/agent/commands"
	agentgrpc "github.com/open-scheduler/agent/grpc"
	cleanupservice "github.com/open-scheduler/agent/service/cleanup"
	instanceservice "github.com/open-scheduler/agent/service/instance"
	statusservice "github.com/open-scheduler/agent/service/status"
	"github.com/open-scheduler/agent/taskdriver"
)

func main() {
	serverFlag := flag.String("server", "", "Centro server address (overrides CENTRO_SERVER_ADDR env var)")
	tokenFlag := flag.String("token", "", "Authentication token (overrides TOKEN env var)")
	flag.Parse()

	log.Println("Starting NodeAgent...")

	serverAddr := *serverFlag
	if serverAddr == "" {
		serverAddr = os.Getenv("CENTRO_SERVER_ADDR")
	}
	if serverAddr == "" {
		log.Fatalf("Server address not provided. Use --server flag or set CENTRO_SERVER_ADDR environment variable")
	}

	token := *tokenFlag
	if token == "" {
		token = os.Getenv("TOKEN")
	}
	if token == "" {
		log.Fatalf("Token not provided. Use --token flag or set TOKEN environment variable")
	}

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatalf("Failed to get machine hostname: %v", err)
		}
		nodeID = hostname
	}

	grpcClient, err := agentgrpc.NewGrpcClient(serverAddr)
	if err != nil {
		log.Fatalf("Failed to create gRPC client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := grpcClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer func() {
		if err := grpcClient.Close(); err != nil {
			log.Printf("Error closing gRPC client: %v", err)
		}
	}()

	log.Println("Successfully connected to Centro server")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Variable to store cleanup service for shutdown
	var cleanupSvc *cleanupservice.CleanupService

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, starting graceful shutdown...")

		// Execute cleanup to stop all running instances
		if cleanupSvc != nil {
			log.Println("Stopping all running instances...")
			// Create a timeout context for cleanup (30 seconds max)
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := cleanupSvc.Execute(cleanupCtx, nodeID, token); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
			cleanupCancel()
			log.Println("Cleanup completed, shutting down agent...")
		}

		cancel()
	}()

	executor := NewCommandExecutor()
	executor.SetToken(token, nodeID)

	driverName := os.Getenv("DRIVER_TYPE")
	if driverName == "" {
		driverName = "podman"
	}

	driver, err := taskdriver.NewDriver(driverName)
	if err != nil {
		log.Printf("Warning: Failed to initialize driver %s: %v", driverName, err)
		log.Printf("Status updates will be disabled")
		driver = nil
	}

	statusService, err := statusservice.NewUpdateStatusService(grpcClient, driver, token, nodeID)
	if err != nil {
		log.Fatalf("Failed to create UpdateStatusService: %v", err)
	}

	instanceService, err := instanceservice.NewSetInstanceDataService(grpcClient, driver, token, nodeID)
	if err != nil {
		log.Fatalf("Failed to create SetInstanceDataService: %v", err)
	}

	var cleanupService *cleanupservice.CleanupService
	cleanupService, err = cleanupservice.NewCleanupService(driver, nodeID)
	if err != nil {
		log.Fatalf("Failed to create CleanupService: %v", err)
	}

	// Assign to cleanup service variable for graceful shutdown
	cleanupSvc = cleanupService

	executor.Register(commands.NewHeartbeatCommand(grpcClient))
	executor.Register(commands.NewGetJobCommand(grpcClient, instanceService))
	executor.Register(commands.NewUpdateStatusCommand(statusService))
	executor.Register(commands.NewSetInstanceDataCommand(instanceService))
	executor.Register(commands.NewCleanUpInstancesCommand(cleanupService))

	executor.StartScheduler(ctx)

	log.Println("NodeAgent initialized with command pattern")
	log.Println("Press Ctrl+C to exit")

	<-ctx.Done()
	log.Println("NodeAgent stopped")
}
