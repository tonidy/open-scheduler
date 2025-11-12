<script>
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Label, Input, Select, Textarea, Alert } from 'flowbite-svelte';
  import { ArrowLeftOutline } from 'flowbite-svelte-icons';
  import { jobs } from '../api/client';

  let loading = false;
  let error = '';
  let success = '';

  // Form fields
  let jobName = '';
  let jobType = 'service';
  let driver = 'podman';
  let workloadType = 'container';
  let command = '';
  let image = 'docker.io/library/alpine:latest';
  let memoryMB = 512;
  let cpuCores = 1.0;

  const driverOptions = [
    { value: 'podman', name: 'Podman' },
    { value: 'incus', name: 'Incus' },
    { value: 'exec', name: 'Exec' },
    { value: 'process', name: 'Process' },
  ];

  const workloadTypeOptions = [
    { value: 'container', name: 'Container' },
    { value: 'process', name: 'Process' },
    { value: 'vm', name: 'Virtual Machine' },
  ];

  async function handleSubmit() {
    error = '';
    success = '';
    loading = true;

    if (!jobName) {
      error = 'Job name is required';
      loading = false;
      return;
    }

    const jobData = {
      job_name: jobName,
      job_type: jobType,
      driver: driver,
      workload_type: workloadType,
      command: command,
      instance_config: workloadType === 'container' ? {
        image: image,
      } : undefined,
      resources: {
        memory_mb: parseInt(memoryMB),
        cpu: parseFloat(cpuCores),
      },
    };

    try {
      const response = await jobs.create(jobData);
      success = `Job created successfully! Job ID: ${response.job_id}`;
      setTimeout(() => {
        navigate(`/jobs/${response.job_id}`);
      }, 1500);
    } catch (err) {
      error = err.message || 'Failed to create job';
    } finally {
      loading = false;
    }
  }
</script>

<div class="space-y-6">
  <div class="flex items-center gap-4">
    <Button color="light" on:click={() => navigate('/jobs')}>
      <ArrowLeftOutline class="w-4 h-4 mr-2" />
      Back
    </Button>
    <Heading tag="h2" class="text-2xl font-bold">Create New Job</Heading>
  </div>

  {#if error}
    <Alert color="red" dismissable>
      <span class="font-medium">Error!</span> {error}
    </Alert>
  {/if}

  {#if success}
    <Alert color="green">
      <span class="font-medium">Success!</span> {success}
    </Alert>
  {/if}

  <Card>
    <form on:submit|preventDefault={handleSubmit} class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Job Name -->
        <div>
          <Label for="job_name" class="mb-2">Job Name *</Label>
          <Input
            id="job_name"
            type="text"
            bind:value={jobName}
            required
            placeholder="my-job"
          />
        </div>

        <!-- Job Type -->
        <div>
          <Label for="job_type" class="mb-2">Job Type</Label>
          <Input
            id="job_type"
            type="text"
            bind:value={jobType}
            placeholder="service"
          />
        </div>

        <!-- Driver -->
        <div>
          <Label for="driver" class="mb-2">Driver *</Label>
          <Select id="driver" bind:value={driver} items={driverOptions} />
        </div>

        <!-- Workload Type -->
        <div>
          <Label for="workload_type" class="mb-2">Workload Type *</Label>
          <Select id="workload_type" bind:value={workloadType} items={workloadTypeOptions} />
        </div>

        <!-- Memory -->
        <div>
          <Label for="memory" class="mb-2">Memory (MB)</Label>
          <Input
            id="memory"
            type="number"
            bind:value={memoryMB}
            min="128"
            step="128"
          />
        </div>

        <!-- CPU -->
        <div>
          <Label for="cpu" class="mb-2">CPU Cores</Label>
          <Input
            id="cpu"
            type="number"
            bind:value={cpuCores}
            min="0.1"
            step="0.1"
          />
        </div>
      </div>

      {#if workloadType === 'container'}
        <div>
          <Label for="image" class="mb-2">Container Image *</Label>
          <Input
            id="image"
            type="text"
            bind:value={image}
            required
            placeholder="docker.io/library/alpine:latest"
          />
        </div>
      {/if}

      <!-- Command -->
      <div>
        <Label for="command" class="mb-2">Command</Label>
        <Textarea
          id="command"
          bind:value={command}
          rows="3"
          placeholder="echo 'Hello World'"
        />
      </div>

      <div class="flex gap-4">
        <Button type="submit" disabled={loading}>
          {loading ? 'Creating...' : 'Create Job'}
        </Button>
        <Button color="light" on:click={() => navigate('/jobs')}>
          Cancel
        </Button>
      </div>
    </form>
  </Card>
</div>

