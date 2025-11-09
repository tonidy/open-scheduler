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
		JobId:       "test-job-1",
		Name:        "Test Job 1",
		Type:        "batch",
		Datacenters: "dc1",
		Tasks: []*pb.Task{
			{
				Name:   "test-task",
				Driver: "podman",
				Kind:   "container",
				Config: &pb.ContainerSpec{
					Image: "docker.io/library/alpine:latest",
					Options: map[string]string{
						"command": "echo 'Hello from test job!'",
					},
				},
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
		},
		Meta: map[string]string{
			"owner": "system",
		},
	}
	centroServer.AddJob(testJob)

	testJob2 := &pb.Job{
		JobId:       "test-job-2",
		Name:        "Test Job 2",
		Type:        "batch",
		Datacenters: "dc1",
		Tasks: []*pb.Task{
			{
				Name:   "test-task-2",
				Driver: "podman",
				Config: &pb.ContainerSpec{
					Image: "docker.io/library/ubuntu:latest",
					Options: map[string]string{
						"command": "echo 'Hello from test job 2!'",
					},
				},
			},
		},
	}
	centroServer.AddJob(testJob2)

	log.Println("[Centro] Test jobs added")
}
