<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Badge, Table, TableHead, TableHeadCell, TableBody, TableBodyRow, TableBodyCell, Spinner, Tabs, TabItem } from 'flowbite-svelte';
  import { PlusOutline } from 'flowbite-svelte-icons';
  import { deployments } from '../api/client';
  import { formatShortDate } from '../utils/dateFormatter';

  let loading = true;
  let error = null;
  let deploymentsData = null;
  let activeTab = 'all';

  onMount(async () => {
    await loadDeployments();
  });

  async function loadDeployments(status = '') {
    try {
      deploymentsData = await deployments.list(status);
      loading = false;
      error = null;
    } catch (err) {
      error = err.message;
      loading = false;
    }
  }

  function handleTabChange(tabName) {
    activeTab = tabName;
    const statusMap = { all: '', queued: 'queued', active: 'active', completed: 'completed' };
    loadDeployments(statusMap[tabName]);
  }

  function getStatusBadge(status) {
    const statusColors = {
      queued: 'yellow',
      running: 'blue',
      completed: 'green',
      failed: 'red',
      failed_retrying: 'red',
      pending: 'gray',
    };
    return statusColors[status?.toLowerCase()] || 'gray';
  }
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <Heading tag="h2" class="text-2xl font-bold">Deployments</Heading>
    <Button on:click={() => navigate('/deployments/create')}>
      <PlusOutline class="w-4 h-4 mr-2" />
      Create Deployment
    </Button>
  </div>

  {#if loading && !deploymentsData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading deployments: {error}</p>
    </Card>
  {:else if deploymentsData}
    <Card class="bg-white dark:bg-gray-800 min-w-full">
      <Tabs style="underline">
        <TabItem open={activeTab === 'all'} title="All Deployments" on:click={() => handleTabChange('all')}>
          <div class="space-y-4 pt-4">
            {#if deploymentsData.queued_count !== undefined}
              <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div class="p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg">
                  <p class="text-sm text-yellow-800 dark:text-yellow-200">Queued</p>
                  <p class="text-2xl font-bold text-yellow-900 dark:text-yellow-100">{deploymentsData.queued_count || 0}</p>
                </div>
                <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                  <p class="text-sm text-blue-800 dark:text-blue-200">Active</p>
                  <p class="text-2xl font-bold text-blue-900 dark:text-blue-100">{deploymentsData.active_deployments?.length || 0}</p>
                </div>
                <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                  <p class="text-sm text-green-800 dark:text-green-200">Completed</p>
                  <p class="text-2xl font-bold text-green-900 dark:text-green-100">{deploymentsData.completed_count || 0}</p>
                </div>
                <div class="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
                  <p class="text-sm text-red-800 dark:text-red-200">Failed</p>
                  <p class="text-2xl font-bold text-red-900 dark:text-red-100">{deploymentsData.failed_count || 0}</p>
                </div>
              </div>
            {/if}

            {#if deploymentsData.queued_deployments && deploymentsData.queued_deployments.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-yellow-600 dark:text-yellow-400">Queued Deployments</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Deployment ID</TableHeadCell>
                    <TableHeadCell>Deployment Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each deploymentsData.queued_deployments as deployment}
                      <TableBodyRow>
                        <TableBodyCell>{deployment.deployment_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{deployment.deployment?.deployment_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(deployment.status)}>{deployment.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{deployment.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(deployment.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/deployments/${deployment.deployment_id}`)}>View</Button>
                        </TableBodyCell>
                      </TableBodyRow>
                    {/each}
                  </TableBody>
                </Table>
              </div>
            {/if}

            {#if deploymentsData.active_deployments && deploymentsData.active_deployments.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-blue-600 dark:text-blue-400">Active Deployments</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Deployment ID</TableHeadCell>
                    <TableHeadCell>Deployment Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Node ID</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Claimed At</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each deploymentsData.active_deployments as deployment}
                      <TableBodyRow>
                        <TableBodyCell>{deployment.deployment_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{deployment.deployment?.deployment_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(deployment.status)}>{deployment.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell>{deployment.node_id ? deployment.node_id.slice(0, 20) : 'N/A'}</TableBodyCell>
                        <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{deployment.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(deployment.claimed_at)}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(deployment.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/deployments/${deployment.deployment_id}`)}>View</Button>
                        </TableBodyCell>
                      </TableBodyRow>
                    {/each}
                  </TableBody>
                </Table>
              </div>
            {/if}

            {#if deploymentsData.failed_deployments && deploymentsData.failed_deployments.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-red-600 dark:text-red-400">Failed Deployments</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Deployment ID</TableHeadCell>
                    <TableHeadCell>Deployment Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Node ID</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each deploymentsData.failed_deployments as deployment}
                      <TableBodyRow>
                        <TableBodyCell>{deployment.deployment_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{deployment.deployment?.deployment_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(deployment.status)}>{deployment.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell>{deployment.node_id || 'N/A'}</TableBodyCell>
                        <TableBodyCell class="text-sm text-red-600 dark:text-red-400">{deployment.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(deployment.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/deployments/${deployment.deployment_id}`)}>View</Button>
                        </TableBodyCell>
                      </TableBodyRow>
                    {/each}
                  </TableBody>
                </Table>
              </div>
            {/if}

            {#if deploymentsData.completed_deployments && deploymentsData.completed_deployments.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-green-600 dark:text-green-400">Completed Deployments</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Deployment ID</TableHeadCell>
                    <TableHeadCell>Deployment Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Node ID</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each deploymentsData.completed_deployments.slice(0, 10) as deployment}
                      <TableBodyRow>
                        <TableBodyCell>{deployment.deployment_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{deployment.deployment?.deployment_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(deployment.status)}>{deployment.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell>{deployment.node_id?.slice(0, 20) || 'N/A'}</TableBodyCell>
                        <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{deployment.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(deployment.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/deployments/${deployment.deployment_id}`)}>View</Button>
                        </TableBodyCell>
                      </TableBodyRow>
                    {/each}
                  </TableBody>
                </Table>
              </div>
            {/if}
          </div>
        </TabItem>

        <TabItem open={activeTab === 'active'} title="Active" on:click={() => handleTabChange('active')}>
          <div class="pt-4">
            {#if deploymentsData.active_deployments && deploymentsData.active_deployments.length > 0}
              <Table>
                <TableHead>
                  <TableHeadCell>Deployment ID</TableHeadCell>
                  <TableHeadCell>Deployment Name</TableHeadCell>
                  <TableHeadCell>Status</TableHeadCell>
                  <TableHeadCell>Node ID</TableHeadCell>
                  <TableHeadCell>Detail</TableHeadCell>
                  <TableHeadCell>Claimed At</TableHeadCell>
                  <TableHeadCell>Updated At</TableHeadCell>
                  <TableHeadCell>Actions</TableHeadCell>
                </TableHead>
                <TableBody>
                  {#each deploymentsData.active_deployments as deployment}
                    <TableBodyRow>
                      <TableBodyCell>{deployment.deployment_id?.slice(0, 8)}...</TableBodyCell>
                      <TableBodyCell>{deployment.deployment?.deployment_name || 'N/A'}</TableBodyCell>
                      <TableBodyCell>
                        <Badge color={getStatusBadge(deployment.status)}>{deployment.status || 'Unknown'}</Badge>
                      </TableBodyCell>
                      <TableBodyCell>{deployment.node_id ? deployment.node_id.slice(0, 20) : 'N/A'}</TableBodyCell>
                      <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{deployment.detail || '-'}</TableBodyCell>
                      <TableBodyCell>{formatShortDate(deployment.claimed_at)}</TableBodyCell>
                      <TableBodyCell>{formatShortDate(deployment.updated_at)}</TableBodyCell>
                      <TableBodyCell>
                        <Button size="xs" on:click={() => navigate(`/deployments/${deployment.deployment_id}`)}>View</Button>
                      </TableBodyCell>
                    </TableBodyRow>
                  {/each}
                </TableBody>
              </Table>
            {:else}
              <p class="text-center text-gray-500 dark:text-gray-400 py-8">No active deployments</p>
            {/if}
          </div>
        </TabItem>

        <TabItem open={activeTab === 'completed'} title="Completed" on:click={() => handleTabChange('completed')}>
          <div class="pt-4">
            {#if deploymentsData.completed_deployments && deploymentsData.completed_deployments.length > 0}
              <Table>
                <TableHead>
                  <TableHeadCell>Deployment ID</TableHeadCell>
                  <TableHeadCell>Deployment Name</TableHeadCell>
                  <TableHeadCell>Status</TableHeadCell>
                  <TableHeadCell>Node ID</TableHeadCell>
                  <TableHeadCell>Updated At</TableHeadCell>
                  <TableHeadCell>Actions</TableHeadCell>
                </TableHead>
                <TableBody>
                  {#each deploymentsData.completed_deployments as deployment}
                    <TableBodyRow>
                      <TableBodyCell>{deployment.deployment_id?.slice(0, 8)}...</TableBodyCell>
                      <TableBodyCell>{deployment.deployment?.deployment_name || 'N/A'}</TableBodyCell>
                      <TableBodyCell>
                        <Badge color={getStatusBadge(deployment.status)}>{deployment.status || 'Unknown'}</Badge>
                      </TableBodyCell>
                      <TableBodyCell>{deployment.node_id?.slice(0, 8) || 'N/A'}</TableBodyCell>
                      <TableBodyCell>{formatShortDate(deployment.updated_at)}</TableBodyCell>
                      <TableBodyCell>
                        <Button size="xs" on:click={() => navigate(`/deployments/${deployment.deployment_id}`)}>View</Button>
                      </TableBodyCell>
                    </TableBodyRow>
                  {/each}
                </TableBody>
              </Table>
            {:else}
              <p class="text-center text-gray-500 dark:text-gray-400 py-8">No completed deployments</p>
            {/if}
          </div>
        </TabItem>
      </Tabs>
    </Card>
  {/if}
</div>

