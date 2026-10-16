import { test, expect } from '@playwright/test'
import { collectProblems, openView, readServerInfo, realDotDirNow, setFolder, VIEWS } from './helpers'

test.describe('Boot against the real server', () => {
  test.beforeEach(async ({ request }) => {
    // Other specs may have deleted the folder the shared server last pointed at.
    await setFolder(request, readServerInfo('open').downloadDir)
  })

  test('boots up without raising critical console errors', async ({ page }) => {
    const { errors, badResponses } = collectProblems(page)

    await page.goto('/')
    await expect(page.locator('.sidebar')).toBeVisible()
    for (const view of VIEWS) await openView(page, view)

    // Same filter as e2e/01-navigation.spec.ts, deliberately not widened.
    const blocking = errors.filter((e) => !/font|preload|favicon|net::ERR|Manifest/i.test(e))
    expect(blocking, blocking.join('\n')).toHaveLength(0)
    expect(badResponses, badResponses.join('\n')).toHaveLength(0)
  })

  test('does not touch the real ~/.flacidal', async ({ page }) => {
    // Snapshot taken by start-server.mjs before the server started.
    const before = readServerInfo('open').realDotDir

    await page.goto('/')
    await expect(page.locator('.sidebar')).toBeVisible()
    await openView(page, 'Settings')

    expect(realDotDirNow()).toEqual(before)
  })
})
