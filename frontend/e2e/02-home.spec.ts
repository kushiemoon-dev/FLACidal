import { test, expect } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

test.describe('Home page', () => {
  test('URL input is visible and focusable', async ({ page }) => {
    await injectWailsMocks(page)
    await page.goto('/')

    const input = page.locator('input.url-input')
    await expect(input).toBeVisible()
    await input.focus()
    await expect(input).toBeFocused()
  })

  test('clicking Fetch with empty input does nothing destructive', async ({ page }) => {
    await injectWailsMocks(page)
    await page.goto('/')

    const fetchBtn = page.locator('button.btn-primary', { hasText: /Fetch/ })
    await expect(fetchBtn).toBeVisible()
    await fetchBtn.click()
    // No content card should be created
    await expect(page.locator('.content-card')).toHaveCount(0)
    // Empty state still visible
    await expect(page.locator('.empty-state h3', { hasText: 'Ready to Download' })).toBeVisible()
  })

  test('Tidal album URL → source detected → fetch displays content card', async ({ page }) => {
    await injectWailsMocks(page, {
      FetchContentFromURL: {
        type: 'album',
        id: 'a1',
        title: 'Random Access Memories',
        creator: 'Daft Punk',
        coverUrl: '',
        source: 'tidal',
        tracks: [
          {
            id: 1,
            title: 'Get Lucky',
            artists: 'Daft Punk',
            duration: 248,
            isrc: 'X',
            tidalUrl: 'https://tidal.com/track/1',
            available: true,
            popularity: 80,
            explicit: false,
            previewUrl: '',
          },
        ],
      },
    })
    await page.goto('/')

    await page.locator('input.url-input').fill('https://tidal.com/browse/album/12345')
    // wait for source detection badge
    await expect(page.locator('.source-badge.tidal')).toBeVisible()

    await page.locator('button.btn-primary', { hasText: /Fetch/ }).click()
    await expect(page.locator('.content-card')).toBeVisible()
    await expect(page.locator('.content-card h2', { hasText: 'Random Access Memories' })).toBeVisible()
    await expect(page.locator('.creator', { hasText: 'Daft Punk' })).toBeVisible()
    await expect(page.locator('.track-title', { hasText: 'Get Lucky' })).toBeVisible()
  })

  test('Spotify discography URL → confirm dialog with album count', async ({ page }) => {
    await injectWailsMocks(page, {
      ExpandDiscographyURL: [
        'https://open.spotify.com/album/aaa',
        'https://open.spotify.com/album/bbb',
        'https://open.spotify.com/album/ccc',
      ],
    })
    await page.goto('/')

    await page.locator('input.url-input').fill(
      'https://open.spotify.com/artist/4tZwfgrHOc3mvqYlEYSvVi/discography/all',
    )
    await page.locator('button.btn-primary', { hasText: /Fetch/ }).click()

    const confirm = page.locator('.discography-confirm')
    await expect(confirm).toBeVisible()
    await expect(confirm).toContainText(/3 albums/)
    await expect(confirm.locator('button', { hasText: 'Confirm' })).toBeVisible()
    await expect(confirm.locator('button', { hasText: 'Cancel' })).toBeVisible()
  })

  test('Cancel button dismisses discography confirm', async ({ page }) => {
    await injectWailsMocks(page, {
      ExpandDiscographyURL: ['https://open.spotify.com/album/aaa'],
    })
    await page.goto('/')

    await page.locator('input.url-input').fill(
      'https://open.spotify.com/artist/abc/discography/all',
    )
    await page.locator('button.btn-primary', { hasText: /Fetch/ }).click()

    await expect(page.locator('.discography-confirm')).toBeVisible()
    await page.locator('.discography-confirm button', { hasText: 'Cancel' }).click()
    await expect(page.locator('.discography-confirm')).toHaveCount(0)
  })
})
