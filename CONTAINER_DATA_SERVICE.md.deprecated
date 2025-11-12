# SetContainerData Service

## Overview
The `SetContainerData` service allows agents to send detailed container inspection data to Centro after a container is running. This service uses `driver.InspectContainer()` to gather comprehensive information about running containers.

## Components Added

### 1. Proto Definition (`proto/agent.proto`)

#### New Messages:
- **`ContainerData`**: Contains detailed container inspection information
  - Container ID, name, image details
  - Command and arguments
  - Timestamps (created, started, finished)
  - Status, exit code, and PID
  - Labels, ports, and volumes

- **`SetContainerDataRequest`**: Request sent from agent to centro
  - Node ID
  - Job ID
  - Container data
  - Timestamp

- **`SetContainerDataResponse`**: Response from centro to agent
  - Acknowledged flag
  - Response message

#### New RPC Method:
```protobuf
rpc SetContainerData(SetContainerDataRequest) returns (SetContainerDataResponse);
```

### 2. Agent Side Implementation

#### gRPC Client (`agent/grpc/client.go`)
- Added `SetContainerData()` method to send container data to Centro
- Includes authentication via metadata
- 5-second timeout for RPC calls

#### Service (`agent/service/container/service.go`)
- **`SetContainerDataService`**: Service that:
  - Lists all containers managed by the driver
  - Filters for containers with job-id labels
  - Only processes containers in "running" state
  - Calls `driver.InspectContainer()` to get detailed info
  - Converts `model.ContainerInspect` to `pb.ContainerData`
  - Sends data to Centro via gRPC

#### Command (`agent/commands/set_container_data.go`)
- **`SetContainerDataCommand`**: Command wrapper for the service
- Runs every 30 seconds (configurable via `IntervalSeconds()`)
- Follows the same pattern as other agent commands

#### Integration (`agent/main.go`)
- Service initialized with gRPC client and driver
- Command registered with the executor
- Automatically runs on the scheduled interval

### 3. Centro Side Implementation

#### Server Handler (`centro/grpc/server.go`)
- **`SetContainerData()`** method handles incoming container data
- Validates required fields (node_id, job_id, container_data)
- Verifies job exists in active jobs
- Logs container information for monitoring
- Saves container data using dedicated `container_data` key in etcd (not as events)
- Returns acknowledgment response

#### Storage (`centro/storage/etcd/storage.go`)
- **`SaveContainerData()`** - Saves container data to etcd with key `/centro/jobs/container_data/{job_id}`
- **`GetContainerData()`** - Retrieves container data for a specific job
- Container data is stored separately from job events for efficient querying

#### REST API (`centro/rest/handlers.go`)
- **`GET /api/v1/jobs/{id}/container`** - Retrieves container data for a specific job
- Returns container data in JSON format
- Requires JWT authentication

## Data Flow

```
1. Container starts running
2. SetContainerDataCommand executes (every 30s)
3. Service calls driver.ListContainers()
4. For each running container:
   a. Call driver.InspectContainer(containerID)
   b. Convert to proto ContainerData
   c. Send to Centro via gRPC
5. Centro receives data
6. Centro validates and saves to container_data key
7. Centro responds with acknowledgment
```

## Container Data Collected

The following information is collected for each running container:

- **Identity**: Container ID, Name
- **Image**: Full image identifier and image name
- **Execution**: Command, arguments, PID
- **Timing**: Created, started, finished timestamps
- **State**: Status, exit code
- **Configuration**: Labels, ports, volumes

## Usage

The service is automatically enabled when the agent starts with a configured driver.

### Agent Configuration
```bash
# The service will automatically run if a driver is configured
export DRIVER_TYPE="podman"  # or "incus", "exec"
export CENTRO_SERVER_ADDR="localhost:50051"
export TOKEN="your-auth-token"

./agent_binary
```

### Monitoring Container Data

Container data is saved in a dedicated storage location and can be queried via:
- REST API: `GET /api/v1/jobs/{job_id}/container`
- Storage: etcd under `/centro/jobs/container_data/{job_id}` keys

Note: Job events are still available at `GET /api/v1/jobs/{job_id}/events` for other job-related events.

## Execution Interval

The service runs every **30 seconds** by default. To change this, modify the `IntervalSeconds()` method in `agent/commands/set_container_data.go`.

## Benefits

1. **Real-time Monitoring**: Centro receives up-to-date container information
2. **Debugging**: Detailed container state helps troubleshoot issues
3. **Audit Trail**: Container data is saved as job events for historical analysis
4. **Resource Tracking**: Monitor actual container PIDs, ports, and volumes
5. **Status Verification**: Validate container status matches job status

## Error Handling

- Service gracefully skips containers without job-id labels
- Only processes running containers
- Logs errors but continues processing other containers
- Failed gRPC calls are logged and retried on next interval

## Related Files

- `proto/agent.proto` - Proto definitions
- `proto/agent_grpc.pb.go` - Generated gRPC code
- `proto/agent.pb.go` - Generated proto code
- `agent/grpc/client.go` - gRPC client
- `agent/service/container/service.go` - Container service
- `agent/commands/set_container_data.go` - Command wrapper
- `agent/main.go` - Agent initialization
- `centro/grpc/server.go` - Centro server handler
- `agent/taskdriver/driver.go` - Driver interface
- `agent/taskdriver/model/container.go` - Container model

