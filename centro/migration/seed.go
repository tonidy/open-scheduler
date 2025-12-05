package migration

import (
	"log"

	centrogrpc "github.com/open-scheduler/centro/grpc"
	pb "github.com/open-scheduler/proto"
)

// SeedTestData adds sample test deployments to the Centro server for development/testing purposes
func SeedTestData(centroServer *centrogrpc.CentroServer) {
	log.Println("[Centro] Adding test deployments to the queue...")

	testDeployment := &pb.Deployment{
		DeploymentId:     "test-deployment-1",
		DeploymentName:   "Test Deployment 1",
		DeploymentType:   "batch",
		SelectedClusters: []string{"default"},
		DriverType:       "podman",
		WorkloadType:     "container",
		Replicas:         1,
		InstanceConfig: &pb.InstanceSpec{
			ImageName: "docker.io/library/alpine:latest",
			DriverOptions: map[string]string{
				"command": "echo 'Hello from test deployment!'",
			},
		},
		EnvironmentVariables: map[string]string{
			"TEST_VAR": "test_value",
		},
		ResourceRequirements: &pb.Resources{
			MemoryLimitMb:    512,
			MemoryReservedMb: 256,
			CpuLimitCores:    1.0,
			CpuReservedCores: 0.5,
		},
		DeploymentMetadata: map[string]string{
			"owner": "system",
		},
		RetryCount: 0,
		MaxRetries: 3,
	}
	centroServer.AddDeployment(testDeployment)

	testDeployment2 := &pb.Deployment{
		DeploymentId:     "test-deployment-2",
		DeploymentName:   "Test Deployment 2",
		DeploymentType:   "batch",
		SelectedClusters: []string{"sg-cluster"},
		DriverType:       "podman",
		WorkloadType:     "container",
		InstanceConfig: &pb.InstanceSpec{
			ImageName: "docker.io/library/ubuntu:latest",
			DriverOptions: map[string]string{
				"command": "echo 'Hello from test deployment 2!'",
			},
		},
		ResourceRequirements: &pb.Resources{
			MemoryLimitMb:    512,
			MemoryReservedMb: 256,
			CpuLimitCores:    1.0,
			CpuReservedCores: 0.5,
		},
		RetryCount: 0,
		MaxRetries: 3,
	}
	centroServer.AddDeployment(testDeployment2)

	log.Println("[Centro] Test deployments added")
}
