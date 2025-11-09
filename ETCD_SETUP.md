# etcd Integration Guide

## Overview

Centro now uses etcd as its persistent datastore, replacing the previous in-memory storage. This provides:
- **Persistence**: Data survives Centro server restarts
- **Distributed deployment**: Multiple Centro instances can share state
- **Consistency**: etcd's Raft consensus ensures data integrity
- **Scalability**: Enables horizontal scaling of the control plane

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Agent 1   │────▶│   Centro    │────▶│    etcd     │
└─────────────┘     │   Server    │     │   Cluster   │
┌─────────────┐     └─────────────┘     └─────────────┘
│   Agent 2   │────▶        │                   │
└─────────────┘             │                   │
┌─────────────┐             │            Persistent
│   Agent N   │────▶        │             Storage
└─────────────┘             └──────────────────┘
```

## Data Storage Layout

etcd stores data with the following key prefixes:

- `/centro/nodes/<node-id>` - Node registration and resource info
- `/centro/jobs/queue/<job-id>-<timestamp>` - Pending jobs queue (FIFO)
- `/centro/jobs/active/<job-id>` - Currently executing jobs
- `/centro/jobs/history/<job-id>` - Completed/failed job history

## Installation & Setup

### Option 1: Using Docker (Recommended for Development)

Start a single-node etcd cluster:

```bash
docker run -d --name etcd-centro \
  -p 2379:2379 \
  -p 2380:2380 \
  quay.io/coreos/etcd:v3.5.10 \
  /usr/local/bin/etcd \
  --name etcd0 \
  --listen-client-urls http://0.0.0.0:2379 \
  --advertise-client-urls http://0.0.0.0:2379 \
  --listen-peer-urls http://0.0.0.0:2380 \
  --initial-advertise-peer-urls http://0.0.0.0:2380 \
  --initial-cluster etcd0=http://0.0.0.0:2380
```

Verify etcd is running:
```bash
docker logs etcd-centro
curl http://localhost:2379/health
```

Stop etcd:
```bash
docker stop etcd-centro
docker rm etcd-centro
```

### Option 2: Using etcd Binary

Download and install etcd:
```bash
# macOS
brew install etcd

# Linux
ETCD_VER=v3.5.10
curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o etcd.tar.gz
tar xzvf etcd.tar.gz
cd etcd-${ETCD_VER}-linux-amd64
sudo cp etcd etcdctl /usr/local/bin/
```

Start etcd:
```bash
etcd --listen-client-urls http://localhost:2379 \
     --advertise-client-urls http://localhost:2379
```

### Option 3: Production etcd Cluster (3+ Nodes)

For production, run a multi-node etcd cluster for high availability:

**Node 1:**
```bash
etcd --name etcd-node1 \
  --initial-advertise-peer-urls http://node1:2380 \
  --listen-peer-urls http://node1:2380 \
  --listen-client-urls http://node1:2379,http://127.0.0.1:2379 \
  --advertise-client-urls http://node1:2379 \
  --initial-cluster-token etcd-cluster-1 \
  --initial-cluster etcd-node1=http://node1:2380,etcd-node2=http://node2:2380,etcd-node3=http://node3:2380 \
  --initial-cluster-state new
```

**Node 2 & 3:** Similar configuration with appropriate node names and URLs.

## Running Centro with etcd

### Default Configuration (localhost:2379)

```bash
cd centro
go run . 
```

### Custom etcd Endpoints

Single endpoint:
```bash
cd centro
go run . --etcd-endpoints localhost:2379
```

Multiple endpoints (for HA):
```bash
cd centro
go run . --etcd-endpoints "node1:2379,node2:2379,node3:2379"
```

### Custom Port + etcd

```bash
cd centro
go run . --port 50052 --etcd-endpoints localhost:2379
```

### Using Pre-built Binary

```bash
./centro/centro-server-etcd --etcd-endpoints localhost:2379
```

## etcd Operations

### Viewing Data

Install etcdctl:
```bash
# macOS
brew install etcdctl

# Or use from docker
docker exec etcd-centro etcdctl version
```

List all Centro data:
```bash
etcdctl get /centro --prefix --keys-only
```

View all nodes:
```bash
etcdctl get /centro/nodes/ --prefix
```

View job queue:
```bash
etcdctl get /centro/jobs/queue/ --prefix
```

View active jobs:
```bash
etcdctl get /centro/jobs/active/ --prefix
```

View job history:
```bash
etcdctl get /centro/jobs/history/ --prefix
```

### Clearing Data

Remove all Centro data (careful!):
```bash
etcdctl del /centro --prefix
```

Remove specific node:
```bash
etcdctl del /centro/nodes/<node-id>
```

Clear job queue:
```bash
etcdctl del /centro/jobs/queue/ --prefix
```

## Monitoring & Health Checks

Check etcd cluster health:
```bash
etcdctl endpoint health
etcdctl endpoint status --write-out=table
```

Check etcd metrics:
```bash
curl http://localhost:2379/metrics
```

Monitor Centro logs:
```bash
# You should see:
# [Centro] Connecting to etcd endpoints: [localhost:2379]
# [Centro] Successfully connected to etcd
# [Centro] Starting gRPC server on :50051
```

## Troubleshooting

### Connection Refused

**Problem:** Centro fails to connect to etcd
```
Failed to connect to etcd: context deadline exceeded
```

**Solution:**
- Verify etcd is running: `curl http://localhost:2379/health`
- Check etcd logs: `docker logs etcd-centro` or check etcd process output
- Verify firewall rules allow port 2379
- Ensure etcd is listening on correct interface (0.0.0.0 for Docker)

### Data Not Persisting

**Problem:** Data disappears after Centro restart

**Solution:**
- Verify etcd is running (not just Centro)
- Check etcd data directory is persistent (not tmpfs)
- Review etcd logs for errors

### Slow Operations

**Problem:** Job operations are slow

**Solution:**
- Check etcd latency: `etcdctl endpoint status`
- Monitor etcd metrics for slow disk I/O
- Consider using SSDs for etcd data directory
- Reduce network latency between Centro and etcd

### Split Brain / Inconsistent State

**Problem:** Different Centro instances see different data

**Solution:**
- Ensure all Centro instances connect to same etcd cluster
- Verify etcd cluster has quorum (majority of nodes healthy)
- Check for network partitions

## Migration from In-Memory Storage

If upgrading from the old in-memory version:

1. **Start etcd** first
2. **Start Centro** with etcd connection
3. **No migration needed** - Centro will start with empty etcd storage
4. **Old test jobs** from in-memory storage are lost (expected)

## Performance Considerations

- **Latency**: Each operation makes 1-2 etcd calls (typically <10ms on local network)
- **Throughput**: etcd can handle thousands of operations per second
- **Storage**: Each job ~1-10KB, each node ~1KB
- **Scaling**: For >10,000 jobs/min, consider batching operations

## Security (Production)

Enable TLS for production deployments:

```bash
# Generate certificates (example)
# Start etcd with TLS
etcd --cert-file=/path/to/server.crt \
     --key-file=/path/to/server.key \
     --client-cert-auth \
     --trusted-ca-file=/path/to/ca.crt

# Update Centro to use TLS (requires code changes)
# Add TLS config to etcd client creation
```

## Backup & Recovery

Backup etcd data:
```bash
etcdctl snapshot save backup.db
```

Restore from backup:
```bash
etcdctl snapshot restore backup.db --data-dir=/var/lib/etcd-restore
```

## Next Steps

- Implement etcd watch for real-time notifications
- Add leader election for active-passive Centro deployment
- Implement distributed locking for job assignment
- Add TTLs for automatic cleanup of old history
- Implement transaction-based job claiming for better consistency

## References

- [etcd Documentation](https://etcd.io/docs/)
- [etcd API Reference](https://etcd.io/docs/v3.5/learning/api/)
- [etcd Operations Guide](https://etcd.io/docs/v3.5/op-guide/)

