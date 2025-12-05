<script>
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Label, Input, Select, Textarea, Alert } from 'flowbite-svelte';
  import { ArrowLeftOutline } from 'flowbite-svelte-icons';
  import { deployments } from '../api/client';

  let loading = false;
  let error = '';
  let success = '';

  // Form fields
  let deploymentName = '';
  let deploymentType = 'service';
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

    if (!deploymentName) {
      error = 'Deployment name is required';
      loading = false;
      return;
    }

    const deploymentData = {
      deployment_name: deploymentName,
      deployment_type: deploymentType,
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
      const response = await deployments.create(deploymentData);
      success = `Deployment created successfully! Deployment ID: ${response.deployment_id}`;
      setTimeout(() => {
        navigate(`/deployments/${response.deployment_id}`);
      }, 1500);
    } catch (err) {
      error = err.message || 'Failed to create deployment';
    } finally {
      loading = false;
    }
  }
</script>

<div class="space-y-6">
  <div class="flex items-center gap-4">
    <Button color="light" on:click={() => navigate('/deployments')}>
      <ArrowLeftOutline class="w-4 h-4 mr-2" />
      Back
    </Button>
    <Heading tag="h2" class="text-2xl font-bold">Create New Deployment</Heading>
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
        <!-- Deployment Name -->
        <div>
          <Label for="deployment_name" class="mb-2">Deployment Name *</Label>
          <Input
            id="deployment_name"
            type="text"
            bind:value={deploymentName}
            required
            placeholder="my-deployment"
          />
        </div>

        <!-- Deployment Type -->
        <div>
          <Label for="deployment_type" class="mb-2">Deployment Type</Label>
          <Input
            id="deployment_type"
            type="text"
            bind:value={deploymentType}
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
          {loading ? 'Creating...' : 'Create Deployment'}
        </Button>
        <Button color="light" on:click={() => navigate('/deployments')}>
          Cancel
        </Button>
      </div>
    </form>
  </Card>
</div>

