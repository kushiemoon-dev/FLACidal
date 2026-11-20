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
}))
vi.mock('../lib/api', () => api)

const ALBUMS = [
  { id: 1, title: 'Geogaddi', artist: 'Boards of Canada', releaseDate: '2002-02-18', trackCount: 23, coverUrl: '', label: 'Warp Records' },
  { id: 2, title: 'Selected Ambient Works', artist: 'Aphex Twin', releaseDate: '1992-11-09', trackCount: 13, coverUrl: '', label: 'Apollo' },
  { id: 3, title: 'Untitled', artist: 'Nobody', releaseDate: '2020-01-01', trackCount: 1, coverUrl: '' },
]

vi.setConfig({ testTimeout: 30000 })

async function mountSearch() {
  const { mount, flushSync, tick } = await import('svelte')
  const { default: Search } = await import('./Search.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  mount(Search, { target })
  const settle = async () => { for (let i = 0; i < 5; i++) await tick(); flushSync() }
  await settle()
  return { target, settle }
}

function type(input: HTMLInputElement, value: string) {
  input.value = value
  input.dispatchEvent(new Event('input', { bubbles: true }))
}

async function searchAlbums(target: HTMLElement, settle: () => Promise<void>) {
  ;(target.querySelectorAll('.search-tabs .tab')[1] as HTMLButtonElement).click()
  await settle()
  type(target.querySelector('.search-input') as HTMLInputElement, 'warp')
  await settle()
  ;(target.querySelector('.search-btn') as HTMLButtonElement).click()
  await settle()
}

describe('Search albums label filter', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    api.SearchTidalAlbums.mockResolvedValue(ALBUMS)
  })

  it('shows the label on cards that have one and nothing on the others', async () => {
    const { target, settle } = await mountSearch()
    await searchAlbums(target, settle)

    const cards = [...target.querySelectorAll('.album-card')]
    expect(cards).toHaveLength(3)
    expect(cards[0].querySelector('.album-card-label')?.textContent).toBe('Warp Records')
    expect(cards[2].querySelector('.album-card-label')).toBeNull()
  })

  it('keeps only albums whose label contains the text, case-insensitively', async () => {
    const { target, settle } = await mountSearch()
    await searchAlbums(target, settle)

    type(target.querySelector('.label-filter-input') as HTMLInputElement, '  WARP ')
    await settle()

    const titles = [...target.querySelectorAll('.album-card-title')].map((e) => e.textContent)
    expect(titles).toEqual(['Geogaddi'])
  })

  it('excludes albums without a label while the filter is active and counts them', async () => {
    const { target, settle } = await mountSearch()
    await searchAlbums(target, settle)
    expect(target.querySelector('.label-unknown-count')).toBeNull()

    type(target.querySelector('.label-filter-input') as HTMLInputElement, 'a')
    await settle()

    const titles = [...target.querySelectorAll('.album-card-title')].map((e) => e.textContent)
    expect(titles).not.toContain('Untitled')
    expect(target.querySelector('.label-unknown-count')?.textContent).toContain('1')
  })

  it('resets the label filter on a new search and hides it outside the albums tab', async () => {
    const { target, settle } = await mountSearch()
    await searchAlbums(target, settle)
    type(target.querySelector('.label-filter-input') as HTMLInputElement, 'warp')
    await settle()

    ;(target.querySelector('.search-btn') as HTMLButtonElement).click()
    await settle()
    expect((target.querySelector('.label-filter-input') as HTMLInputElement).value).toBe('')
    expect(target.querySelectorAll('.album-card')).toHaveLength(3)

    ;(target.querySelectorAll('.search-tabs .tab')[0] as HTMLButtonElement).click()
    await settle()
    expect(target.querySelector('.label-filter-input')).toBeNull()
  })
})
