<script>
  import { Sidebar, SidebarWrapper, SidebarGroup, SidebarItem, SidebarDropdownWrapper, SidebarDropdownItem } from 'flowbite-svelte';
  import { Navbar, NavBrand, NavHamburger, NavUl, NavLi, Avatar, Dropdown, DropdownHeader, DropdownItem, DropdownDivider } from 'flowbite-svelte';
  import { ChartPieSolid, BriefcaseSolid, ServerSolid, LayersOutline, ArrowRightToBracketOutline } from 'flowbite-svelte-icons';
  import { authStore, logout } from '../stores/authStore';
  import { navigate } from 'svelte-routing';

  let spanClass = 'pl-2 self-center text-md text-gray-900 whitespace-nowrap dark:text-white';
  let activeUrl = window.location.pathname;

  function handleLogout() {
    logout();
    navigate('/');
  }

  $: username = $authStore.username || 'Admin';
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
  <Navbar let:hidden let:toggle fluid class="border-b">
    <NavBrand href="/">
      <span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white">
        Open Scheduler
      </span>
    </NavBrand>
    <div class="flex items-center md:order-2">
      <Avatar id="avatar-menu" />
      <Dropdown placement="bottom" triggeredBy="#avatar-menu">
        <DropdownHeader>
          <span class="block text-sm">{username}</span>
          <span class="block truncate text-sm font-medium">Admin User</span>
        </DropdownHeader>
        <DropdownDivider />
        <DropdownItem on:click={handleLogout}>Sign out</DropdownItem>
      </Dropdown>
      <NavHamburger on:click={toggle} />
    </div>
  </Navbar>

  <div class="flex">
    <Sidebar {activeUrl} class="h-screen sticky top-0 flex-shrink-0">
      <SidebarWrapper>
        <SidebarGroup>
          <SidebarItem label="Dashboard" href="/">
            <svelte:fragment slot="icon">
              <ChartPieSolid class="w-5 h-5 text-gray-500 transition duration-75 dark:text-gray-400 group-hover:text-gray-900 dark:group-hover:text-white" />
            </svelte:fragment>
          </SidebarItem>
          
          <SidebarItem label="Jobs" href="/jobs">
            <svelte:fragment slot="icon">
              <BriefcaseSolid class="w-5 h-5 text-gray-500 transition duration-75 dark:text-gray-400 group-hover:text-gray-900 dark:group-hover:text-white" />
            </svelte:fragment>
          </SidebarItem>

          <SidebarItem label="Nodes" href="/nodes">
            <svelte:fragment slot="icon">
              <ServerSolid class="w-5 h-5 text-gray-500 transition duration-75 dark:text-gray-400 group-hover:text-gray-900 dark:group-hover:text-white" />
            </svelte:fragment>
          </SidebarItem>

          <SidebarItem label="Instances" href="/instances">
            <svelte:fragment slot="icon">
              <LayersOutline class="w-5 h-5 text-gray-500 transition duration-75 dark:text-gray-400 group-hover:text-gray-900 dark:group-hover:text-white" />
            </svelte:fragment>
          </SidebarItem>
        </SidebarGroup>
      </SidebarWrapper>
    </Sidebar>

    <main class="flex-1 p-4 md:p-6 lg:p-8 overflow-x-hidden w-full">
      <div class="max-w-7xl mx-auto w-full">
        <slot />
      </div>
    </main>
  </div>
</div>

