package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/open-scheduler/agent/commands"
	sharedgrpc "github.com/open-scheduler/agent/grpc"
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

	grpcClient, err := sharedgrpc.NewSharedClient(serverAddr)
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

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, cleaning up...")
		cancel()
	}()

	executor := NewCommandExecutor()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = uuid.New().String()
	}
	executor.SetToken(token, nodeID)
	
	// Register commands
	executor.Register(commands.NewHeartbeatCommand(grpcClient))
	executor.Register(commands.NewGetJobCommand(grpcClient))

	executor.StartScheduler(ctx)

	log.Println("NodeAgent initialized with command pattern boilerplate")
	log.Println("Press Ctrl+C to exit")

	<-ctx.Done()
	log.Println("NodeAgent stopped")
}
