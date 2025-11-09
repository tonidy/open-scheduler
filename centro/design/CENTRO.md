# Centro Server Implementation

## Completed Tasks

✅ Created gRPC server in `centro/grpc/server.go`
✅ Updated Centro main in `centro/main.go`
✅ Fixed agent in `agent/main.go`
✅ Removed all code comments

## Current Implementation

### centro/storage/etcd/storage.go
- Storage: etcd client wrapper for persistent data storage
- NodeInfo: Tracks node resources (CPU, RAM, Disk) and metadata
- JobStatus: Tracks job lifecycle and ownership
- SaveNode/GetNode/GetAllNodes: Node management operations
- EnqueueJob/DequeueJob/GetQueueLength: Job queue operations
- SaveJobActive/GetJobActive/DeleteJobActive: Active job tracking
- SaveJobHistory/GetJobHistoryCount: Job completion history

### centro/grpc/server.go
- CentroServer: Main server struct using etcd storage backend
- Heartbeat: Registers nodes and updates their status in etcd
- GetJob: Distributes jobs from etcd queue to nodes
- ClaimJob: Allows nodes to claim jobs with etcd-backed state
- UpdateStatus: Receives job status updates and persists to etcd
- monitorNodes: Background process checking for offline nodes (60s timeout)

### centro/main.go
- Connects to etcd cluster (configurable endpoints)
- Starts gRPC server on configurable port (default: 50051)
- Adds test jobs after 5 seconds
- Periodic status reporting every 30 seconds
- Graceful shutdown with etcd connection cleanup

## Next Steps for Development

### 1. Persistent Storage ✅ COMPLETED
- ✅ Replaced in-memory maps with etcd distributed key-value store
- ✅ Store: nodes, job queue, active jobs, job history
- ✅ Benefits: Survives restarts, enables multi-instance deployment, distributed consensus

### 2. Job Scheduling Logic
- Implement resource-based scheduling (match job requirements with node capacity)
- Add datacenter filtering (job.Datacenters vs node.Metadata["datacenter"])
- Priority queues for different job types
- Fair scheduling across nodes

### 3. Job Queue Management
- REST/gRPC API to submit new jobs (currently only test jobs exist)
- Job validation before adding to queue
- Job cancellation support
- Job retry logic for failed jobs

### 4. Node Health Management
- Mark jobs as failed when node goes offline
- Automatic job requeue for crashed nodes
- Node blacklist/whitelist
- Resource quota enforcement

### 5. Security & Authentication
- Implement token validation in all RPC methods
- TLS/mTLS for gRPC connections
- Node registration approval workflow
- RBAC for job submission

### 6. Observability
- Metrics endpoint (Prometheus format)
- Structured logging (JSON format)
- Distributed tracing (OpenTelemetry)
- Dashboard for monitoring (Grafana)

### 7. High Availability
- Support multiple Centro instances with leader election
- Shared state via database or consensus (etcd/Consul)
- Load balancing across Centro servers
- Backup and restore procedures

### 8. Advanced Features
- Job dependencies and workflows
- Scheduled/cron jobs
- Job templates and parameterization
- Multi-task job support with dependencies
- Resource reservations

### 9. Testing
- Unit tests for all RPC handlers
- Integration tests with agent
- Load testing for job queue throughput
- Chaos testing for node failures

### 10. Documentation
- API documentation (OpenAPI/Swagger)
- Deployment guide
- Troubleshooting runbook
- Architecture diagrams

## Usage

### Prerequisites
Start etcd (version 3.5+):
```bash
# Using docker
docker run -d --name etcd \
  -p 2379:2379 \
  -p 2380:2380 \
  quay.io/coreos/etcd:v3.5.10 \
  /usr/local/bin/etcd \
  --listen-client-urls http://0.0.0.0:2379 \
  --advertise-client-urls http://0.0.0.0:2379

# Or using etcd binary
etcd --listen-client-urls http://localhost:2379 \
     --advertise-client-urls http://localhost:2379
```

### Start the Centro server:
```bash
make run-centro
```

Or with custom configuration:
```bash
cd centro && go run . --port 50052 --etcd-endpoints localhost:2379
```

For multiple etcd nodes:
```bash
cd centro && go run . --etcd-endpoints "node1:2379,node2:2379,node3:2379"
```

### Start the agent:
```bash
cd agent && CENTRO_SERVER_ADDR=localhost:50051 TOKEN=test-token go run .
```

## Architecture Notes

Current design uses:
- etcd distributed key-value store for persistent storage
- FIFO job queue in etcd (no prioritization yet)
- Simple node registration (no validation)
- Goroutines for background tasks (monitoring, status reporting)
- etcd for distributed consensus and data consistency

Benefits of etcd integration:
- Data survives Centro server restarts
- Enables multiple Centro instances with shared state
- Built-in distributed consensus via Raft protocol
- Watch/notification support for real-time updates (future enhancement)
- Automatic leader election capability (not yet implemented)

When scaling further, consider:
- Implementing etcd watch for real-time job updates
- Leader election for active-passive Centro deployment
- Priority queues using etcd key prefixes
- WebSocket/gRPC streaming for live job updates
- Distributed locking for critical operations
