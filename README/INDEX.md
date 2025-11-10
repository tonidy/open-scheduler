# Documentation Index

This folder contains all the documentation for the Open Scheduler project.

## Architecture & Design

### Core Components
- **[SCHEDULER.md](SCHEDULER.md)** - Agent scheduler design and architecture
- **[CENTRO.md](CENTRO.md)** - Centro (Control Plane) design and architecture
- **[CLUSTER_TERMINOLOGY.md](CLUSTER_TERMINOLOGY.md)** - Cluster-aware job scheduling terminology and implementation

### Component Documentation
- **[AGENT.md](AGENT.md)** - Agent (Data Plane) documentation
- **[API.md](API.md)** - Centro API design and specifications

## Setup & Configuration

- **[ETCD_SETUP.md](ETCD_SETUP.md)** - etcd storage backend setup guide
- **[GRPC_SETUP.md](GRPC_SETUP.md)** - gRPC service setup and configuration

## API Documentation

- **[REST_API_IMPLEMENTATION.md](REST_API_IMPLEMENTATION.md)** - REST API implementation details
- **[SWAGGER.md](SWAGGER.md)** - Swagger/OpenAPI documentation setup

## Workflows

- **[workflow.md](workflow.md)** - System workflows and operational procedures

## Quick Links

### Getting Started
1. [Setup etcd](ETCD_SETUP.md)
2. [Configure gRPC](GRPC_SETUP.md)
3. [Understand Centro architecture](CENTRO.md)
4. [Understand Agent architecture](AGENT.md)

### Working with the System
- Submit jobs via [REST API](REST_API_IMPLEMENTATION.md)
- View API documentation using [Swagger](SWAGGER.md)
- Configure cluster-based scheduling with [Cluster Terminology](CLUSTER_TERMINOLOGY.md)

### Architecture Deep Dive
- [Control Plane (Centro)](CENTRO.md) - Job queue, scheduling logic
- [Data Plane (Agent)](AGENT.md) - Job execution, heartbeats
- [Scheduler Design](SCHEDULER.md) - Overall scheduling architecture
- [System Workflows](workflow.md) - End-to-end system flows

## Document Organization

```
README/
├── INDEX.md                      # This file
├── AGENT.md                      # Agent documentation
├── API.md                        # API design
├── CENTRO.md                     # Centro documentation
├── CLUSTER_TERMINOLOGY.md        # Cluster terminology guide
├── ETCD_SETUP.md                 # etcd setup
├── GRPC_SETUP.md                 # gRPC setup
├── REST_API_IMPLEMENTATION.md    # REST API implementation
├── SCHEDULER.md                  # Scheduler design
├── SWAGGER.md                    # Swagger documentation
└── workflow.md                   # System workflows
```

## Contributing

When adding new documentation:
1. Place markdown files in this `README/` directory
2. Update this INDEX.md with a link and description
3. Keep documentation focused and organized by topic
4. Use clear, descriptive filenames

## Related Files

- **Main README**: `/README.md` - Project overview and quick start
- **API Specs**: `/docs/` - Swagger/OpenAPI specifications
- **Proto Definitions**: `/proto/agent.proto` - Protocol buffer definitions

