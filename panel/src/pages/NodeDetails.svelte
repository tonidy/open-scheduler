<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Badge, Spinner } from 'flowbite-svelte';
  import { ArrowLeftOutline } from 'flowbite-svelte-icons';
  import { nodes } from '../api/client';
  import { formatEventTimestamp } from '../utils/dateFormatter';

  export let id;

  let loading = true;
  let error = null;
  let nodeData = null;
  let healthData = null;

  onMount(async () => {
    await loadNodeDetails();
  });

  async function loadNodeDetails() {
    try {
      nodeData = await nodes.get(id);
      healthData = await nodes.getHealth(id);
      loading = false;
      error = null;
    } catch (err) {
      error = err.message;
      loading = false;
    }
  }
</script>

<div class="space-y-6">
  <div class="flex items-center gap-4">
    <Button color="light" on:click={() => navigate('/nodes')}>
      <ArrowLeftOutline class="w-4 h-4 mr-2" />
      Back
    </Button>
    <Heading tag="h2" class="text-2xl font-bold">Node Details</Heading>
  </div>

  {#if loading && !nodeData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading node: {error}</p>
    </Card>
  {:else if nodeData}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Node Information -->
      <Card class="bg-white dark:bg-gray-800 min-w-full">
        <Heading tag="h3" class="text-xl font-semibold mb-4">Node Information</Heading>
        <div class="space-y-3">
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Node ID:</span>
            <p class="text-gray-900 dark:text-white font-mono break-all">{nodeData.node_id}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Status:</span>
            <div class="mt-1">
              {#if healthData}
                <Badge color={healthData.healthy ? 'green' : 'red'} large>
                  {healthData.status}
                </Badge>
              {/if}
            </div>
          </div>
          <div>
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Last Heartbeat:</span>
            <p class="text-gray-900 dark:text-white">{formatEventTimestamp(nodeData.last_heartbeat)}</p>
          </div>
        </div>
      </Card>

      <!-- Health Status -->
      {#if healthData}
        <Card class="bg-white dark:bg-gray-800 min-w-full">
          <Heading tag="h3" class="text-xl font-semibold mb-4">Health Status</Heading>
          <div class="space-y-3">
            <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <span class="text-gray-700 dark:text-gray-300">Health Check</span>
              <Badge color={healthData.healthy ? 'green' : 'red'}>
                {healthData.healthy ? 'Passing' : 'Failing'}
              </Badge>
            </div>
            <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <span class="text-gray-700 dark:text-gray-300">Last Update</span>
              <span class="text-gray-900 dark:text-white">{formatEventTimestamp(healthData.last_heartbeat)}</span>
            </div>
          </div>
        </Card>
      {/if}
    </div>

    <!-- Resources -->
    <Card class="bg-white dark:bg-gray-800 min-w-full">
      <Heading tag="h3" class="text-xl font-semibold mb-4">Resources</Heading>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <p class="text-sm text-blue-800 dark:text-blue-200">CPU Cores</p>
          <p class="text-3xl font-bold text-blue-900 dark:text-blue-100">{nodeData.cpu_cores || 0}</p>
        </div>
        <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
          <p class="text-sm text-green-800 dark:text-green-200">RAM</p>
          <p class="text-3xl font-bold text-green-900 dark:text-green-100">{nodeData.ram_mb || 0} <span class="text-lg">MB</span></p>
        </div>
        <div class="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
          <p class="text-sm text-purple-800 dark:text-purple-200">Disk</p>
          <p class="text-3xl font-bold text-purple-900 dark:text-purple-100">{nodeData.disk_mb || 0} <span class="text-lg">MB</span></p>
        </div>
      </div>
    </Card>

    <!-- Metadata -->
    {#if nodeData.metadata && Object.keys(nodeData.metadata).length > 0}
      <Card class="bg-white dark:bg-gray-800 min-w-full">
        <Heading tag="h3" class="text-xl font-semibold mb-4">Metadata</Heading>
        <div class="space-y-2">
          {#each Object.entries(nodeData.metadata) as [key, value]}
            <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded">
              <span class="text-sm font-medium text-gray-600 dark:text-gray-400">{key}:</span>
              <span class="text-sm text-gray-900 dark:text-white">{value}</span>
            </div>
          {/each}
        </div>
      </Card>
    {/if}
  {/if}
</div>

