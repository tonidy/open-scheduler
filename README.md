# How to Install

Follow the steps below to install and run the Open Scheduler prototype:

## Prerequisites

- **Go** (version 1.20 or higher)
- **Etcd** (see [README/ETCD_SETUP.md](README/ETCD_SETUP.md) for details)
- **Docker** or **Podman** (if running agent workloads as containers)
- **Node.js** (version 18 or higher, for running the web panel)
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
./centro/centro --etcd-endpoints <etcd-host:port>
```

## Running the Agent

On each node (worker), run:

```sh
./agent/agent --node-id <node-identifier>
```

## Running the Panel

The web-based control panel provides a UI for managing jobs, nodes, and instances.

### Prerequisites

- **Node.js** (version 18 or higher)
- **npm** (comes with Node.js)

### Installation and Setup

1. Install dependencies:

```sh
cd panel
npm install
```

2. Start the development server:

```sh
npm run dev
```

The panel will be available at `http://localhost:3000`

> **Note:** The panel requires the Centro server to be running on port 8080. The development server automatically proxies API requests to the backend.

### Default Login Credentials

- **Username:** admin
- **Password:** admin123

### Building for Production

To build the panel for production:

```sh
npm run build
```

The production files will be in the `panel/dist` directory.

For more details, see the [Panel README](panel/README.md).
### Screenshot panel
<img width="1428" height="387" alt="image" src="https://github.com/user-attachments/assets/6b574b92-2e83-454f-8764-2fe752ef69a7" />
<img width="1435" height="750" alt="image" src="https://github.com/user-attachments/assets/6f6e446d-6ede-4ee0-83bb-97a12ccf7999" />



## Cli


### Using the CLI (`osctl`)

The CLI allows you to interact with the Centro control plane and managed jobs, nodes, and instances.

#### Build the CLI

```sh
go build -o osctl ./cli
```

#### Login (if authentication is enabled)

```sh
osctl login --username <admin> --password <admin123>
```

#### Example Usage

```sh
osctl get nodes

osctl describe node <NODE_ID>

osctl get jobs

osctl describe job <JOB_ID>

osctl get jobs --active

osctl get jobs --failed

osctl get instance

osctl describe instance <JOB_ID>

osctl apply -f sample-job.yaml   # Submit a new job from a YAML spec
```

#### Example Job YAML

See `cli/sample-job.yaml` for a real example.

```yaml
# Example Job spec for Open Scheduler (based on proto Job definition)
job_id: "test-job-1-cli"
job_name: "Test Job CLI"
job_type: "single"
selected_clusters:
  - "default-cluster"
driver_type: "podman"
command: "/bin/sh -c 'echo hello world'"
instance_config:
  image_name: "alpine:latest"
  entrypoint:
    - "/bin/sh"
  arguments:
    - "-c"
    - "echo hello world"
  driver_options:
    restart_policy: "OnFailure"
environment_variables:
  ENV: "production"
  DEBUG: "false"
resource_requirements:
  memory_limit_mb: 128
  memory_reserved_mb: 64
  cpu_limit_cores: 0.5
  cpu_reserved_cores: 0.25
volume_mounts:
  - source_path: "/tmp/data"
    target_path: "/mnt/data"
    read_only: false
workload_type: "container"
job_metadata:
  creator: "admin"
  description: "A simple test batch job"
retry_count: 0
max_retries: 3
last_retry_time: 0
```

- Edit this file as needed and submit with `osctl apply -f sample-job.yaml`

#### More

- Use `osctl --help` to browse all commands.
- For development/testing, see `demo.sh` to start all components and view logs quickly.




## API Documentation

- The Control Plane provides a REST API (see [README/REST_API_IMPLEMENTATION.md](README/REST_API_IMPLEMENTATION.md))
- Swagger UI is available at: `http://localhost:8080/swagger/`

---

> For more detailed developer and deployment documentation, see the [`README/`](README/) folder and the [Documentation Index](README/INDEX.md).

