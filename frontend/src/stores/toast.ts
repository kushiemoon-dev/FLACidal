import { writable } from 'svelte/store';

interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
}

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);
  let nextId = 0;

  return {
    subscribe,
    show(message: string, type: Toast['type'] = 'success', duration = 3000) {
      const id = nextId++;
      update(toasts => [...toasts, { id, message, type }]);
      setTimeout(() => {
        update(toasts => toasts.filter(t => t.id !== id));
      }, duration);
    }
  };
}

export const toastStore = createToastStore();
