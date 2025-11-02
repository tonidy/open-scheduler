# NodeAgent - Command Pattern Implementation

## Overview
The NodeAgent implements the Command Pattern to handle various operations in a structured and extensible way.

## Architecture

### Command Pattern Structure
- **Command Interface**: Defines the contract for all commands (`Execute`, `Name`)
- **CommandExecutor**: Manages and executes registered commands
- **Concrete Commands**: Individual command implementations

## Available Commands

### 1. HeartbeatCommand
Sends periodic heartbeat to the ControlPlane to indicate node availability.

```go
cmd := commands.NewHeartbeatCommand(nodeID)
executor.Register(cmd)
```

### 2. GetJobCommand
Requests a job from the ControlPlane for execution.

```go
cmd := commands.NewGetJobCommand(nodeID)
executor.Register(cmd)
```

### 3. UpdateStatusCommand
Updates the status of a running job back to the ControlPlane.

```go
cmd := commands.NewUpdateStatusCommand(nodeID, jobID, status, detail)
executor.Register(cmd)
```

## Usage

```go
// Create executor
executor := NewCommandExecutor()

// Register commands
executor.Register(commands.NewHeartbeatCommand("node-1"))
executor.Register(commands.NewGetJobCommand("node-1"))

// Execute specific command
err := executor.ExecuteCommand(ctx, "heartbeat")

// Or execute all commands
err := executor.ExecuteAll(ctx)
```

## Adding New Commands

To add a new command:

1. Create a new file in `agent/commands/`
2. Implement the `Command` interface:
   - `Execute(ctx context.Context) error`
   - `Name() string`
3. Register it in `main.go`

## Future Enhancements
- Implement actual gRPC communication
- Add command scheduling and queuing
- Implement command retry logic
- Add command history/logging

