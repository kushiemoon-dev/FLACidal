import { test, expect } from '@playwright/test'
import { openView, readServerInfo, setFolder } from './helpers'

// Saving an Amazon priority endpoint makes Core write a log line, on demand.
const MARKER = 'Amazon proxy priority endpoints'

test('a new log entry shows in the Terminal without reloading, once', async ({ page, request }) => {
  await setFolder(request, readServerInfo('open').downloadDir)
  await page.goto('/')
  await openView(page, 'Terminal')
  const entries = page.locator('.log-entry', { hasText: MARKER })
  await expect(entries).toHaveCount(0)

  const cfg = await (await request.get('/api/config')).json()
  const saved = await request.post('/api/config', { data: { ...cfg, amazonPriorityEndpoints: ['https://127.0.0.1:1'] } })
  expect(saved.ok(), await saved.text()).toBeTruthy()

  await expect(entries).toHaveCount(1)
  await page.waitForTimeout(500)
  await expect(entries).toHaveCount(1)

  // Back to a clean config for the other specs.
  await request.post('/api/config', { data: { ...cfg, amazonPriorityEndpoints: [] } })
})

test('history is reloaded after the page is opened again', async ({ page }) => {
  await page.goto('/')
  await openView(page, 'Terminal')
  await expect(page.locator('.log-entry', { hasText: MARKER }).first()).toBeVisible()
})
