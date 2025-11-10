package migration

import (
	"log"

	centrogrpc "github.com/open-scheduler/centro/grpc"
	pb "github.com/open-scheduler/proto"
)

// SeedTestData adds sample test jobs to the Centro server for development/testing purposes
func SeedTestData(centroServer *centrogrpc.CentroServer) {
	log.Println("[Centro] Adding test jobs to the queue...")

	testJob := &pb.Job{
		JobId:            "test-job-1",
		JobName:          "Test Job 1",
		JobType:          "batch",
		SelectedClusters: []string{"default"},
		Tasks: []*pb.Task{
			{
				TaskName:     "test-task",
				DriverType:   "podman",
				WorkloadType: "container",
				ContainerConfig: &pb.ContainerSpec{
					ImageName: "docker.io/library/alpine:latest",
					DriverOptions: map[string]string{
						"command": "echo 'Hello from test job!'",
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
			},
		},
		JobMetadata: map[string]string{
			"owner": "system",
		},
	}
	centroServer.AddJob(testJob)

	testJob2 := &pb.Job{
		JobId:            "test-job-2",
		JobName:          "Test Job 2",
		JobType:          "batch",
		SelectedClusters: []string{ "test-cluster"},
		Tasks: []*pb.Task{
			{
				TaskName:     "test-task-2",
				DriverType:   "podman",
				WorkloadType: "container",
				ContainerConfig: &pb.ContainerSpec{
					ImageName: "docker.io/library/ubuntu:latest",
					DriverOptions: map[string]string{
						"command": "echo 'Hello from test job 2!'",
					},
				},
				ResourceRequirements: &pb.Resources{
					MemoryLimitMb:    512,
					MemoryReservedMb: 256,
					CpuLimitCores:    1.0,
					CpuReservedCores: 0.5,
				},
			},
		},
	}
	centroServer.AddJob(testJob2)

	log.Println("[Centro] Test jobs added")
}
