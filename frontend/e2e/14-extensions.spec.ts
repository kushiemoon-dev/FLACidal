import { test, expect, type Page } from '@playwright/test'
import { injectWailsMocks } from './mocks/wails'

const DEMO = {
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
  authSet: { 'X-Api-Key': true },
  category: 'download',
}

async function gotoExtensions(page: Page) {
  await injectWailsMocks(page, { Extensions: [DEMO] })
  await page.goto('/')
  await page.locator('.sidebar button[title="Settings"]').click()
  await page.waitForTimeout(250) // transition:fade is 150 ms
  await expect(page.getByRole('heading', { name: 'Extensions' }).first()).toBeVisible()
}

test.describe('Settings > Extensions', () => {
  test('lists an installed extension without any secret', async ({ page }) => {
    await gotoExtensions(page)
    const card = page.getByRole('listitem').filter({ hasText: 'Demo Ext' }).first()
    await expect(card).toContainText('Version 1.2.3')
    await expect(card).toContainText('api.demo.example')
    await expect(card).toContainText('network')
    await expect(page.getByLabel('Enable Demo Ext')).not.toBeChecked()
    await expect(card.getByLabel('API key')).toHaveAttribute('type', 'password')
    await expect(card.getByLabel('API key')).toHaveValue('')
    await expect(card.getByLabel('API key')).toHaveAttribute('placeholder', /Already set/)
  })

  test('enabling asks for review first, Escape cancels, confirm enables', async ({ page }) => {
    await gotoExtensions(page)
    await page.getByLabel('Enable Demo Ext').click()
    const dialog = page.getByRole('dialog', { name: 'Enable Demo Ext?' })
    await expect(dialog).toContainText('Ada Author')
    await expect(dialog).toContainText('network')
    await expect(dialog).toContainText('api.demo.example')
    await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(page.getByLabel('Enable Demo Ext')).not.toBeChecked()

    await page.getByLabel('Enable Demo Ext').click()
    await page.getByRole('dialog', { name: 'Enable Demo Ext?' }).getByRole('button', { name: 'Enable' }).click()
    await expect(page.getByRole('status').getByText('Extension enabled.')).toBeVisible()
    await expect(page.getByLabel('Enable Demo Ext')).toBeChecked()
    await expect(page.locator('.source-priority-name', { hasText: 'Demo Ext' })).toBeVisible()
  })

  test('install from a URL needs confirmation and lands disabled', async ({ page }) => {
    await gotoExtensions(page)
    await page.getByLabel('Install from a zip URL').fill('https://example.com/extension.zip')
    await page.getByRole('button', { name: 'Install', exact: true }).first().click()
    const dialog = page.getByRole('dialog', { name: 'Install this extension?' })
    await expect(dialog).toContainText('https://example.com/extension.zip')
    await dialog.getByRole('button', { name: 'Install' }).click()
    await expect(page.getByRole('status').getByText(/Extension installed/)).toBeVisible()
    await expect(page.getByLabel('Enable Installed from-url')).not.toBeChecked()
  })

  test('an insecure URL is reported with a translated error', async ({ page }) => {
    await gotoExtensions(page)
    await page.getByLabel('Install from a zip URL').fill('http://example.com/extension.zip')
    await page.getByRole('button', { name: 'Install', exact: true }).first().click()
    await page.getByRole('dialog', { name: 'Install this extension?' }).getByRole('button', { name: 'Install' }).click()
    await expect(page.getByRole('alert').getByText(/only accept https addresses/)).toBeVisible()
  })

  test('uninstall asks for confirmation', async ({ page }) => {
    await gotoExtensions(page)
    await page.getByRole('button', { name: 'Uninstall Demo Ext' }).click()
    const dialog = page.getByRole('dialog', { name: 'Uninstall Demo Ext?' })
    await dialog.getByRole('button', { name: 'Uninstall' }).click()
    await expect(page.getByRole('status').getByText('Extension uninstalled.')).toBeVisible()
    await expect(page.getByLabel('Enable Demo Ext')).toHaveCount(0)
  })

  test('registries: default listed, add, remove, browse and install an entry', async ({ page }) => {
    await gotoExtensions(page)
    const official = 'https://github.com/kushiemoon-dev/flacidal-extensions'
    await expect(page.getByText(official).first()).toBeVisible()
    await expect(page.getByText('(default)').first()).toBeVisible()

    await page.getByLabel('Add a registry (GitHub repository URL)').fill('https://github.com/someone/exts')
    await page.getByRole('button', { name: 'Add', exact: true }).click()
    await expect(page.getByText('https://github.com/someone/exts').first()).toBeVisible()
    await page.getByRole('button', { name: 'Remove registry https://github.com/someone/exts' }).click()
    await expect(page.getByText('https://github.com/someone/exts')).toHaveCount(0)

    await page.getByRole('button', { name: `Browse ${official}` }).click()
    await expect(page.getByText('From the registry')).toBeVisible()
    await page.getByRole('button', { name: 'Install Registry Ext' }).click()
    await page.getByRole('dialog', { name: 'Install this extension?' }).getByRole('button', { name: 'Install' }).click()
    await expect(page.getByLabel('Enable Installed registry-ext')).not.toBeChecked()
  })
})
