import { get } from 'svelte/store';
import { authStore, logout } from '../stores/authStore';

const API_BASE = '/api/v1';

async function request(endpoint, options = {}) {
  const auth = get(authStore);
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  if (auth.token && !endpoint.includes('/auth/login')) {
    headers['Authorization'] = `Bearer ${auth.token}`;
  }

  const config = {
    ...options,
    headers,
  };

  try {
    const response = await fetch(`${API_BASE}${endpoint}`, config);
    
    if (response.status === 401) {
      logout();
      throw new Error('Unauthorized');
    }

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'An error occurred' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('API request failed:', error);
    throw error;
  }
}

// Authentication
export const auth = {
  login: (username, password) =>
    request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
};

// Deployments
export const deployments = {
  list: (status = '') => {
    const query = status ? `?status=${status}` : '';
    return request(`/deployments${query}`);
  },
  get: (id) => request(`/deployments/${id}`),
  create: (deploymentData) =>
    request('/deployments', {
      method: 'POST',
      body: JSON.stringify(deploymentData),
    }),
  getStatus: (id) => request(`/deployments/${id}/status`),
  getEvents: (id) => request(`/deployments/${id}/events`),
};

// Legacy alias for backward compatibility (can be removed later)
export const jobs = deployments;

// Instances
export const instances = {
  list: () => request('/instances'),
  get: (id) => request(`/instances/${id}`),
};

// Nodes
export const nodes = {
  list: () => request('/nodes'),
  get: (id) => request(`/nodes/${id}`),
  getHealth: (id) => request(`/nodes/${id}/health`),
};

// Stats
export const stats = {
  get: () => request('/stats'),
};

