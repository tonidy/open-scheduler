# gRPC Setup for Open Scheduler

This document describes the gRPC client implementation for the Open Scheduler agent.

## What Was Created

### 1. Protocol Buffer Files

#### Generated Proto Files
- **`proto/agent.pb.go`** - Generated Go protobuf messages
- **`proto/agent_grpc.pb.go`** - Generated gRPC service client and server stubs

These files were generated from `proto/agent.proto` and provide:
- Message types: `HeartbeatRequest`, `HeartbeatResponse`, etc.
- Client interface: `NodeAgentServiceClient`
- Server interface: `NodeAgentServiceServer`

### 2. Heartbeat Service Implementation

#### `agent/service/heartbeat/grpc.go`
Complete gRPC client implementation with:
- `GRPCClient` struct for managing connections
- `NewGRPCClient()` - Creates a new gRPC client
- `Connect()` - Establishes connection to the server
- `SendHeartbeat()` - Sends heartbeat with metrics and metadata
- `Close()` - Closes the connection
- `IsConnected()` - Checks connection status

Features:
- Automatic timeout handling (10s for connection, 5s for RPC calls)
- Token-based authentication via gRPC metadata
- Insecure credentials (for development)
- Comprehensive error handling

#### `agent/service/heartbeat/service.go`
HeartbeatService wrapper that:
- Uses the GRPCClient internally
- Implements the Command pattern for scheduled execution
- Provides default resource metrics (can be extended with real system metrics)
- Handles connection management automatically

### 3. Integration with Agent

#### `agent/commands/heartbeat.go`
Updated HeartbeatCommand to:
- Create and use HeartbeatService
- Read server address from `GRPC_SERVER_ADDR` environment variable
- Execute heartbeats via gRPC

### 4. Build Configuration

#### `Makefile`
Added targets for:
- `proto` - Generate protobuf files
- `install-tools` - Install protoc-gen-go tools
- `clean` - Remove generated files
- `run-agent` - Run the agent
- `run-centro` - Run the control plane

### 5. Documentation and Examples

#### `agent/service/heartbeat/README.md`
Complete documentation with:
- Usage examples
- API reference
- Environment variables
- Proto definitions

#### `examples/heartbeat_client.go`
Working examples showing:
- Using HeartbeatService (recommended approach)
- Using gRPC client directly (for custom use cases)

## How to Use

### Running the Agent

1. **Set environment variables:**
```bash
export TOKEN="your-auth-token"
export NODE_ID="node-123"
export GRPC_SERVER_ADDR="localhost:50051"
```

2. **Build the agent:**
```bash
cd agent
go build -v
```

3. **Run the agent:**
```bash
./agent
```

The agent will automatically:
- Connect to the gRPC server at `localhost:50051`
- Send heartbeats every 15 seconds
- Include node metrics and metadata

### Running the Example

```bash
cd examples
go run heartbeat_client.go
```

This will demonstrate both the service wrapper and direct client usage.

## Architecture

```
┌─────────────────────────────────────┐
│         NodeAgent (Client)          │
│                                     │
│  ┌───────────────────────────────┐ │
│  │   HeartbeatCommand            │ │
│  │   (Scheduled every 15s)       │ │
│  └──────────┬────────────────────┘ │
│             │                       │
│  ┌──────────▼────────────────────┐ │
│  │   HeartbeatService            │ │
│  │   (Business Logic)            │ │
│  └──────────┬────────────────────┘ │
│             │                       │
│  ┌──────────▼────────────────────┐ │
│  │   GRPCClient                  │ │
│  │   (gRPC Connection)           │ │
│  └──────────┬────────────────────┘ │
└─────────────┼───────────────────────┘
              │
              │ gRPC/HTTP2
              │
┌─────────────▼───────────────────────┐
│      ControlPlane (Server)          │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  NodeAgentServiceServer       │ │
│  │  (Implements Heartbeat RPC)   │ │
│  └───────────────────────────────┘ │
└─────────────────────────────────────┘
```

## gRPC Service Definition

The `NodeAgentService` provides three RPCs:

1. **Heartbeat** - Node sends periodic heartbeat with metrics
   - Request: `HeartbeatRequest` (node_id, timestamp, metrics, metadata)
   - Response: `HeartbeatResponse` (ok, message)

2. **GetJob** - Node requests a job to execute
   - Request: `GetJobRequest` (node_id)
   - Response: `GetJobResponse` (has_job, job, message)

3. **UpdateStatus** - Node updates job status
   - Request: `UpdateStatusRequest` (node_id, job_id, status, detail, timestamp)
   - Response: `UpdateStatusResponse` (ok, message)

## Authentication

The client uses bearer token authentication via gRPC metadata:

```go
md := metadata.New(map[string]string{
    "authorization": fmt.Sprintf("Bearer %s", token),
})
ctx = metadata.NewOutgoingContext(ctx, md)
```

The server can extract and validate this token from the context.

## Next Steps

To complete the system, you need to:

1. **Implement the gRPC Server** in `centro/` (ControlPlane)
   - Implement `NodeAgentServiceServer` interface
   - Handle Heartbeat, GetJob, and UpdateStatus RPCs
   - Validate authentication tokens

2. **Add System Metrics Collection**
   - Replace hardcoded metrics with real system data
   - Use libraries like `gopsutil` for RAM, CPU, disk metrics

3. **Add TLS Support** (for production)
   - Replace `insecure.NewCredentials()` with TLS credentials
   - Generate certificates for server and clients

4. **Add Health Checks**
   - Implement gRPC health check protocol
   - Monitor connection status

5. **Add Retry Logic**
   - Implement exponential backoff for failed RPCs
   - Handle transient network failures

## Dependencies

The following dependencies were added to `go.mod`:

```
google.golang.org/grpc v1.59.0
google.golang.org/protobuf v1.31.0
```

## Testing

To test the implementation:

1. Start a gRPC server on `localhost:50051` that implements `NodeAgentService`
2. Run the agent with appropriate environment variables
3. Verify heartbeats are being sent and received

## Troubleshooting

### "Connection refused"
- Ensure the gRPC server is running on the specified address
- Check firewall rules

### "Method not implemented"
- Ensure the server implements the `NodeAgentService` interface
- Verify proto files are in sync between client and server

### "Authentication failed"
- Verify the TOKEN environment variable is set correctly
- Check server-side token validation logic

## References

- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffers](https://protobuf.dev/)
- [Proto file](proto/agent.proto)

