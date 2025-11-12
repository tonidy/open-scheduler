<script>
  import { onMount } from 'svelte';
  import { Card, Heading, Spinner, Badge } from 'flowbite-svelte';
  import { ChartPieSolid, BriefcaseSolid, ServerSolid, CheckCircleSolid } from 'flowbite-svelte-icons';
  import { stats } from '../api/client';

  let loading = true;
  let error = null;
  let statsData = null;

  onMount(async () => {
    await loadStats();
  });

  async function loadStats() {
    try {
      statsData = await stats.get();
      loading = false;
      error = null;
    } catch (err) {
      error = err.message;
      loading = false;
    }
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <Heading tag="h2" class="text-2xl font-bold">Dashboard</Heading>
    {#if !loading}
      <Badge color="green" class="animate-pulse">Live</Badge>
    {/if}
  </div>

  {#if loading && !statsData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading stats: {error}</p>
    </Card>
  {:else if statsData}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      <!-- Total Nodes -->
      <Card class="hover:shadow-lg transition-shadow">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">Total Nodes</p>
            <p class="text-3xl font-bold text-gray-900 dark:text-white mt-1">
              {statsData.nodes?.total || 0}
            </p>
          </div>
          <div class="p-3 bg-blue-100 dark:bg-blue-900 rounded-lg">
            <ServerSolid class="w-8 h-8 text-blue-600 dark:text-blue-300" />
          </div>
        </div>
      </Card>

      <!-- Healthy Nodes -->
      <Card class="hover:shadow-lg transition-shadow">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">Healthy Nodes</p>
            <p class="text-3xl font-bold text-green-600 dark:text-green-400 mt-1">
              {statsData.nodes?.healthy || 0}
            </p>
          </div>
          <div class="p-3 bg-green-100 dark:bg-green-900 rounded-lg">
            <CheckCircleSolid class="w-8 h-8 text-green-600 dark:text-green-300" />
          </div>
        </div>
      </Card>

      <!-- Queued Jobs -->
      <Card class="hover:shadow-lg transition-shadow">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">Queued Jobs</p>
            <p class="text-3xl font-bold text-yellow-600 dark:text-yellow-400 mt-1">
              {statsData.jobs?.queued || 0}
            </p>
          </div>
          <div class="p-3 bg-yellow-100 dark:bg-yellow-900 rounded-lg">
            <ChartPieSolid class="w-8 h-8 text-yellow-600 dark:text-yellow-300" />
          </div>
        </div>
      </Card>

      <!-- Active Jobs -->
      <Card class="hover:shadow-lg transition-shadow">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">Active Jobs</p>
            <p class="text-3xl font-bold text-purple-600 dark:text-purple-400 mt-1">
              {statsData.jobs?.active || 0}
            </p>
          </div>
          <div class="p-3 bg-purple-100 dark:bg-purple-900 rounded-lg">
            <BriefcaseSolid class="w-8 h-8 text-purple-600 dark:text-purple-300" />
          </div>
        </div>
      </Card>
    </div>

    <!-- Jobs Summary -->
    <Card>
      <Heading tag="h3" class="text-xl font-semibold mb-4">Jobs Summary</Heading>
      <div class="space-y-4">
        <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <span class="text-gray-700 dark:text-gray-300">Queued Jobs</span>
          <Badge color="yellow" large>{statsData.jobs?.queued || 0}</Badge>
        </div>
        <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <span class="text-gray-700 dark:text-gray-300">Active Jobs</span>
          <Badge color="purple" large>{statsData.jobs?.active || 0}</Badge>
        </div>
        <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <span class="text-gray-700 dark:text-gray-300">Completed Jobs</span>
          <Badge color="green" large>{statsData.jobs?.completed || 0}</Badge>
        </div>
      </div>
    </Card>

    <!-- Cluster Health -->
    <Card>
      <Heading tag="h3" class="text-xl font-semibold mb-4">Cluster Health</Heading>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <span class="text-gray-700 dark:text-gray-300">Node Health</span>
          <div class="flex items-center gap-2">
            <span class="text-lg font-semibold text-green-600 dark:text-green-400">
              {statsData.nodes?.healthy || 0} / {statsData.nodes?.total || 0}
            </span>
            {#if statsData.nodes?.healthy === statsData.nodes?.total}
              <Badge color="green">All Healthy</Badge>
            {:else}
              <Badge color="red">Issues Detected</Badge>
            {/if}
          </div>
        </div>
        
        <div class="w-full bg-gray-200 rounded-full h-4 dark:bg-gray-700">
          <div
            class="bg-green-600 h-4 rounded-full transition-all duration-300"
            style="width: {statsData.nodes?.total > 0 ? (statsData.nodes?.healthy / statsData.nodes?.total * 100) : 0}%"
          ></div>
        </div>
      </div>
    </Card>
  {/if}
</div>

