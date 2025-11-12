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
./centro/centro
```

## Running the Agent

On each node (worker), run:

```sh
./agent/agent --node-id <node-identifier> --etcd-endpoints <etcd-host:port>
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



## API Documentation

- The Control Plane provides a REST API (see [README/REST_API_IMPLEMENTATION.md](README/REST_API_IMPLEMENTATION.md))
- Swagger UI is available at: `http://localhost:8080/swagger/`

---

> For more detailed developer and deployment documentation, see the [`README/`](README/) folder and the [Documentation Index](README/INDEX.md).

