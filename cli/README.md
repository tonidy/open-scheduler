# Centro client with cli friendly

use golang command pattern

sample cli

```bash
$ osctl get nodes

$ osctl describe node NODE_NAME

$ osctl get jobs

$ osctl describe job JOB_ID

$ osctl get jobs --active

$ osctl get jobs --failed

$ osctl get instance

$ osctl describe instance JOB_ID

$ osctl apply -f spec.yaml // submit new job

```

give sample yaml here

```yaml
# Example Job spec for Open Scheduler (based on proto Job definition)
job_id: "test-job-1"
job_name: "Test Job 1"
job_type: "single"
selected_clusters:
  - "default-cluster"
driver_type: "podman"
command: "/bin/sh -c 'echo hello world'"
instance_config:
  image_name: "alpine:latest"
  entrypoint:
    - "/bin/sh"
  arguments:
    - "-c"
    - "echo hello world"
  driver_options:
    restart_policy: "OnFailure"
environment_variables:
  ENV: "production"
  DEBUG: "false"
resource_requirements:
  memory_limit_mb: 128
  memory_reserved_mb: 64
  cpu_limit_cores: 0.5
  cpu_reserved_cores: 0.25
volume_mounts:
  - source_path: "/tmp/data"
    target_path: "/mnt/data"
    read_only: false
workload_type: "container"
job_metadata:
  creator: "admin"
  description: "A simple test batch job"
retry_count: 0
max_retries: 3
last_retry_time: 0
```
