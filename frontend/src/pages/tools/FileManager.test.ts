import { describe, it, expect, beforeEach, vi } from 'vitest'

const api = vi.hoisted(() => ({
  isWailsRuntime: vi.fn(() => true),
  ListDownloadedFiles: vi.fn(),
  PreviewRename: vi.fn(),
  RenameFiles: vi.fn(),
  SelectDownloadFolder: vi.fn(),
  SetDownloadFolder: vi.fn(),
  GetDownloadFolder: vi.fn(),
  GetRenameTemplates: vi.fn(),
}))
vi.mock('../../lib/api', () => api)

const TEMPLATES = [
  { name: 'Title - Artist', template: '{title} - {artist}' },
  { name: 'Artist - Title', template: '{artist} - {title}' },
]
const FILES = [
  { path: '/m/a.flac', name: 'a.flac', size: 10 },
  { path: '/m/b.flac', name: 'b.flac', size: 20 },
]

async function mountPage() {
  const { mount, flushSync, tick } = await import('svelte')
  const { default: FileManager } = await import('./FileManager.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  mount(FileManager, { target })
  for (let i = 0; i < 5; i++) await tick()
  flushSync()
  return { target, settle: async () => { for (let i = 0; i < 5; i++) await tick(); flushSync() } }
}

// the first mount compiles the Svelte component graph, which is slow when cold
vi.setConfig({ testTimeout: 30000 })

describe('FileManager', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    api.isWailsRuntime.mockReturnValue(true)
    api.GetDownloadFolder.mockResolvedValue('/m')
    api.GetRenameTemplates.mockResolvedValue(TEMPLATES)
    api.ListDownloadedFiles.mockResolvedValue(FILES)
    api.RenameFiles.mockResolvedValue([])
  })

  it('Browse persists the chosen folder, then reloads the list', async () => {
    api.SelectDownloadFolder.mockResolvedValue('/new')
    api.SetDownloadFolder.mockResolvedValue(undefined)
    const { target, settle } = await mountPage()

    ;(target.querySelector('.folder-bar .btn-accent') as HTMLButtonElement).click()
    await settle()

    expect(api.SetDownloadFolder).toHaveBeenCalledWith('/new')
    expect(api.ListDownloadedFiles.mock.calls.length).toBe(2)
    expect((target.querySelector('.folder-input') as HTMLInputElement).value).toBe('/new')
  })

  it('shows an alert when listing fails, distinct from the empty state', async () => {
    api.ListDownloadedFiles.mockRejectedValue(new Error('[folder.missing] /m'))
    const { target } = await mountPage()

    const alert = target.querySelector('[role="alert"]')
    expect(alert?.textContent).toContain('/m')
    expect(alert?.textContent).toContain('does not exist')
    expect(target.textContent).not.toContain('No track files found')
  })

  it('reports partial rename failures with count and reason', async () => {
    api.RenameFiles.mockResolvedValue([
      { oldPath: '/m/a.flac', newPath: '/m/x.flac', success: true },
      { oldPath: '/m/b.flac', newPath: '', success: false, error: 'permission denied' },
    ])
    const { target, settle } = await mountPage()

    ;(target.querySelector('.file-list-left input[type="checkbox"]') as HTMLInputElement).click()
    await settle()
    ;(target.querySelectorAll('.file-list-actions button')[1] as HTMLButtonElement).click()
    await settle()

    expect(target.textContent).toContain('b.flac')
    expect(target.textContent).toContain('permission denied')
  })

  it('takes rename templates from the Core, with the first one selected', async () => {
    const { target } = await mountPage()
    const select = target.querySelector('select') as HTMLSelectElement
    expect([...select.options].map((o) => o.textContent)).toEqual(['Title - Artist', 'Artist - Title'])
    expect(select.value).toBe('{title} - {artist}')
    expect(target.innerHTML).not.toContain('{track}')
  })

  it('has no Lyrics or Covers tabs', async () => {
    const { target } = await mountPage()
    expect(target.querySelector('[role="tablist"]')).toBeNull()
    expect(target.textContent).not.toMatch(/lyric|cover/i)
  })

  it('disables Browse outside the desktop app', async () => {
    api.isWailsRuntime.mockReturnValue(false)
    const { target } = await mountPage()
    expect((target.querySelector('.folder-bar .btn-accent') as HTMLButtonElement).disabled).toBe(true)
    expect(target.textContent).toContain('Settings')
  })
})
