import { writable } from 'svelte/store';

const storedAuth = localStorage.getItem('auth');
const initialAuth = storedAuth ? JSON.parse(storedAuth) : { token: null, username: null, expiresAt: null };

export const authStore = writable(initialAuth);

authStore.subscribe(value => {
  if (typeof window !== 'undefined') {
    if (value.token) {
      localStorage.setItem('auth', JSON.stringify(value));
    } else {
      localStorage.removeItem('auth');
    }
  }
});

export function logout() {
  authStore.set({ token: null, username: null, expiresAt: null });
}

