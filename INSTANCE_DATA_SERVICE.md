# SetInstanceData Service

## Overview
The `SetInstanceData` service allows agents to send detailed instance inspection data to Centro after an instance (container or VM) is running. This service uses `driver.InspectInstance()` to gather comprehensive information about running instances.

## Components Added

### 1. Proto Definition (`proto/agent.proto`)

#### New Messages:
- **`InstanceData`**: Contains detailed instance inspection information
  - Instance ID, name, image details
  - Command and arguments
  - Timestamps (created, started, finished)
  - Status, exit code, and PID
  - Labels, ports, and volumes

- **`SetInstanceDataRequest`**: Request sent from agent to centro
  - Node ID
  - Job ID
  - Instance data
  - Timestamp

- **`SetInstanceDataResponse`**: Response from centro to agent
  - Acknowledged flag
  - Response message

#### New RPC Method:
```protobuf
rpc SetInstanceData(SetInstanceDataRequest) returns (SetInstanceDataResponse);
```

### 2. Agent Side Implementation

#### gRPC Client (`agent/grpc/client.go`)
- Added `SetInstanceData()` method to send instance data to Centro
- Includes authentication via metadata
- 5-second timeout for RPC calls

#### Service (`agent/service/instance/service.go`)
- **`SetInstanceDataService`**: Service that:
  - Lists all instances managed by the driver
  - Filters for instances with job-id labels
  - Only processes instances in "running" state
  - Calls `driver.InspectInstance()` to get detailed info
  - Converts instance data to `pb.InstanceData`
  - Sends data to Centro via gRPC

#### Command (`agent/commands/set_instance_data.go`)
- **`SetInstanceDataCommand`**: Command wrapper for the service
- Runs every 30 seconds (configurable via `IntervalSeconds()`)
- Follows the same pattern as other agent commands

#### Integration (`agent/main.go`)
- Service initialized with gRPC client and driver
- Command registered with the executor
- Automatically runs on the scheduled interval

### 3. Centro Side Implementation

#### Server Handler (`centro/grpc/server.go`)
- **`SetInstanceData()`** method handles incoming instance data
- Validates required fields (node_id, job_id, instance_data)
- Verifies job exists in active jobs
- Logs instance information for monitoring
- Saves instance data using dedicated `instance_data` key in etcd (not as events)
- Returns acknowledgment response

#### Storage (`centro/storage/etcd/storage.go`)
- **`SaveInstanceData()`** - Saves instance data to etcd with key `/centro/jobs/instance_data/{job_id}`
- **`GetInstanceData()`** - Retrieves instance data for a specific job
- Instance data is stored separately from job events for efficient querying

#### REST API (`centro/rest/handlers.go`)
- **`GET /api/v1/instances/{id}`** - Retrieves instance data for a specific job
- Returns instance data in JSON format
- Requires JWT authentication

## Data Flow

```
1. Instance starts running
2. SetInstanceDataCommand executes (every 30s)
3. Service calls driver.ListInstances()
4. For each running instance:
   a. Call driver.InspectInstance(instanceID)
   b. Convert to proto InstanceData
   c. Send to Centro via gRPC
5. Centro receives data
6. Centro validates and saves to instance_data key
7. Centro responds with acknowledgment
```

## Instance Data Collected

The following information is collected for each running instance:

- **Identity**: Instance ID, Name
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

### Monitoring Instance Data

Instance data is saved in a dedicated storage location and can be queried via:
- REST API: `GET /api/v1/instances/{job_id}`
- Storage: etcd under `/centro/jobs/instance_data/{job_id}` keys

Note: Job events are still available at `GET /api/v1/jobs/{job_id}/events` for other job-related events.

## Execution Interval

The service runs every **30 seconds** by default. To change this, modify the `IntervalSeconds()` method in `agent/commands/set_instance_data.go`.

## Benefits

1. **Real-time Monitoring**: Centro receives up-to-date instance information
2. **Debugging**: Detailed instance state helps troubleshoot issues
3. **Audit Trail**: Instance data is saved for historical analysis
4. **Resource Tracking**: Monitor actual instance PIDs, ports, and volumes
5. **Status Verification**: Validate instance status matches job status
6. **Multi-workload Support**: Works with containers, VMs, and processes

## Error Handling

- Service gracefully skips instances without job-id labels
- Only processes running instances
- Logs errors but continues processing other instances
- Failed gRPC calls are logged and retried on next interval

## Related Files

- `proto/agent.proto` - Proto definitions
- `proto/agent_grpc.pb.go` - Generated gRPC code
- `proto/agent.pb.go` - Generated proto code
- `agent/grpc/client.go` - gRPC client
- `agent/service/instance/service.go` - Instance service
- `agent/commands/set_instance_data.go` - Command wrapper
- `agent/main.go` - Agent initialization
- `centro/grpc/server.go` - Centro server handler
- `agent/taskdriver/driver.go` - Driver interface

