import { test, expect, type Page } from '@playwright/test'
import { createRequire } from 'node:module'
import { injectWailsMocks } from './mocks/wails'

const axePath = createRequire(import.meta.url).resolve('axe-core/axe.min.js')

const FILES = [1, 2, 3].map((n) => ({
  path: `/music/track-${n}.flac`,
  name: `track-${n}.flac`,
  size: 1024 * 1024 * 20 * n,
  modTime: `2026-10-0${n}T10:00:00Z`,
  title: `Track ${n}`,
  artist: `Artist ${n}`,
  album: `Album ${n}`,
}))

const SIDEBAR_VIEWS = ['Home', 'Search', 'Queue', 'Files', 'History', 'Settings', 'Terminal', 'About']
const TOOL_COUNT = 5
const VIEWS = [...SIDEBAR_VIEWS, ...Array.from({ length: TOOL_COUNT }, (_, i) => `tool-${i}`)]

const THEMES = ['dark', 'light'] as const
const VIEWPORTS = [
  { width: 390, height: 844 },
  { width: 1024, height: 768 },
]

const BLOCKING_IDS = new Set([
  'label',
  'select-name',
  'button-name',
  'landmark-unique',
  'region',
  'scrollable-region-focusable',
  'heading-order',
])

// context scopes the scan; a modal's backdrop makes the page behind it unreliable to score.
async function expectNoBlockingViolations(page: Page, label: string, context?: string) {
  await page.addScriptTag({ path: axePath })

  const violations = await page.evaluate(async (ctx) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const results = await (window as any).axe.run(ctx ? document.querySelector(ctx) : document)
    return results.violations.map((v: { id: string; impact: string; nodes: { target: unknown[] }[] }) => ({
      id: v.id,
      impact: v.impact,
      targets: v.nodes.map((n) => n.target.join(' ')),
    }))
  }, context)

  const blocking = violations.filter(
    (v: { id: string; impact: string }) =>
      v.impact === 'serious' || v.impact === 'critical' || BLOCKING_IDS.has(v.id),
  )
  const report = blocking
    .map((v: { id: string; targets: string[] }) => `${v.id} [${label}]\n  ${v.targets.join('\n  ')}`)
    .join('\n')
  expect(blocking, report).toEqual([])
}

async function openView(page: Page, view: string) {
  if (view.startsWith('tool-')) {
    await page.locator('.sidebar button[title="Tools"]').click()
    await page.locator('.flyout-item').nth(Number(view.slice(5))).click()
  } else {
    await page.locator(`.sidebar button[title="${view}"]`).click()
  }
  await page.waitForTimeout(250) // transition:fade is 150 ms
}

for (const viewport of VIEWPORTS) {
  for (const theme of THEMES) {
    test.describe(`axe ${viewport.width}px ${theme}`, () => {
      test.use({ viewport })

      for (const view of VIEWS) {
        test(view, async ({ page }) => {
          await injectWailsMocks(page, { ListDownloadedFiles: FILES })
          await page.goto('/')
          await expect(page.locator('.sidebar')).toBeVisible()
          await openView(page, view)
          await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), theme)
          await page.waitForTimeout(400) // let colour transitions settle
          await expectNoBlockingViolations(page, `${view} ${theme} ${viewport.width}px`)
        })
      }
    })
  }
}

const INSTALLED_EXTENSION = {
  manifest: {
    id: 'demo-ext',
    name: 'Demo Ext',
    version: '1.2.3',
    author: 'Ada Author',
    category: 'download',
    capabilities: ['source'],
    permissions: ['network'],
    authFields: [{ key: 'X-Api-Key', label: 'API key', type: 'password', required: true }],
  },
  enabled: false,
  hosts: ['api.demo.example'],
  authSet: {},
  category: 'download',
}

for (const viewport of VIEWPORTS) {
  for (const theme of THEMES) {
    test.describe(`axe Settings > Extensions ${viewport.width}px ${theme}`, () => {
      test.use({ viewport })

      for (const state of ['installed extension', 'activation dialog open', 'uninstall confirmation open'] as const) {
        test(state, async ({ page }) => {
          await injectWailsMocks(page, { Extensions: [INSTALLED_EXTENSION] })
          await page.goto('/')
          await expect(page.locator('.sidebar')).toBeVisible()
          await openView(page, 'Settings')
          await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), theme)
          await expect(page.getByLabel('Enable Demo Ext')).toBeVisible()
          if (state === 'activation dialog open') {
            await page.getByLabel('Enable Demo Ext').click()
            await expect(page.getByRole('dialog', { name: 'Enable Demo Ext?' })).toBeVisible()
          }
          if (state === 'uninstall confirmation open') {
            await page.getByRole('button', { name: 'Uninstall Demo Ext' }).click()
            await expect(page.getByRole('dialog', { name: 'Uninstall Demo Ext?' })).toBeVisible()
          }
          await page.waitForTimeout(400) // let colour transitions settle
          await expectNoBlockingViolations(
            page,
            `extensions ${state} ${theme} ${viewport.width}px`,
            state === 'installed extension' ? undefined : '[role="dialog"]',
          )
        })
      }
    })
  }
}
