import { describe, it, expect, beforeEach } from 'vitest'
import { get } from 'svelte/store'
import {
  queueStore,
  queueItems,
  queueStats,
  downloadFolder,
  queuePaused,
  currentContent,
  type QueueItem,
  type TidalContent,
} from '../stores/queue'

describe('queueStore', () => {
  beforeEach(() => {
    queueStore.reset()
  })

  describe('addItem', () => {
    it('should add an item to the queue', () => {
      const item: QueueItem = {
        trackId: 12345,
        title: 'One More Time',
        artist: 'Daft Punk',
        status: 'pending',
      }

      queueStore.addItem(item)

      const items = get(queueItems)
      expect(items).toHaveLength(1)
      expect(items[0].trackId).toBe(12345)
      expect(items[0].title).toBe('One More Time')
    })

    it('should add multiple items', () => {
      const item1: QueueItem = {
        trackId: 1,
        title: 'Track 1',
        artist: 'Artist 1',
        status: 'pending',
      }
      const item2: QueueItem = {
        trackId: 2,
        title: 'Track 2',
        artist: 'Artist 2',
        status: 'pending',
      }

      queueStore.addItem(item1)
      queueStore.addItem(item2)

      const items = get(queueItems)
      expect(items).toHaveLength(2)
    })

    it('should replace item with same trackId', () => {
      const item1: QueueItem = {
        trackId: 1,
        title: 'Original Title',
        artist: 'Artist',
        status: 'pending',
      }
      const item2: QueueItem = {
        trackId: 1,
        title: 'Updated Title',
        artist: 'Artist',
        status: 'queued',
      }

      queueStore.addItem(item1)
      queueStore.addItem(item2)

      const items = get(queueItems)
      expect(items).toHaveLength(1)
      expect(items[0].title).toBe('Updated Title')
    })
  })

  describe('updateItem', () => {
    it('should update an existing item', () => {
      const item: QueueItem = {
        trackId: 1,
        title: 'Track',
        artist: 'Artist',
        status: 'pending',
      }

      queueStore.addItem(item)
      queueStore.updateItem(1, { status: 'downloading' })

      const items = get(queueItems)
      expect(items[0].status).toBe('downloading')
    })

    it('should not crash when updating non-existent item', () => {
      queueStore.updateItem(999, { status: 'completed' })
      const items = get(queueItems)
      expect(items).toHaveLength(0)
    })
  })

  describe('removeItem', () => {
    it('should remove an item', () => {
      const item: QueueItem = {
        trackId: 1,
        title: 'Track',
        artist: 'Artist',
        status: 'pending',
      }

      queueStore.addItem(item)
      queueStore.removeItem(1)

      const items = get(queueItems)
      expect(items).toHaveLength(0)
    })
  })

  describe('clearCompleted', () => {
    it('should clear only completed items', () => {
      const items: QueueItem[] = [
        { trackId: 1, title: 'T1', artist: 'A1', status: 'completed' },
        { trackId: 2, title: 'T2', artist: 'A2', status: 'pending' },
        { trackId: 3, title: 'T3', artist: 'A3', status: 'completed' },
      ]

      items.forEach((item) => queueStore.addItem(item))
      queueStore.clearCompleted()

      const remaining = get(queueItems)
      expect(remaining).toHaveLength(1)
      expect(remaining[0].trackId).toBe(2)
    })
  })

  describe('clearFailed', () => {
    it('should clear only error items', () => {
      const items: QueueItem[] = [
        { trackId: 1, title: 'T1', artist: 'A1', status: 'error', error: 'Failed' },
        { trackId: 2, title: 'T2', artist: 'A2', status: 'completed' },
        { trackId: 3, title: 'T3', artist: 'A3', status: 'error', error: 'Failed' },
      ]

      items.forEach((item) => queueStore.addItem(item))
      queueStore.clearFailed()

      const remaining = get(queueItems)
      expect(remaining).toHaveLength(1)
      expect(remaining[0].status).toBe('completed')
    })
  })

  describe('clearCancelled', () => {
    it('should clear only cancelled items', () => {
      const items: QueueItem[] = [
        { trackId: 1, title: 'T1', artist: 'A1', status: 'cancelled' },
        { trackId: 2, title: 'T2', artist: 'A2', status: 'pending' },
      ]

      items.forEach((item) => queueStore.addItem(item))
      queueStore.clearCancelled()

      const remaining = get(queueItems)
      expect(remaining).toHaveLength(1)
      expect(remaining[0].status).toBe('pending')
    })
  })

  describe('clearAll', () => {
    it('should clear all items', () => {
      const items: QueueItem[] = [
        { trackId: 1, title: 'T1', artist: 'A1', status: 'completed' },
        { trackId: 2, title: 'T2', artist: 'A2', status: 'pending' },
        { trackId: 3, title: 'T3', artist: 'A3', status: 'error', error: 'Failed' },
      ]

      items.forEach((item) => queueStore.addItem(item))
      queueStore.clearAll()

      const remaining = get(queueItems)
      expect(remaining).toHaveLength(0)
    })
  })
})

describe('queueStats', () => {
  beforeEach(() => {
    queueStore.reset()
  })

  it('should calculate correct stats', () => {
    const items: QueueItem[] = [
      { trackId: 1, title: 'T1', artist: 'A1', status: 'pending' },
      { trackId: 2, title: 'T2', artist: 'A2', status: 'queued' },
      { trackId: 3, title: 'T3', artist: 'A3', status: 'downloading' },
      { trackId: 4, title: 'T4', artist: 'A4', status: 'completed' },
      { trackId: 5, title: 'T5', artist: 'A5', status: 'error', error: 'Failed' },
      { trackId: 6, title: 'T6', artist: 'A6', status: 'cancelled' },
    ]

    items.forEach((item) => queueStore.addItem(item))

    const stats = get(queueStats)

    expect(stats.total).toBe(6)
    expect(stats.pending).toBe(2) // pending + queued
    expect(stats.downloading).toBe(1)
    expect(stats.completed).toBe(1)
    expect(stats.failed).toBe(1)
    expect(stats.cancelled).toBe(1)
  })

  it('should return zeros for empty queue', () => {
    const stats = get(queueStats)

    expect(stats.total).toBe(0)
    expect(stats.pending).toBe(0)
    expect(stats.downloading).toBe(0)
    expect(stats.completed).toBe(0)
    expect(stats.failed).toBe(0)
    expect(stats.cancelled).toBe(0)
  })
})

describe('downloadFolder', () => {
  it('should have default empty string', () => {
    const folder = get(downloadFolder)
    expect(folder).toBe('')
  })

  it('should be updatable', () => {
    downloadFolder.set('/music/downloads')
    expect(get(downloadFolder)).toBe('/music/downloads')
    downloadFolder.set('')
  })
})

describe('queuePaused', () => {
  it('should have default false', () => {
    const paused = get(queuePaused)
    expect(paused).toBe(false)
  })

  it('should be updatable', () => {
    queuePaused.set(true)
    expect(get(queuePaused)).toBe(true)
    queuePaused.set(false)
  })
})

describe('currentContent', () => {
  it('should have default null', () => {
    const content = get(currentContent)
    expect(content).toBeNull()
  })

  it('should be updatable with TidalContent', () => {
    const content: TidalContent = {
      type: 'album',
      id: 'album-123',
      title: 'Discovery',
      creator: 'Daft Punk',
      coverUrl: 'https://example.com/cover.jpg',
      tracks: [],
      source: 'tidal',
    }

    currentContent.set(content)
    const stored = get(currentContent)

    expect(stored).not.toBeNull()
    expect(stored?.title).toBe('Discovery')
    expect(stored?.type).toBe('album')

    currentContent.set(null)
  })
})
