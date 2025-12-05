<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Badge, Spinner, Timeline, TimelineItem } from 'flowbite-svelte';
  import { ArrowLeftOutline } from 'flowbite-svelte-icons';
  import { deployments } from '../api/client';
  import { formatDate, parseEvent } from '../utils/dateFormatter';

  export let id;

  let loading = true;
  let error = null;
  let deploymentData = null;
  let events = [];

  onMount(async () => {
    await loadDeploymentDetails();
  });

  async function loadDeploymentDetails() {
    try {
      deploymentData = await deployments.get(id);
      const eventsResponse = await deployments.getEvents(id);
      events = eventsResponse.events || [];
      loading = false;
      error = null;
    } catch (err) {
      error = err.message;
      loading = false;
    }
  }

  function getStatusBadge(status) {
    const statusColors = {
      queued: 'yellow',
      running: 'blue',
      completed: 'green',
      failed: 'red',
      pending: 'gray',
    };
    return statusColors[status?.toLowerCase()] || 'gray';
  }
</script>

<div class="space-y-6">
  <div class="flex items-center gap-4">
    <Button color="light" on:click={() => navigate('/deployments')}>
      <ArrowLeftOutline class="w-4 h-4 mr-2" />
      Back
    </Button>
    <Heading tag="h2" class="text-2xl font-bold">Deployment Details</Heading>
  </div>

  {#if loading && !deploymentData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading deployment: {error}</p>
    </Card>
  {:else if deploymentData}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Deployment Information -->
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Deployment Information</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Deployment ID:</span>
            <p class="text-gray-900 dark:text-white font-mono">{deploymentData.deployment_id}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Deployment Name:</span>
            <p class="text-gray-900 dark:text-white">{deploymentData.deployment?.deployment_name || 'N/A'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Status:</span>
            <div class="mt-1">
              <Badge color={getStatusBadge(deploymentData.status)} large>{deploymentData.status || 'Unknown'}</Badge>
            </div>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Node ID:</span>
            <p class="text-gray-900 dark:text-white font-mono">{deploymentData.node_id || 'Not assigned'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Driver:</span>
            <p class="text-gray-900 dark:text-white">{deploymentData.deployment?.driver_type || 'N/A'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Workload Type:</span>
            <p class="text-gray-900 dark:text-white">{deploymentData.deployment?.workload_type || 'N/A'}</p>
          </div>
          {#if deploymentData.deployment?.command}
            <div>
              <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Command:</span>
              <p class="text-gray-900 dark:text-white font-mono text-sm">{deploymentData.deployment.command}</p>
            </div>
          {/if}
        </div>
      </Card>

      <!-- Status & Timestamps -->
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Status & Timestamps</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Detail:</span>
            <p class="text-gray-900 dark:text-white">{deploymentData.detail || 'No details available'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Claimed At:</span>
            <p class="text-gray-900 dark:text-white">{formatDate(deploymentData.claimed_at)}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Updated At:</span>
            <p class="text-gray-900 dark:text-white">{formatDate(deploymentData.updated_at)}</p>
          </div>
        </div>
      </Card>
    </div>

    <!-- Instance Config -->
    {#if deploymentData.deployment?.instance_config}
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Instance Configuration</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Image:</span>
            <p class="text-gray-900 dark:text-white font-mono">{deploymentData.deployment.instance_config.image_name || 'N/A'}</p>
          </div>
          {#if deploymentData.deployment.instance_config.entrypoint && deploymentData.deployment.instance_config.entrypoint.length > 0}
            <div>
              <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Entrypoint:</span>
              <p class="text-gray-900 dark:text-white font-mono text-sm">{deploymentData.deployment.instance_config.entrypoint.join(' ')}</p>
            </div>
          {/if}
          {#if deploymentData.deployment.instance_config.arguments && deploymentData.deployment.instance_config.arguments.length > 0}
            <div>
              <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Arguments:</span>
              <p class="text-gray-900 dark:text-white font-mono text-sm">{deploymentData.deployment.instance_config.arguments.join(' ')}</p>
            </div>
          {/if}
        </div>
      </Card>
    {/if}

    <!-- Resource Requirements -->
    {#if deploymentData.deployment?.resource_requirements}
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Resource Requirements</Heading>
        <div class="grid grid-cols-2 gap-4">
          <div class="p-3 bg-gray-50 dark:bg-gray-800 rounded">
            <span class="text-sm text-gray-500 dark:text-gray-400">CPU Limit</span>
            <p class="text-lg font-semibold text-gray-900 dark:text-white">
              {deploymentData.deployment.resource_requirements.cpu_limit_cores || 0} cores
            </p>
          </div>
          <div class="p-3 bg-gray-50 dark:bg-gray-800 rounded">
            <span class="text-sm text-gray-500 dark:text-gray-400">Memory Limit</span>
            <p class="text-lg font-semibold text-gray-900 dark:text-white">
              {deploymentData.deployment.resource_requirements.memory_limit_mb || 0} MB
            </p>
          </div>
        </div>
      </Card>
    {/if}

    <!-- Events Timeline -->
    <Card>
      <Heading tag="h3" class="text-xl font-semibold mb-4">Event Timeline</Heading>
      {#if events.length > 0}
        <Timeline>
          {#each events as event}
            {@const parsed = parseEvent(event)}
            <TimelineItem>
              <div class="space-y-1">
                <p class="text-sm text-gray-900 dark:text-white">{parsed.message}</p>
                {#if parsed.formattedTime}
                  <p class="text-xs text-gray-500 dark:text-gray-400">{parsed.formattedTime}</p>
                {/if}
              </div>
            </TimelineItem>
          {/each}
        </Timeline>
      {:else}
        <p class="text-center text-gray-500 dark:text-gray-400 py-4">No events recorded</p>
      {/if}
    </Card>
  {/if}
</div>

