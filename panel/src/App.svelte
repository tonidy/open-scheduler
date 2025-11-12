<script>
  import { Router, Route } from 'svelte-routing';
  import { authStore } from './stores/authStore';
  import Login from './pages/Login.svelte';
  import Layout from './components/Layout.svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import Jobs from './pages/Jobs.svelte';
  import JobDetails from './pages/JobDetails.svelte';
  import CreateJob from './pages/CreateJob.svelte';
  import Nodes from './pages/Nodes.svelte';
  import NodeDetails from './pages/NodeDetails.svelte';
  import Instances from './pages/Instances.svelte';
  import InstanceDetails from './pages/InstanceDetails.svelte';

  export let url = '';
</script>

<Router {url}>
  {#if !$authStore.token}
    <Route path="*" component={Login} />
  {:else}
    <Layout>
      <Route path="/" component={Dashboard} />
      <Route path="/dashboard" component={Dashboard} />
      <Route path="/jobs" component={Jobs} />
      <Route path="/jobs/create" component={CreateJob} />
      <Route path="/jobs/:id" component={JobDetails} />
      <Route path="/nodes" component={Nodes} />
      <Route path="/nodes/:id" component={NodeDetails} />
      <Route path="/instances" component={Instances} />
      <Route path="/instances/:id" component={InstanceDetails} />
    </Layout>
  {/if}
</Router>

