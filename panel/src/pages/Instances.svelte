<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Badge, Table, TableHead, TableHeadCell, TableBody, TableBodyRow, TableBodyCell, Spinner, Button } from 'flowbite-svelte';
  import { instances } from '../api/client';
  import { formatShortDate } from '../utils/dateFormatter';

  let loading = true;
  let error = null;
  let instancesData = null;

  onMount(async () => {
    await loadInstances();
  });

  async function loadInstances() {
    try {
      const response = await instances.list();
      instancesData = response.instances || [];
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
  <div class="flex items-center justify-between">
    <Heading tag="h2" class="text-2xl font-bold">Instances</Heading>
    {#if instancesData}
      <Badge color="blue" large>{instancesData.length} Instance{instancesData.length !== 1 ? 's' : ''}</Badge>
    {/if}
  </div>

  {#if loading && !instancesData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading instances: {error}</p>
    </Card>
  {:else if instancesData}
    <Card class="bg-white dark:bg-gray-800 min-w-full">
      {#if instancesData.length > 0}
        <Table>
          <TableHead>
            <TableHeadCell>Instance Name</TableHeadCell>
            <TableHeadCell>Status</TableHeadCell>
            <TableHeadCell>Created At</TableHeadCell>
            <TableHeadCell>Actions</TableHeadCell>
          </TableHead>
          <TableBody>
            {#each instancesData as instance}
              <TableBodyRow>
                <TableBodyCell>
                  <span class="font-mono">{instance.instance_name || 'N/A'}</span>
                </TableBodyCell>
                <TableBodyCell>
                  <Badge color={getStatusBadge(instance.status)}>
                    {instance.status || 'Unknown'}
                  </Badge>
                </TableBodyCell>
                <TableBodyCell>{formatShortDate(instance.created)}</TableBodyCell>
                <TableBodyCell>
                  {#if instance.instance_name}
                    <Button size="xs" on:click={() => navigate(`/instances/${instance.job_id}`)}>View</Button>
                  {:else}
                    <span class="text-gray-400">N/A</span>
                  {/if}
                </TableBodyCell>
              </TableBodyRow>
            {/each}
          </TableBody>
        </Table>
      {:else}
        <p class="text-center text-gray-500 dark:text-gray-400 py-8">No instances running</p>
      {/if}
    </Card>
  {/if}
</div>

