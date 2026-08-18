import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

test.describe('Navigation and routing', () => {
  test.beforeEach(async ({ page }) => {
    await injectWailsMocks(page)
  })

  test('boots up without raising critical console errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (e) => errors.push(e.message))
    page.on('console', (m) => {
      if (m.type() === 'error') errors.push(m.text())
    })

    await page.goto('/')
    await expect(page.locator('.sidebar')).toBeVisible()
    const blocking = errors.filter(
      (e) => !/font|preload|favicon|net::ERR|Manifest/i.test(e),
    )
    expect(blocking, blocking.join('\n')).toHaveLength(0)
  })

  test('renders the primary sidebar navigation buttons', async ({ page }) => {
    await page.goto('/')
    const sidebar = page.locator('.sidebar')
    await expect(sidebar).toBeVisible()

    // Every nav button carries a `title=` attribute that also acts as its accessible name.
    for (const label of ['Home', 'Search', 'Queue', 'Files', 'History', 'Settings', 'Terminal', 'About']) {
      await expect(sidebar.locator(`button[title="${label}"]`)).toBeVisible()
    }
    await expect(sidebar.locator('button[title="Tools"]')).toBeVisible()
  })

  test('switches between Home, Search, Queue, History, and Settings', async ({ page }) => {
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

  test('opens the Tools flyout and lists all five tools', async ({ page }) => {
    await page.goto('/')
    await page.locator('.sidebar button[title="Tools"]').click()
    const flyout = page.locator('.flyout')
    await expect(flyout).toBeVisible()
    for (const label of ['Audio Quality Analyzer', 'Audio Resampler', 'Audio Converter', 'File Manager', 'Lyrics Manager']) {
      await expect(flyout.locator('.flyout-item', { hasText: label })).toBeVisible()
    }
  })

  test('opens the Audio Converter tool', async ({ page }) => {
    await page.goto('/')
    await page.locator('.sidebar button[title="Tools"]').click()
    await page.locator('.flyout-item', { hasText: 'Audio Converter' }).click()
    await page.waitForTimeout(250) // let the fade transition settle first
    await expect(page.locator('h1', { hasText: /Audio Converter/i }).first()).toBeVisible()
  })

  test('opens the Lyrics Manager tool', async ({ page }) => {
    await page.goto('/')
    await page.locator('.sidebar button[title="Tools"]').click()
    await page.locator('.flyout-item', { hasText: 'Lyrics Manager' }).click()
    await page.waitForTimeout(250)
    await expect(page.locator('h1', { hasText: /Lyrics Manager/i }).first()).toBeVisible()
  })
})
