package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/google/uuid"
	"github.com/open-scheduler/agent/commands"
)

var intervalSeconds = 15

func main() {
	log.Println("Starting NodeAgent...")
	log.Println("Interval seconds:", intervalSeconds)
	intervalEnv := os.Getenv("INTERVAL_SECONDS")
	if intervalEnv != "" {
		val, _ := strconv.Atoi(intervalEnv)
		intervalSeconds = val
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatalf("TOKEN is not set")
	}
	executor.SetToken(token, nodeID)
	executor.Register(commands.NewHeartbeatCommand())
	executor.Register(commands.NewGetJobCommand())
	executor.Register(commands.NewUpdateStatusCommand())

	executor.StartScheduler(ctx)

	log.Println("NodeAgent initialized with command pattern boilerplate")
	log.Println("Press Ctrl+C to exit")

	<-ctx.Done()
	log.Println("NodeAgent stopped")
}
