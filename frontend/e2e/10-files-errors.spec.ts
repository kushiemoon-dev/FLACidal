import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

test.describe('Folder errors are visible and translated', () => {
  test('Files shows an alert, not the empty state, when listing fails', async ({ page }) => {
    await injectWailsMocks(page, { ListDownloadedFilesError: '[folder.missing] /gone' })
    await page.goto('/')
    await page.locator('.sidebar button[title="Files"]').click()
    await page.waitForTimeout(250)

    const alert = page.locator('.error-state[role="alert"]').first()
    await expect(alert).toContainText('does not exist')
    await expect(alert).toContainText('/gone')
    await expect(page.locator('.toast').first()).toContainText('/gone')
  })

  test('Settings shows a translated toast when the chosen folder is refused', async ({ page }) => {
    await injectWailsMocks(page, { SetDownloadFolderError: '[folder.not_writable] /flacidal-selected' })
    await page.goto('/')
    await page.locator('.sidebar button[title="Settings"]').click()
    await page.waitForTimeout(250)

    await page.locator('.browse-btn').first().click()

    const toast = page.locator('.toast').first()
    await expect(toast).toContainText('cannot write')
    await expect(toast).toContainText('/flacidal-selected')
  })

  test('Settings explains an imported config whose folder is missing on this machine', async ({ page }) => {
    await injectWailsMocks(page, { ImportConfigError: '[folder.missing] /other-machine/music' })
    await page.goto('/')
    await page.locator('.sidebar button[title="Settings"]').click()
    await page.waitForTimeout(250)

    await page.getByRole('button', { name: /Import Config/i }).first().click()

    await expect(page.locator('.toast').first()).toContainText('does not exist')
    await expect(page.locator('.toast').first()).toContainText('/other-machine/music')
  })

  test('Settings lets you type the folder in the browser and shows a refusal', async ({ page }) => {
    // no Wails mock: the app talks to /api
    let posted = ''
    await page.route('**/api/**', async (route) => {
      const url = new URL(route.request().url())
      const method = route.request().method()
      if (url.pathname === '/api/folder' && method === 'POST') {
        posted = JSON.parse(route.request().postData() ?? '{}').folder
        return route.fulfill({ status: 400, contentType: 'application/json', body: JSON.stringify({ error: `[folder.missing] ${posted}` }) })
      }
      if (url.pathname === '/api/folder') {
        return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ folder: '/old' }) })
      }
      return route.fulfill({ contentType: 'application/json', body: '{}' })
    })
    await page.goto('/')
    await page.locator('.sidebar button[title="Settings"]').click()
    await page.waitForTimeout(250)

    const input = page.locator('#download-folder')
    await expect(input).toBeEditable()
    await expect(page.locator('.browse-btn').first()).toBeDisabled()
    await input.fill('/typed/path')
    await input.blur()

    await expect(page.locator('.toast').first()).toContainText('does not exist')
    expect(posted).toBe('/typed/path')
    await expect(input).toHaveValue('/old')
  })
})
