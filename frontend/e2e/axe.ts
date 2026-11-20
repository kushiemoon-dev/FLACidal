import { expect, type Page } from '@playwright/test'
import { createRequire } from 'node:module'

const axePath = createRequire(import.meta.url).resolve('axe-core/axe.min.js')

const BLOCKING_IDS = new Set([
  'label',
  'select-name',
  'button-name',
  'landmark-unique',
  'region',
  'scrollable-region-focusable',
  'heading-order',
])

/** Fails on serious/critical axe violations and on the blocking rule ids used by 11-a11y-axe. */
export async function expectNoBlockingViolations(page: Page, label: string) {
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
    (v: { id: string; impact: string }) => v.impact === 'serious' || v.impact === 'critical' || BLOCKING_IDS.has(v.id),
  )
  const report = blocking
    .map((v: { id: string; targets: string[] }) => `${v.id} [${label}]\n  ${v.targets.join('\n  ')}`)
    .join('\n')
  expect(blocking, report).toEqual([])
}

export async function setTheme(page: Page, theme: 'dark' | 'light') {
  await page.evaluate((t) => document.documentElement.setAttribute('data-theme', t), theme)
  await page.waitForTimeout(400) // let colour transitions settle
}
