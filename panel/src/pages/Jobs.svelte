<script>
  import { onMount } from 'svelte';
  import { navigate } from 'svelte-routing';
  import { Card, Heading, Button, Badge, Table, TableHead, TableHeadCell, TableBody, TableBodyRow, TableBodyCell, Spinner, Tabs, TabItem } from 'flowbite-svelte';
  import { PlusOutline } from 'flowbite-svelte-icons';
  import { jobs } from '../api/client';
  import { formatShortDate } from '../utils/dateFormatter';

  let loading = true;
  let error = null;
  let jobsData = null;
  let activeTab = 'all';

  onMount(async () => {
    await loadJobs();
  });

  async function loadJobs(status = '') {
    try {
      jobsData = await jobs.list(status);
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
    loadJobs(statusMap[tabName]);
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
    <Heading tag="h2" class="text-2xl font-bold">Jobs</Heading>
    <Button on:click={() => navigate('/jobs/create')}>
      <PlusOutline class="w-4 h-4 mr-2" />
      Create Job
    </Button>
  </div>

  {#if loading && !jobsData}
    <div class="flex justify-center items-center h-64">
      <Spinner size="12" />
    </div>
  {:else if error}
    <Card class="bg-red-50 dark:bg-red-900/20">
      <p class="text-red-800 dark:text-red-200">Error loading jobs: {error}</p>
    </Card>
  {:else if jobsData}
    <Card class="bg-white dark:bg-gray-800 min-w-full">
      <Tabs style="underline">
        <TabItem open={activeTab === 'all'} title="All Jobs" on:click={() => handleTabChange('all')}>
          <div class="space-y-4 pt-4">
            {#if jobsData.queued_count !== undefined}
              <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div class="p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg">
                  <p class="text-sm text-yellow-800 dark:text-yellow-200">Queued</p>
                  <p class="text-2xl font-bold text-yellow-900 dark:text-yellow-100">{jobsData.queued_count || 0}</p>
                </div>
                <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                  <p class="text-sm text-blue-800 dark:text-blue-200">Active</p>
                  <p class="text-2xl font-bold text-blue-900 dark:text-blue-100">{jobsData.active_jobs?.length || 0}</p>
                </div>
                <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                  <p class="text-sm text-green-800 dark:text-green-200">Completed</p>
                  <p class="text-2xl font-bold text-green-900 dark:text-green-100">{jobsData.completed_count || 0}</p>
                </div>
                <div class="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
                  <p class="text-sm text-red-800 dark:text-red-200">Failed</p>
                  <p class="text-2xl font-bold text-red-900 dark:text-red-100">{jobsData.failed_count || 0}</p>
                </div>
              </div>
            {/if}

            {#if jobsData.queued_jobs && jobsData.queued_jobs.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-yellow-600 dark:text-yellow-400">Queued Jobs</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Job ID</TableHeadCell>
                    <TableHeadCell>Job Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each jobsData.queued_jobs as job}
                      <TableBodyRow>
                        <TableBodyCell>{job.job_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{job.job?.job_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(job.status)}>{job.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{job.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(job.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/jobs/${job.job_id}`)}>View</Button>
                        </TableBodyCell>
                      </TableBodyRow>
                    {/each}
                  </TableBody>
                </Table>
              </div>
            {/if}

            {#if jobsData.active_jobs && jobsData.active_jobs.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-blue-600 dark:text-blue-400">Active Jobs</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Job ID</TableHeadCell>
                    <TableHeadCell>Job Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Node ID</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Claimed At</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each jobsData.active_jobs as job}
                      <TableBodyRow>
                        <TableBodyCell>{job.job_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{job.job?.job_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(job.status)}>{job.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell>{job.node_id ? job.node_id.slice(0, 20) : 'N/A'}</TableBodyCell>
                        <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{job.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(job.claimed_at)}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(job.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/jobs/${job.job_id}`)}>View</Button>
                        </TableBodyCell>
                      </TableBodyRow>
                    {/each}
                  </TableBody>
                </Table>
              </div>
            {/if}

            {#if jobsData.failed_jobs && jobsData.failed_jobs.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-red-600 dark:text-red-400">Failed Jobs</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Job ID</TableHeadCell>
                    <TableHeadCell>Job Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Node ID</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each jobsData.failed_jobs as job}
                      <TableBodyRow>
                        <TableBodyCell>{job.job_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{job.job?.job_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(job.status)}>{job.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell>{job.node_id || 'N/A'}</TableBodyCell>
                        <TableBodyCell class="text-sm text-red-600 dark:text-red-400">{job.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(job.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/jobs/${job.job_id}`)}>View</Button>
                        </TableBodyCell>
                      </TableBodyRow>
                    {/each}
                  </TableBody>
                </Table>
              </div>
            {/if}

            {#if jobsData.completed_jobs && jobsData.completed_jobs.length > 0}
              <div>
                <h3 class="text-lg font-semibold mb-2 text-green-600 dark:text-green-400">Completed Jobs</h3>
                <Table>
                  <TableHead>
                    <TableHeadCell>Job ID</TableHeadCell>
                    <TableHeadCell>Job Name</TableHeadCell>
                    <TableHeadCell>Status</TableHeadCell>
                    <TableHeadCell>Node ID</TableHeadCell>
                    <TableHeadCell>Detail</TableHeadCell>
                    <TableHeadCell>Updated At</TableHeadCell>
                    <TableHeadCell>Actions</TableHeadCell>
                  </TableHead>
                  <TableBody>
                    {#each jobsData.completed_jobs.slice(0, 10) as job}
                      <TableBodyRow>
                        <TableBodyCell>{job.job_id?.slice(0, 8)}...</TableBodyCell>
                        <TableBodyCell>{job.job?.job_name || 'N/A'}</TableBodyCell>
                        <TableBodyCell>
                          <Badge color={getStatusBadge(job.status)}>{job.status || 'Unknown'}</Badge>
                        </TableBodyCell>
                        <TableBodyCell>{job.node_id?.slice(0, 20) || 'N/A'}</TableBodyCell>
                        <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{job.detail || '-'}</TableBodyCell>
                        <TableBodyCell>{formatShortDate(job.updated_at)}</TableBodyCell>
                        <TableBodyCell>
                          <Button size="xs" on:click={() => navigate(`/jobs/${job.job_id}`)}>View</Button>
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
            {#if jobsData.active_jobs && jobsData.active_jobs.length > 0}
              <Table>
                <TableHead>
                  <TableHeadCell>Job ID</TableHeadCell>
                  <TableHeadCell>Job Name</TableHeadCell>
                  <TableHeadCell>Status</TableHeadCell>
                  <TableHeadCell>Node ID</TableHeadCell>
                  <TableHeadCell>Detail</TableHeadCell>
                  <TableHeadCell>Claimed At</TableHeadCell>
                  <TableHeadCell>Updated At</TableHeadCell>
                  <TableHeadCell>Actions</TableHeadCell>
                </TableHead>
                <TableBody>
                  {#each jobsData.active_jobs as job}
                    <TableBodyRow>
                      <TableBodyCell>{job.job_id?.slice(0, 8)}...</TableBodyCell>
                      <TableBodyCell>{job.job?.job_name || 'N/A'}</TableBodyCell>
                      <TableBodyCell>
                        <Badge color={getStatusBadge(job.status)}>{job.status || 'Unknown'}</Badge>
                      </TableBodyCell>
                      <TableBodyCell>{job.node_id ? job.node_id.slice(0, 20) : 'N/A'}</TableBodyCell>
                      <TableBodyCell class="text-sm text-gray-600 dark:text-gray-400">{job.detail || '-'}</TableBodyCell>
                      <TableBodyCell>{formatShortDate(job.claimed_at)}</TableBodyCell>
                      <TableBodyCell>{formatShortDate(job.updated_at)}</TableBodyCell>
                      <TableBodyCell>
                        <Button size="xs" on:click={() => navigate(`/jobs/${job.job_id}`)}>View</Button>
                      </TableBodyCell>
                    </TableBodyRow>
                  {/each}
                </TableBody>
              </Table>
            {:else}
              <p class="text-center text-gray-500 dark:text-gray-400 py-8">No active jobs</p>
            {/if}
          </div>
        </TabItem>

        <TabItem open={activeTab === 'completed'} title="Completed" on:click={() => handleTabChange('completed')}>
          <div class="pt-4">
            {#if jobsData.completed_jobs && jobsData.completed_jobs.length > 0}
              <Table>
                <TableHead>
                  <TableHeadCell>Job ID</TableHeadCell>
                  <TableHeadCell>Job Name</TableHeadCell>
                  <TableHeadCell>Status</TableHeadCell>
                  <TableHeadCell>Node ID</TableHeadCell>
                  <TableHeadCell>Updated At</TableHeadCell>
                  <TableHeadCell>Actions</TableHeadCell>
                </TableHead>
                <TableBody>
                  {#each jobsData.completed_jobs as job}
                    <TableBodyRow>
                      <TableBodyCell>{job.job_id?.slice(0, 8)}...</TableBodyCell>
                      <TableBodyCell>{job.job?.job_name || 'N/A'}</TableBodyCell>
                      <TableBodyCell>
                        <Badge color={getStatusBadge(job.status)}>{job.status || 'Unknown'}</Badge>
                      </TableBodyCell>
                      <TableBodyCell>{job.node_id?.slice(0, 8) || 'N/A'}</TableBodyCell>
                      <TableBodyCell>{formatShortDate(job.updated_at)}</TableBodyCell>
                      <TableBodyCell>
                        <Button size="xs" on:click={() => navigate(`/jobs/${job.job_id}`)}>View</Button>
                      </TableBodyCell>
                    </TableBodyRow>
                  {/each}
                </TableBody>
              </Table>
            {:else}
              <p class="text-center text-gray-500 dark:text-gray-400 py-8">No completed jobs</p>
            {/if}
          </div>
        </TabItem>
      </Tabs>
    </Card>
  {/if}
</div>

