# Terminology Simplification: Job to Container 1:1 Relationship

## Overview
The codebase has been refactored to simplify the relationship between Jobs and Containers. Previously, a Job could contain multiple Tasks, each creating a container. Now, **each Job directly represents and runs exactly one container**.

## Key Changes

### Data Model Changes

#### Before (Complex Model)
```
Job
  ├── JobId, JobName, JobType
  ├── SelectedClusters
  └── Tasks[] (array of tasks)
       └── Task
            ├── TaskName
            ├── DriverType
            ├── ContainerConfig
            ├── EnvironmentVariables
            ├── ResourceRequirements
            └── VolumeMounts
```

#### After (Simplified Model)
```
Job (1:1 with Container)
  ├── JobId, JobName, JobType
  ├── SelectedClusters
  ├── DriverType
  ├── ContainerConfig
  ├── EnvironmentVariables
  ├── ResourceRequirements
  └── VolumeMounts
```

### Modified Files

#### Proto Definition
- **`proto/agent.proto`**: 
  - Removed `repeated Task tasks` field from Job message
  - Merged Task fields directly into Job message
  - Updated comments to reflect simplified model

#### Agent Side
- **`agent/taskdriver/driver.go`**: Changed `Run(ctx, *pb.Task)` → `Run(ctx, *pb.Job)`
- **`agent/taskdriver/podman/driver.go`**: Updated to work with Job directly
- **`agent/taskdriver/exec/driver.go`**: Updated to work with Job directly
- **`agent/taskdriver/incus/driver.go`**: Updated to work with Job directly
- **`agent/service/job/service.go`**: Simplified `handleJob()` - no longer loops through tasks
- **`agent/taskdriver/model/container.go`**: Deleted (no longer needed)

#### Centro (Server) Side
- **`centro/rest/handlers.go`**:
  - Simplified `SubmitJobRequest` structure (removed Tasks array)
  - Merged task fields directly into job request
  - Updated `handleSubmitJob()` to create Job without Tasks array
  
- **`centro/migration/seed.go`**: Updated test data to use simplified Job structure
- **`centro/grpc/server.go`**: Simplified `calculateJobResourceRequirements()` - no longer iterates through tasks

### Container Labels
- Container label changed from `open-scheduler.task-name` → `open-scheduler.job-name`
- Job ID is set directly: `open-scheduler.job-id`

### API Changes

#### Old Job Submission Format
```json
{
  "name": "web-server",
  "type": "service",
  "tasks": [
    {
      "name": "nginx-task",
      "driver": "podman",
      "container_config": { "image": "nginx:latest" },
      "resources": { "memory_mb": 512, "cpu": 1.0 }
    }
  ]
}
```

#### New Job Submission Format (Simplified)
```json
{
  "name": "web-server",
  "type": "service",
  "driver": "podman",
  "container_config": { "image": "nginx:latest" },
  "resources": { "memory_mb": 512, "cpu": 1.0 }
}
```

## Benefits

1. **Simpler Mental Model**: One job = one container, easier to reason about
2. **Less Code**: ~124 lines of code removed
3. **Clearer API**: Job submission is more straightforward
4. **Easier Debugging**: 1:1 relationship makes tracking and monitoring simpler
5. **Better Resource Management**: Resource calculation is direct, not aggregated

## Migration Notes

- Existing jobs with multiple tasks will need to be split into separate jobs
- API clients submitting jobs need to update their request format
- The relationship is now strictly 1:1 - if you need multiple containers, submit multiple jobs

## Testing

All code compiles successfully with no linter errors. Run tests with:
```bash
go test ./... -short
```

Build the project with:
```bash
go build ./...
```

