import { describe, it, expect, beforeEach, vi } from 'vitest'

const api = vi.hoisted(() => ({
  SearchTidal: vi.fn(),
  SearchTidalAlbums: vi.fn(),
  SearchTidalArtists: vi.fn(),
  SearchDeezer: vi.fn(),
  FetchContentFromURL: vi.fn(),
  QueueDownloads: vi.fn(),
  QueueSingleDownload: vi.fn(),
  QueueArtistAlbum: vi.fn(),
  SetDownloadOptions: vi.fn(),
  SaveConfig: vi.fn(),
}))
vi.mock('../lib/api', () => api)

const TRACKS = [
  { id: 11, title: 'One', artist: 'Art', artists: 'Art', album: 'Alb', coverUrl: '', duration: 100 },
]
const ALBUMS = [
  { id: 5, title: 'Alb', artist: 'Art', releaseDate: '2020-01-01', trackCount: 3, coverUrl: '' },
]

vi.setConfig({ testTimeout: 30000 })

async function mountSearch() {
  const { mount, flushSync, tick } = await import('svelte')
  const { downloadFolder } = await import('../stores/queue')
  downloadFolder.set('/music')
  const { default: Search } = await import('./Search.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  mount(Search, { target })
  const settle = async () => { for (let i = 0; i < 5; i++) await tick(); flushSync() }
  await settle()
  return { target, settle }
}

async function search(target: HTMLElement, settle: () => Promise<void>, tab: number) {
  ;(target.querySelectorAll('.search-tabs .tab')[tab] as HTMLButtonElement).click()
  await settle()
  const input = target.querySelector('.search-input') as HTMLInputElement
  input.value = 'q'
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await settle()
  ;(target.querySelector('.search-btn') as HTMLButtonElement).click()
  await settle()
}

describe('Search Atmos buttons', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    api.SearchTidal.mockResolvedValue(TRACKS)
    api.SearchTidalAlbums.mockResolvedValue(ALBUMS)
    api.QueueSingleDownload.mockResolvedValue(undefined)
    api.QueueArtistAlbum.mockResolvedValue(3)
  })

  it('a track Atmos button queues with ATMOS and never touches the global quality', async () => {
    const { target, settle } = await mountSearch()
    await search(target, settle, 0)

    ;(target.querySelector('.track-atmos-btn') as HTMLButtonElement).click()
    await settle()

    expect(api.QueueSingleDownload).toHaveBeenCalledWith(11, '/music', 'One', 'Art', 'ATMOS')
    expect(api.SetDownloadOptions).not.toHaveBeenCalled()
    expect(api.SaveConfig).not.toHaveBeenCalled()
  })

  it('the regular track button sends no quality', async () => {
    const { target, settle } = await mountSearch()
    await search(target, settle, 0)

    ;(target.querySelector('.track-download-btn') as HTMLButtonElement).click()
    await settle()

    expect(api.QueueSingleDownload).toHaveBeenCalledWith(11, '/music', 'One', 'Art', '')
  })

  it('an album Atmos button queues the album with ATMOS, the regular one does not', async () => {
    const { target, settle } = await mountSearch()
    await search(target, settle, 1)

    ;(target.querySelector('.album-atmos-btn') as HTMLButtonElement).click()
    await settle()
    expect(api.QueueArtistAlbum).toHaveBeenLastCalledWith('5', 'Art', '/music', 'ATMOS')

    ;(target.querySelector('.album-download-btn') as HTMLButtonElement).click()
    await settle()
    expect(api.QueueArtistAlbum).toHaveBeenLastCalledWith('5', 'Art', '/music', '')
  })

  it('has an explicit accessible name and no Atmos button on the Universal tab', async () => {
    const { target, settle } = await mountSearch()
    await search(target, settle, 0)
    const btn = target.querySelector('.track-atmos-btn') as HTMLButtonElement
    expect(btn.getAttribute('aria-label')).toContain('Dolby Atmos')

    api.SearchDeezer.mockResolvedValue([{ id: 1, title: 'D', artist: 'A', album: 'X', cover: '', duration: 1 }])
    await search(target, settle, 3)
    expect(target.querySelector('.track-atmos-btn')).toBeNull()
  })
})
