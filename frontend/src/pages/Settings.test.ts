import { describe, it, expect, beforeEach, vi } from 'vitest'

// A fake backend that persists the way the Go handlers do: SaveConfig stores the
// whole config, SetDownloadOptions overwrites downloadQuality and organizeFolders.
// Settings must not call SetDownloadOptions on save; if that call ever comes back,
// this fake makes the stored values drift and the tests fail.
const backend = vi.hoisted(() => ({ stored: {} as Record<string, unknown> }))

const api = vi.hoisted(() => ({
  isWailsRuntime: vi.fn(() => true),
  GetConfig: vi.fn(),
  SaveConfig: vi.fn(),
  GetDownloadOptions: vi.fn(),
  SetDownloadOptions: vi.fn(),
  GetEffectiveSourceOrder: vi.fn(async () => []),
  ListExtensions: vi.fn(async () => []),
  GetRegistrySources: vi.fn(async () => []),
}))

vi.mock('../lib/websocket', () => ({
  EventsOn: vi.fn(() => () => {}),
  EventsOff: vi.fn(),
}))

vi.mock('../lib/api', async (importOriginal) => {
  const real = await importOriginal<Record<string, unknown>>()
  const stubs = Object.fromEntries(
    Object.entries(real).map(([k, v]) => [k, typeof v === 'function' ? vi.fn(async () => undefined) : v]),
  )
  return { ...stubs, ...api }
})

vi.setConfig({ testTimeout: 30000 })

async function mountSettings() {
  const { mount, flushSync, tick } = await import('svelte')
  const { default: Settings } = await import('./Settings.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  mount(Settings, { target })
  const settle = async () => {
    for (let i = 0; i < 8; i++) await tick()
    flushSync()
  }
  await settle()
  return { target, settle }
}

describe('Settings download quality', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
    backend.stored = { downloadQuality: 'LOSSLESS', organizeFolders: true }
    api.isWailsRuntime.mockReturnValue(true)
    api.GetConfig.mockImplementation(async () => ({ ...backend.stored }))
    api.GetDownloadOptions.mockImplementation(async () => ({ quality: backend.stored.downloadQuality }))
    api.SaveConfig.mockImplementation(async (cfg: Record<string, unknown>) => {
      backend.stored = { ...cfg }
    })
    api.SetDownloadOptions.mockImplementation(async (quality: string, _fmt: string, organizeFolders: boolean) => {
      backend.stored = { ...backend.stored, downloadQuality: quality, organizeFolders }
    })
  })

  it('keeps the chosen quality after saving', async () => {
    const { target, settle } = await mountSettings()

    const select = target.querySelector('#download-quality') as HTMLSelectElement
    select.value = 'HI_RES'
    select.dispatchEvent(new Event('change', { bubbles: true }))
    await settle()
    ;(target.querySelector('.btn-accent') as HTMLButtonElement).click()
    await settle()

    // the form did hand HI_RES to the backend, so whatever lands in storage is decided after that
    expect(api.SaveConfig.mock.calls[0][0].downloadQuality).toBe('HI_RES')
    expect(backend.stored.downloadQuality).toBe('HI_RES')
  })

  it('keeps the folder organisation setting after saving', async () => {
    const { target, settle } = await mountSettings()

    ;(target.querySelector('.btn-accent') as HTMLButtonElement).click()
    await settle()

    expect(backend.stored.organizeFolders).toBe(true)
  })
})
