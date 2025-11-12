<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Badge, Spinner, Timeline, TimelineItem } from 'flowbite-svelte';
  import { ArrowLeftOutline } from 'flowbite-svelte-icons';
  import { jobs } from '../api/client';
  import { formatDate, parseEvent } from '../utils/dateFormatter';

  export let id;

  let loading = true;
  let error = null;
  let jobData = null;
  let events = [];

  onMount(async () => {
    await loadJobDetails();
  });

  async function loadJobDetails() {
    try {
      jobData = await jobs.get(id);
      const eventsResponse = await jobs.getEvents(id);
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
    <Button color="light" on:click={() => navigate('/jobs')}>
      <ArrowLeftOutline class="w-4 h-4 mr-2" />
      Back
    </Button>
    <Heading tag="h2" class="text-2xl font-bold">Job Details</Heading>
  </div>

  {#if loading && !jobData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading job: {error}</p>
    </Card>
  {:else if jobData}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Job Information -->
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Job Information</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Job ID:</span>
            <p class="text-gray-900 dark:text-white font-mono">{jobData.job_id}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Job Name:</span>
            <p class="text-gray-900 dark:text-white">{jobData.job?.job_name || 'N/A'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Status:</span>
            <div class="mt-1">
              <Badge color={getStatusBadge(jobData.status)} large>{jobData.status || 'Unknown'}</Badge>
            </div>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Node ID:</span>
            <p class="text-gray-900 dark:text-white font-mono">{jobData.node_id || 'Not assigned'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Driver:</span>
            <p class="text-gray-900 dark:text-white">{jobData.job?.driver_type || 'N/A'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Workload Type:</span>
            <p class="text-gray-900 dark:text-white">{jobData.job?.workload_type || 'N/A'}</p>
          </div>
          {#if jobData.job?.command}
            <div>
              <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Command:</span>
              <p class="text-gray-900 dark:text-white font-mono text-sm">{jobData.job.command}</p>
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
            <p class="text-gray-900 dark:text-white">{jobData.detail || 'No details available'}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Claimed At:</span>
            <p class="text-gray-900 dark:text-white">{formatDate(jobData.claimed_at)}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Updated At:</span>
            <p class="text-gray-900 dark:text-white">{formatDate(jobData.updated_at)}</p>
          </div>
        </div>
      </Card>
    </div>

    <!-- Instance Config -->
    {#if jobData.job?.instance_config}
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Instance Configuration</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Image:</span>
            <p class="text-gray-900 dark:text-white font-mono">{jobData.job.instance_config.image_name || 'N/A'}</p>
          </div>
          {#if jobData.job.instance_config.entrypoint && jobData.job.instance_config.entrypoint.length > 0}
            <div>
              <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Entrypoint:</span>
              <p class="text-gray-900 dark:text-white font-mono text-sm">{jobData.job.instance_config.entrypoint.join(' ')}</p>
            </div>
          {/if}
          {#if jobData.job.instance_config.arguments && jobData.job.instance_config.arguments.length > 0}
            <div>
              <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Arguments:</span>
              <p class="text-gray-900 dark:text-white font-mono text-sm">{jobData.job.instance_config.arguments.join(' ')}</p>
            </div>
          {/if}
        </div>
      </Card>
    {/if}

    <!-- Resource Requirements -->
    {#if jobData.job?.resource_requirements}
      <Card>
        <Heading tag="h3" class="text-xl font-semibold mb-4">Resource Requirements</Heading>
        <div class="grid grid-cols-2 gap-4">
          <div class="p-3 bg-gray-50 dark:bg-gray-800 rounded">
            <span class="text-sm text-gray-500 dark:text-gray-400">CPU Limit</span>
            <p class="text-lg font-semibold text-gray-900 dark:text-white">
              {jobData.job.resource_requirements.cpu_limit_cores || 0} cores
            </p>
          </div>
          <div class="p-3 bg-gray-50 dark:bg-gray-800 rounded">
            <span class="text-sm text-gray-500 dark:text-gray-400">Memory Limit</span>
            <p class="text-lg font-semibold text-gray-900 dark:text-white">
              {jobData.job.resource_requirements.memory_limit_mb || 0} MB
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

