# Proto Compatibility Analysis: template.yaml vs agent.proto

## Summary
✅ **UPDATED**: The `Job` message in `proto/agent.proto` has been **extended to fully support** the features defined in `template.yaml`. All missing features have been added.

## Status: ✅ FULLY SUPPORTED

All features from `template.yaml` are now supported in the proto definition.

## Features Added to proto/agent.proto

### 1. **Replicas** ✅
- **Field**: `int32 replicas = 16`
- **Status**: Added to `Job` message
- **Usage**: Specifies number of instances to run (default: 1)

### 2. **Placement Constraints** ✅
- **Message**: `Placement` with `constraints` and `strategy`
- **Field**: `Placement placement = 17` in `Job`
- **Status**: Fully supported
- **Usage**: Express complex placement logic (driver types, node labels, spread strategy)

### 3. **Working Directory** ✅
- **Field**: `string working_dir = 18` in `Job`
- **Status**: Added
- **Usage**: Set working directory for containers/instances

### 4. **Port Mappings** ✅
- **Message**: `PortMapping` with `host_port`, `container_port`, `protocol`
- **Field**: `repeated PortMapping ports = 19` in `Job`
- **Status**: Fully supported
- **Usage**: Configure port mappings for services

### 5. **Security Settings** ✅
- **Message**: `SecuritySettings` with `privileged`, `capabilities_add`, `capabilities_drop`, `read_only_root_filesystem`
- **Field**: `SecuritySettings security = 20` in `Job`
- **Status**: Fully supported
- **Usage**: Configure security settings (privileged mode, capabilities, read-only root)

### 6. **Health Checks** ✅
- **Message**: `HealthCheck` with `test`, `interval`, `timeout`, `retries`, `start_period`
- **Field**: `HealthCheck health_check = 21` in `Job`
- **Status**: Fully supported
- **Usage**: Configure health checks for services

### 7. **Restart Policy** ✅
- **Message**: `RestartPolicy` with `condition` and `max_attempts`
- **Field**: `RestartPolicy restart_policy = 22` in `Job`
- **Status**: Fully supported (separate from job retry policy)
- **Usage**: Configure restart behavior for failed containers/instances

### 8. **Network Configuration** ✅
- **Message**: `NetworkReference` with `name`
- **Field**: `repeated NetworkReference networks = 23` in `Job`
- **Status**: Fully supported
- **Usage**: Assign services to specific networks

### 9. **Instance Type** ✅
- **Field**: `string instance_type = 24` in `Job`
- **Status**: Added
- **Usage**: Distinguish between container and VM instances for Incus ("virtual-machine", "container")

### 10. **Image Source Configuration** ✅
- **Message**: `ImageSource` with `alias`, `server`, `mode`
- **Field**: `ImageSource image_source = 5` in `InstanceSpec`
- **Status**: Fully supported
- **Usage**: Specify image source server or pull mode

### 11. **User Data (Cloud-init)** ✅
- **Field**: `string user_data = 6` in `InstanceSpec`
- **Status**: Added
- **Usage**: Provide cloud-init configuration for VMs

### 12. **Device Configuration** ✅
- **Message**: `Device` with `name`, `type`, `properties`
- **Field**: `repeated Device devices = 7` in `InstanceSpec`
- **Status**: Fully supported
- **Usage**: Configure network devices for instances

### 13. **Command Format** ✅
- **Field**: `repeated string command_array = 25` in `Job` (alongside existing `command` string)
- **Status**: Added (supports both string and array formats)
- **Usage**: Support array commands like `["nginx", "-g", "daemon off;"]`

### 14. **Volume Type** ✅
- **Field**: `string type = 4` added to `Volume` message
- **Status**: Added
- **Usage**: Specify mount type ("bind", "volume", etc.)

### Note: Top-level Networks and Volumes
- **Status**: Not directly in proto (these are orchestrator-level concepts)
- **Impact**: Networks/volumes are referenced by name in jobs, but their definitions are handled at the orchestrator level, not in the Job proto message

## Currently Supported Features ✅

1. ✅ Basic job identification (`job_id`, `job_name`, `job_type`)
2. ✅ Cluster selection (`selected_clusters`)
3. ✅ Driver type (`driver_type`)
4. ✅ Image name (`InstanceSpec.image_name`)
5. ✅ Entrypoint/arguments (`InstanceSpec.entrypoint`, `arguments`)
6. ✅ Environment variables (`environment_variables`)
7. ✅ Resource limits (`Resources` - CPU/memory limits and reservations)
8. ✅ Volume mounts (`Volume` - source_path, target_path, read_only)
9. ✅ Workload type (`workload_type`)
10. ✅ Job metadata (`job_metadata`)
11. ✅ Retry configuration (`retry_count`, `max_retries`)

## Recommendations

To fully support `template.yaml`, the proto needs to be extended with:

1. **Placement message** - For constraints and strategy
2. **PortMapping message** - For port configurations
3. **SecuritySettings message** - For security configurations
4. **HealthCheck message** - For health check configuration
5. **RestartPolicy message** - For restart behavior
6. **NetworkReference** - For network assignments
7. **ImageSource message** - For image source configuration
8. **Additional fields** - replicas, working_dir, instance_type, user_data, devices


