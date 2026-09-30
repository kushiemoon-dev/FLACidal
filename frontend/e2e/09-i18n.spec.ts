import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

const config = (language: string) => ({ theme: 'dark', accentColor: '#f472b6', soundEffects: false, soundVolume: 70, language })

async function homeLabel(page: import('@playwright/test').Page) {
  await page.goto('/')
  await expect(page.locator('.sidebar')).toBeVisible()
  return page.locator('.sidebar .nav-item[aria-current="page"]').getAttribute('aria-label')
}

test.describe('Language selection', () => {
  test('explicit fr shows French', async ({ page }) => {
    await injectWailsMocks(page, { GetConfig: config('fr') })
    expect(await homeLabel(page)).toBe('Accueil')
  })

  test('explicit de shows German', async ({ page }) => {
    await injectWailsMocks(page, { GetConfig: config('de') })
    expect(await homeLabel(page)).toBe('Startseite')
  })

  test.describe('auto with fr-FR browser', () => {
    test.use({ locale: 'fr-FR' })
    test('shows French', async ({ page }) => {
      await injectWailsMocks(page, { GetConfig: config('') })
      expect(await homeLabel(page)).toBe('Accueil')
    })
  })

  test.describe('auto with de-AT browser', () => {
    test.use({ locale: 'de-AT' })
    test('shows German', async ({ page }) => {
      await injectWailsMocks(page, { GetConfig: config('') })
      expect(await homeLabel(page)).toBe('Startseite')
    })
  })

  test.describe('auto with unsupported browser language', () => {
    test.use({ locale: 'es-ES' })
    test('falls back to English', async ({ page }) => {
      await injectWailsMocks(page, { GetConfig: config('') })
      expect(await homeLabel(page)).toBe('Home')
    })
  })
})
