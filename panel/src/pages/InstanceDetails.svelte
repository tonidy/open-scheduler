<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Badge, Spinner, Timeline, TimelineItem } from 'flowbite-svelte';
  import { ArrowLeftOutline } from 'flowbite-svelte-icons';
  import { instances } from '../api/client';
  import { formatDate, parseEvent } from '../utils/dateFormatter';

  export let id;

  let loading = true;
  let error = null;
  let instanceData = null;
  let events = [];

  onMount(async () => {
    await loadInstanceDetails();
  });

  async function loadInstanceDetails() {
    try {
      const response = await instances.get(id);
      instanceData = response.instance_data;
      events = response.events || [];
      loading = false;
      error = null;
    } catch (err) {
      error = err.message;
      loading = false;
    }
  }

  function getStatusBadge(status) {
    const statusColors = {
      running: 'green',
      stopped: 'red',
      starting: 'yellow',
      stopping: 'yellow',
    };
    return statusColors[status?.toLowerCase()] || 'gray';
  }
</script>

<div class="space-y-6">
  <div class="flex items-center gap-4">
    <Button color="light" on:click={() => navigate('/instances')}>
      <ArrowLeftOutline class="w-4 h-4 mr-2" />
      Back
    </Button>
    <Heading tag="h2" class="text-2xl font-bold">Instance Details</Heading>
  </div>

  {#if loading && !instanceData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading instance: {error}</p>
    </Card>
  {:else if instanceData}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Instance Information -->
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Instance Information</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Instance Name:</span>
            <p class="text-gray-900 dark:text-white font-mono">{instanceData.instance_name || 'N/A'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Instance ID:</span>
            <p class="text-gray-900 dark:text-white font-mono text-xs break-all">{instanceData.instance_id || 'N/A'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Status:</span>
            <div class="mt-1">
              <Badge color={getStatusBadge(instanceData.status)} large>
                {instanceData.status || 'Unknown'}
              </Badge>
            </div>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">PID:</span>
            <p class="text-gray-900 dark:text-white">{instanceData.pid || 'N/A'}</p>
          </div>
        </div>
      </Card>

      <!-- Timestamps -->
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Timestamps</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Created:</span>
            <p class="text-gray-900 dark:text-white">{formatDate(instanceData.created)}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Started At:</span>
            <p class="text-gray-900 dark:text-white">{formatDate(instanceData.started_at)}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Finished At:</span>
            <p class="text-gray-900 dark:text-white">{formatDate(instanceData.finished_at)}</p>
          </div>
        </div>
      </Card>
    </div>

    <!-- Image Information -->
    <Card>
      <Heading tag="h3" class="text-xl font-semibold mb-4">Image Information</Heading>
      <div class="space-y-3">
        <div>
          <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Image Name:</span>
          <p class="text-gray-900 dark:text-white font-mono">{instanceData.image_name || 'N/A'}</p>
        </div>
        <div>
          <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Image ID:</span>
          <p class="text-gray-900 dark:text-white font-mono text-xs break-all">{instanceData.image || 'N/A'}</p>
        </div>
      </div>
    </Card>

    <!-- Command Information -->
    <Card>
      <Heading tag="h3" class="text-xl font-semibold mb-4">Command</Heading>
      <div class="space-y-3">
        {#if instanceData.command && instanceData.command.length > 0}
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Command:</span>
            <p class="text-gray-900 dark:text-white font-mono bg-gray-50 dark:bg-gray-800 p-3 rounded mt-1">
              {instanceData.command.join(' ')}
            </p>
          </div>
        {/if}
        {#if instanceData.args && instanceData.args.length > 0}
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Arguments:</span>
            <p class="text-gray-900 dark:text-white font-mono bg-gray-50 dark:bg-gray-800 p-3 rounded mt-1">
              {instanceData.args.join(' ')}
            </p>
          </div>
        {/if}
      </div>
    </Card>

    <!-- Exit Status -->
    {#if instanceData.exit_code !== undefined && instanceData.exit_code !== null}
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Exit Status</Heading>
        <div class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div class="flex items-center justify-between">
            <span class="text-gray-700 dark:text-gray-300">Exit Code:</span>
            <Badge color={instanceData.exit_code === 0 ? 'green' : 'red'} large>
              {instanceData.exit_code}
            </Badge>
          </div>
        </div>
      </Card>
    {/if}

    <!-- Labels -->
    {#if instanceData.labels && Object.keys(instanceData.labels).length > 0}
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Labels</Heading>
        <div class="space-y-2">
          {#each Object.entries(instanceData.labels) as [key, value]}
            <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded">
              <span class="text-sm font-medium text-gray-600 dark:text-gray-400">{key}:</span>
              <span class="text-sm text-gray-900 dark:text-white font-mono">{value}</span>
            </div>
          {/each}
        </div>
      </Card>
    {/if}

    <!-- Events -->
    <Card>
      <Heading tag="h3" class="text-xl font-semibold mb-4">Events</Heading>
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

