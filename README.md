# How to Install

Follow the steps below to install and run the Open Scheduler prototype:

## Prerequisites

- **Go** (version 1.20 or higher)
- **Etcd** (see [README/ETCD_SETUP.md](README/ETCD_SETUP.md) for details)
- **Docker** or **Podman** (if running agent workloads as containers)
- (Optional) `protoc` for regenerating protobuf definitions

## Clone the Repository

```sh
git clone https://github.com/your-org/open-scheduler.git
cd open-scheduler
```

## Install Dependencies

```sh
go mod tidy
```

## Build the Control Plane (Centro)

```sh
cd centro
go build -o centro cmd/centro/main.go
```

## Build the Agent

```sh
cd ../agent
go build -o agent cmd/agent/main.go
```

## Running Etcd

Refer to [README/ETCD_SETUP.md](README/ETCD_SETUP.md) for installation and startup instructions.

## Running the Control Plane

```sh
./centro/centro
```

## Running the Agent

On each node (worker), run:

```sh
./agent/agent --node-id <node-identifier> --etcd-endpoints <etcd-host:port>
```

## API Documentation

- The Control Plane provides a REST API (see [README/REST_API_IMPLEMENTATION.md](README/REST_API_IMPLEMENTATION.md))
- Swagger UI is available at: `http://localhost:8080/swagger/`

---

> For more detailed developer and deployment documentation, see the [`README/`](README/) folder and the [Documentation Index](README/INDEX.md).

