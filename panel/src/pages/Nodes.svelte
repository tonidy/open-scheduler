<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Badge, Table, TableHead, TableHeadCell, TableBody, TableBodyRow, TableBodyCell, Spinner, Button } from 'flowbite-svelte';
  import { nodes } from '../api/client';
  import { formatEventTimestamp } from '../utils/dateFormatter';

  let loading = true;
  let error = null;
  let nodesData = null;

  onMount(async () => {
    await loadNodes();
  });

  async function loadNodes() {
    try {
      const response = await nodes.list();
      nodesData = response.nodes || [];
      loading = false;
      error = null;
    } catch (err) {
      error = err.message;
      loading = false;
    }
  }

  function getHealthBadge(node) {
    // Simple health check based on last heartbeat (within 30 seconds)
    if (!node.last_heartbeat) return 'red';
    const lastHeartbeat = new Date(node.last_heartbeat).getTime();
    const now = Date.now();
    const diffSeconds = (now - lastHeartbeat) / 1000;
    return diffSeconds < 30 ? 'green' : 'red';
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <Heading tag="h2" class="text-2xl font-bold">Nodes</Heading>
    {#if nodesData}
      <Badge color="blue" large>{nodesData.length} Node{nodesData.length !== 1 ? 's' : ''}</Badge>
    {/if}
  </div>

  {#if loading && !nodesData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading nodes: {error}</p>
    </Card>
  {:else if nodesData}
    <Card class="bg-white dark:bg-gray-800 min-w-full">
      {#if nodesData.length > 0}
        <Table>
          <TableHead>
            <TableHeadCell>Node ID</TableHeadCell>
            <TableHeadCell>Status</TableHeadCell>
            <TableHeadCell>CPU Cores</TableHeadCell>
            <TableHeadCell>RAM (MB)</TableHeadCell>
            <TableHeadCell>Disk (MB)</TableHeadCell>
            <TableHeadCell>Last Heartbeat</TableHeadCell>
            <TableHeadCell>Actions</TableHeadCell>
          </TableHead>
          <TableBody>
            {#each nodesData as node}
              <TableBodyRow>
                <TableBodyCell>
                  <span class="font-mono">{node.node_id?.slice(0, 12)}...</span>
                </TableBodyCell>
                <TableBodyCell>
                  <Badge color={getHealthBadge(node)}>
                    {getHealthBadge(node) === 'green' ? 'Healthy' : 'Unhealthy'}
                  </Badge>
                </TableBodyCell>
                <TableBodyCell>{node.cpu_cores || 0}</TableBodyCell>
                <TableBodyCell>{node.ram_mb || 0}</TableBodyCell>
                <TableBodyCell>{node.disk_mb || 0}</TableBodyCell>
                <TableBodyCell>{formatEventTimestamp(node.last_heartbeat)}</TableBodyCell>
                <TableBodyCell>
                  <Button size="xs" on:click={() => navigate(`/nodes/${node.node_id}`)}>View</Button>
                </TableBodyCell>
              </TableBodyRow>
            {/each}
          </TableBody>
        </Table>
      {:else}
        <p class="text-center text-gray-500 dark:text-gray-400 py-8">No nodes registered</p>
      {/if}
    </Card>
  {/if}
</div>

