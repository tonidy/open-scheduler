# Open Scheduler Control Panel

A modern web-based control panel for managing Open Scheduler jobs, nodes, and instances.

## Features

- 🎨 Modern UI with Flowbite CSS and Tailwind CSS
- 📊 Real-time dashboard with cluster statistics
- 💼 Job management (create, view, monitor)
- 🖥️ Node monitoring and health checks
- 📦 Instance tracking and event logs
- 🔐 JWT-based authentication
- 🔄 Auto-refresh for live updates

## Tech Stack

- **Frontend Framework:** Svelte 4
- **Build Tool:** Vite
- **UI Components:** Flowbite Svelte
- **Styling:** Tailwind CSS
- **Routing:** svelte-routing
- **Backend API:** Centro REST API (proxied via Vite)

## Prerequisites

- Node.js (v18 or higher)
- npm or yarn
- Running Centro server (backend API on port 8080)

## Installation

1. Install dependencies:

```bash
cd panel
npm install
```

2. Start the development server:

```bash
npm run dev
```

The panel will be available at `http://localhost:3000`

## Building for Production

```bash
npm run build
```

The production-ready files will be in the `dist` directory.

To preview the production build:

```bash
npm run preview
```

## Configuration

The Vite development server is configured to proxy API requests to the Centro backend:

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080` (proxied through `/api`)

To change the backend URL, edit `vite.config.js`:

```js
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://your-centro-server:8080',
      changeOrigin: true
    }
  }
}
```

## Default Login Credentials

- **Username:** admin
- **Password:** admin123

## Pages

### Dashboard (`/`)
- System statistics overview
- Real-time job and node counts
- Cluster health monitoring

### Jobs (`/jobs`)
- View all jobs (queued, active, completed)
- Create new jobs with custom configurations
- Monitor job status and events
- View detailed job information

### Nodes (`/nodes`)
- List all registered nodes
- View node resources (CPU, RAM, Disk)
- Check node health status
- Real-time heartbeat monitoring

### Instances (`/instances`)
- View running instances
- Track instance lifecycle
- View instance metadata
- Monitor instance events

## API Integration

The panel integrates with the Centro REST API:

- **Authentication:** `/api/v1/auth/login`
- **Jobs:** `/api/v1/jobs`
- **Nodes:** `/api/v1/nodes`
- **Instances:** `/api/v1/instances`
- **Statistics:** `/api/v1/stats`

All requests (except login) require a JWT token obtained from the login endpoint.

## Development

### Project Structure

```
panel/
├── src/
│   ├── api/              # API client
│   │   └── client.js
│   ├── components/       # Reusable components
│   │   └── Layout.svelte
│   ├── pages/           # Page components
│   │   ├── Dashboard.svelte
│   │   ├── Jobs.svelte
│   │   ├── JobDetails.svelte
│   │   ├── CreateJob.svelte
│   │   ├── Nodes.svelte
│   │   ├── NodeDetails.svelte
│   │   ├── Instances.svelte
│   │   ├── InstanceDetails.svelte
│   │   └── Login.svelte
│   ├── stores/          # State management
│   │   └── authStore.js
│   ├── App.svelte       # Root component
│   ├── main.js          # Entry point
│   └── app.css          # Global styles
├── index.html
├── package.json
├── vite.config.js
├── tailwind.config.js
└── README.md
```

### Auto-refresh Intervals

- **Dashboard stats:** 5 seconds
- **Jobs list:** 3 seconds
- **Job details:** 2 seconds
- **Nodes list:** 5 seconds
- **Node details:** 3 seconds
- **Instances list:** 3 seconds
- **Instance details:** 2 seconds

## Features in Detail

### Job Creation

Create jobs with various configurations:
- Multiple driver types (Podman, Incus, Exec, Process)
- Container workloads with custom images
- Resource allocation (CPU, Memory)
- Custom commands and arguments

### Real-time Monitoring

The panel automatically refreshes data to provide real-time insights:
- Live job status updates
- Node health monitoring
- Instance state tracking
- Event timeline visualization

### Responsive Design

The panel is fully responsive and works on:
- Desktop computers
- Tablets
- Mobile devices

## Troubleshooting

### Panel can't connect to backend

1. Ensure Centro server is running on port 8080
2. Check the proxy configuration in `vite.config.js`
3. Verify CORS is enabled on the backend

### Login fails

1. Verify credentials (default: admin/admin123)
2. Check Centro server logs
3. Ensure JWT middleware is working properly

### Jobs not appearing

1. Check if jobs exist in the queue
2. Verify API authentication token is valid
3. Check browser console for errors

## License

Same as Open Scheduler project license.
