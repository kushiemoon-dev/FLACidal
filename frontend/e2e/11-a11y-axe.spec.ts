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
          await page.addScriptTag({ path: axePath })

          const violations = await page.evaluate(async () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const results = await (window as any).axe.run(document)
            return results.violations.map((v: { id: string; impact: string; nodes: { target: unknown[] }[] }) => ({
              id: v.id,
              impact: v.impact,
              targets: v.nodes.map((n) => n.target.join(' ')),
            }))
          })

          const blocking = violations.filter(
            (v: { id: string; impact: string }) =>
              v.impact === 'serious' || v.impact === 'critical' || BLOCKING_IDS.has(v.id),
          )
          const report = blocking
            .map((v: { id: string; targets: string[] }) => `${v.id} [${view} ${theme} ${viewport.width}px]\n  ${v.targets.join('\n  ')}`)
            .join('\n')
          expect(blocking, report).toEqual([])
        })
      }
    })
  }
}
