import { writable, derived } from 'svelte/store';

export interface QueueItem {
  trackId: number;
  title: string;
  artist: string;
  status: 'pending' | 'queued' | 'downloading' | 'completed' | 'error' | 'cancelled';
  error?: string;
  result?: {
    filePath: string;
    fileSize: number;
  };
}

export interface QueueStats {
  total: number;
  pending: number;
  downloading: number;
  completed: number;
  failed: number;
  cancelled: number;
}

// Main queue store
function createQueueStore() {
  const { subscribe, set, update } = writable<Map<number, QueueItem>>(new Map());

  return {
    subscribe,

    addItem: (item: QueueItem) => {
      update(queue => {
        queue.set(item.trackId, item);
        return new Map(queue);
      });
    },

    updateItem: (trackId: number, updates: Partial<QueueItem>) => {
      update(queue => {
        const existing = queue.get(trackId);
        if (existing) {
          queue.set(trackId, { ...existing, ...updates });
        }
        return new Map(queue);
      });
    },

    removeItem: (trackId: number) => {
      update(queue => {
        queue.delete(trackId);
        return new Map(queue);
      });
    },

    clearCompleted: () => {
      update(queue => {
        const newQueue = new Map<number, QueueItem>();
        queue.forEach((item, key) => {
          if (item.status !== 'completed') {
            newQueue.set(key, item);
          }
        });
        return newQueue;
      });
    },

    clearFailed: () => {
      update(queue => {
        const newQueue = new Map<number, QueueItem>();
        queue.forEach((item, key) => {
          if (item.status !== 'error') {
            newQueue.set(key, item);
          }
        });
        return newQueue;
      });
    },

    clearCancelled: () => {
      update(queue => {
        const newQueue = new Map<number, QueueItem>();
        queue.forEach((item, key) => {
          if (item.status !== 'cancelled') {
            newQueue.set(key, item);
          }
        });
        return newQueue;
      });
    },

    clearAll: () => {
      set(new Map());
    },

    reset: () => {
      set(new Map());
    }
  };
}

export const queueStore = createQueueStore();

// Derived store for queue items as array
export const queueItems = derived(queueStore, ($queue) => {
  return Array.from($queue.values());
});

// Derived store for queue stats
export const queueStats = derived(queueStore, ($queue): QueueStats => {
  let pending = 0;
  let downloading = 0;
  let completed = 0;
  let failed = 0;
  let cancelled = 0;

  $queue.forEach(item => {
    switch (item.status) {
      case 'pending':
      case 'queued':
        pending++;
        break;
      case 'downloading':
        downloading++;
        break;
      case 'completed':
        completed++;
        break;
      case 'error':
        failed++;
        break;
      case 'cancelled':
        cancelled++;
        break;
    }
  });

  return {
    total: $queue.size,
    pending,
    downloading,
    completed,
    failed,
    cancelled
  };
});

// Download folder store
export const downloadFolder = writable<string>('');

// Queue paused state
export const queuePaused = writable<boolean>(false);

// Current content store (playlist/album/track being viewed)
export interface TidalContent {
  type: 'playlist' | 'album' | 'track';
  title: string;
  creator: string;
  coverUrl: string;
  tracks: TidalTrack[];
  source?: 'tidal' | 'qobuz';
}

export interface TidalTrack {
  id: number;
  title: string;
  artist: string;
  artists: string;
  album: string;
  albumId: number;
  duration: number;
  trackNumber: number;
  isrc: string;
  coverUrl: string;
  explicit: boolean;
  tidalUrl: string;
}

export const currentContent = writable<TidalContent | null>(null);
