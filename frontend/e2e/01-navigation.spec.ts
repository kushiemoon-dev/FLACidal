import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

test.describe('Navigation & routing', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMocks(page)
  })

  test('app loads without critical console errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    page.on('console', (m) => {
      if (m.type() === 'error') errors.push(m.text())
    })

    await page.goto('/')
    // Sidebar logo is the first stable visible element
    await expect(page.locator('.sidebar')).toBeVisible()
    // Filter out the noisy CSS / fonts errors that aren't blockers
    const blocking = errors.filter(
      (e) => !/font|preload|favicon|net::ERR|Manifest/i.test(e),
    )
    expect(blocking, blocking.join('\n')).toHaveLength(0)
  })

  test('sidebar shows main nav buttons', async ({ page }) => {
    await page.goto('/')
    const sidebar = page.locator('.sidebar')
    await expect(sidebar).toBeVisible()

    // Each nav item has a `title=` attribute that doubles as accessible name.
    for (const label of ['Home', 'Search', 'Queue', 'Files', 'History', 'Settings', 'Terminal', 'About']) {
      await expect(sidebar.locator(`button[title="${label}"]`)).toBeVisible()
    }
    // Tools flyout
    await expect(sidebar.locator('button[title="Tools"]')).toBeVisible()
  })

  test('navigates between Home, Search, Queue, History, Settings', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('h1', { hasText: 'FLACidal' })).toBeVisible()

    await page.locator('.sidebar button[title="Search"]').click()
    await expect(page.locator('h1, h2').filter({ hasText: /Search/i }).first()).toBeVisible()

    await page.locator('.sidebar button[title="Queue"]').click()
    await expect(page.locator('h1, h2').filter({ hasText: /Queue/i }).first()).toBeVisible()

    await page.locator('.sidebar button[title="History"]').click()
    await expect(page.locator('h1, h2').filter({ hasText: /History/i }).first()).toBeVisible()

    await page.locator('.sidebar button[title="Settings"]').click()
    await expect(page.locator('h1, h2').filter({ hasText: /Settings/i }).first()).toBeVisible()
  })

  test('Tools flyout opens and reveals all 4 tools', async ({ page }) => {
    await page.goto('/')
    await page.locator('.sidebar button[title="Tools"]').click()
    const flyout = page.locator('.flyout')
    await expect(flyout).toBeVisible()
    for (const label of ['Audio Quality Analyzer', 'Audio Resampler', 'Audio Converter', 'File Manager']) {
      await expect(flyout.locator('.flyout-item', { hasText: label })).toBeVisible()
    }
  })

  test('navigates to Audio Converter tool', async ({ page }) => {
    await page.goto('/')
    await page.locator('.sidebar button[title="Tools"]').click()
    await page.locator('.flyout-item', { hasText: 'Audio Converter' }).click()
    await page.waitForTimeout(250) // wait for fade transition
    await expect(page.locator('h1', { hasText: /Audio Converter/i }).first()).toBeVisible()
  })
})
