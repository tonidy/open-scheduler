# Cluster-Aware Terminology Improvements

## Summary

Updated the scheduler to use cluster-based terminology and implement cluster-aware job assignment.

## Key Changes

### 1. Proto Definitions (`proto/agent.proto`)

#### Job Specification
- **Changed**: `datacenters` (string) → `selected_clusters` (repeated string)
- **Meaning**: Jobs can now specify which clusters they should run on
- **Behavior**: Empty list = can run on any cluster

#### Heartbeat Request
- **Added**: `cluster_name` field (string)
- **Purpose**: Agents report which cluster they belong to
- **Default**: "default" (from `CLUSTER_NAME` environment variable)

### 2. Cluster Matching Logic (`centro/grpc/server.go`)

When an agent requests a job:
1. Centro dequeues a job from the queue
2. **NEW**: Checks if job has `selected_clusters` requirement
3. **NEW**: If job specifies clusters, verifies node's `cluster_name` matches
4. **NEW**: If no match, re-queues the job and returns "no matching jobs"
5. If match (or no requirement), assigns job to node

```go
// Example cluster matching logic
if len(job.SelectedClusters) > 0 {
    if !clusterMatches(job.SelectedClusters, node.ClusterName) {
        // Re-queue for nodes in the right cluster
        requeue(job)
        return noJobsAvailable
    }
}
```

### 3. Node Storage (`centro/storage/etcd/storage.go`)

#### NodeInfo Structure
- **Added**: `ClusterName` field
- **Updated**: Stored and persisted with heartbeat

### 4. Agent Configuration

#### Cluster Name Source
- **Environment Variable**: `CLUSTER_NAME`
- **Default Value**: "default"
- **Usage**: Set via environment when starting agent

Example:
```bash
# Agent in production cluster
export CLUSTER_NAME="production"
./agent_binary

# Agent in staging cluster  
export CLUSTER_NAME="staging"
./agent_binary

# Agent with default cluster
./agent_binary  # Uses "default"
```

### 5. REST API (`centro/rest/handlers.go`)

#### Job Submission
- **Field**: Still accepts `datacenters` (for backward compatibility)
- **Parsing**: Supports comma-separated cluster names
- **Storage**: Converts to `selected_clusters` array

Example REST request:
```json
{
  "name": "My Job",
  "type": "batch",
  "datacenters": "production,staging",  // Will run on either cluster
  "tasks": [...]
}
```

### 6. Test Data (`centro/migration/seed.go`)

Updated seed jobs to use:
```go
SelectedClusters: []string{"default"}
```

## Benefits

1. **Better Terminology**: "clusters" is more specific than "datacenters"
2. **Flexible Job Placement**: Jobs can target specific infrastructure clusters
3. **Multi-Cluster Support**: Single Centro can manage multiple clusters
4. **Environment Isolation**: Production/staging/development workloads separated
5. **Resource Optimization**: Jobs only assigned to appropriate nodes

## Use Cases

### Development vs Production
```bash
# Production nodes
CLUSTER_NAME=production ./agent_binary

# Development nodes  
CLUSTER_NAME=development ./agent_binary

# Jobs target specific clusters
selected_clusters: ["production"]  # Only production nodes
selected_clusters: ["development"] # Only dev nodes
selected_clusters: []              # Any cluster
```

### Geographic Distribution
```bash
# US cluster
CLUSTER_NAME=us-west ./agent_binary

# EU cluster
CLUSTER_NAME=eu-central ./agent_binary

# Jobs for EU compliance
selected_clusters: ["eu-central"]
```

### Specialized Hardware
```bash
# GPU cluster
CLUSTER_NAME=gpu-cluster ./agent_binary

# CPU cluster
CLUSTER_NAME=cpu-cluster ./agent_binary

# ML training jobs
selected_clusters: ["gpu-cluster"]

# Regular batch jobs
selected_clusters: ["cpu-cluster"]
```

## Backward Compatibility

- Existing jobs without `selected_clusters` will run on any cluster
- REST API still accepts `datacenters` field name
- Default cluster name is "default" if not specified

## Files Modified

1. ✅ `proto/agent.proto` - Added cluster_name to heartbeat, changed datacenters to selected_clusters
2. ✅ `agent/grpc/client.go` - Added clusterName parameter to SendHeartbeat
3. ✅ `agent/service/heartbeat/service.go` - Read CLUSTER_NAME env var
4. ✅ `centro/grpc/server.go` - Implemented cluster matching logic
5. ✅ `centro/storage/etcd/storage.go` - Added ClusterName to NodeInfo
6. ✅ `centro/rest/handlers.go` - Parse clusters from comma-separated string
7. ✅ `centro/migration/seed.go` - Updated test data
8. ✅ `agent/service/job/service.go` - Updated logging

## Testing

Start agent with cluster name:
```bash
export CLUSTER_NAME="test-cluster"
cd agent && go run .
```

Submit job targeting specific cluster:
```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Job",
    "type": "batch",
    "datacenters": "test-cluster",
    "tasks": [...]
  }'
```

## Migration Notes

For existing deployments:
1. Agents without `CLUSTER_NAME` will use "default"
2. Existing jobs without cluster specification work on all clusters
3. No breaking changes to existing functionality
4. Gradual migration: add `CLUSTER_NAME` to agents as needed

