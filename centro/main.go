package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	centrogrpc "github.com/open-scheduler/centro/grpc"
	"github.com/open-scheduler/centro/migration"
	etcdstorage "github.com/open-scheduler/centro/storage/etcd"
	pb "github.com/open-scheduler/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := flag.String("port", "50051", "The gRPC server port")
	etcdEndpoints := flag.String("etcd-endpoints", "localhost:2379", "Comma-separated list of etcd endpoints")
	flag.Parse()

	endpoints := strings.Split(*etcdEndpoints, ",")
	for i := range endpoints {
		endpoints[i] = strings.TrimSpace(endpoints[i])
	}

	log.Printf("[Centro] Connecting to etcd endpoints: %v", endpoints)
	storage, err := etcdstorage.NewStorage(endpoints)
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer storage.Close()

	log.Printf("[Centro] Successfully connected to etcd")

	address := fmt.Sprintf(":%s", *port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	log.Printf("[Centro] Starting gRPC server on %s", address)

	grpcServer := grpc.NewServer()

	centroServer := centrogrpc.NewCentroServer(storage)
	pb.RegisterNodeAgentServiceServer(grpcServer, centroServer)

	reflection.Register(grpcServer)

	go func() {
		time.Sleep(5 * time.Second)
		migration.SeedTestData(centroServer)
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			nodeCount := centroServer.GetNodeCount()
			queued, active, completed := centroServer.GetJobStats()
			log.Printf("[Centro] Status - Nodes: %d, Jobs (Queued: %d, Active: %d, Completed: %d)",
				nodeCount, queued, active, completed)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	log.Printf("[Centro] Server is ready and listening on %s", address)
	log.Println("[Centro] Press Ctrl+C to stop")

	<-sigChan
	log.Println("\n[Centro] Shutting down gracefully...")

	grpcServer.GracefulStop()
	log.Println("[Centro] Server stopped")
}
