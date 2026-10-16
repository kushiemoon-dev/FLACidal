import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { expect, type APIRequestContext, type Page } from '@playwright/test'
import { snapshotRealDotDir } from './dotdir.mjs'

export const realDotDirNow: () => { exists: boolean; files: string[] } = snapshotRealDotDir

const here = path.dirname(fileURLToPath(import.meta.url))
const FIXTURES = path.join(here, 'fixtures')

export type ServerMode = 'open' | 'token'

export function readServerInfo(mode: ServerMode): { root: string; downloadDir: string; home: string; realDotDir: { exists: boolean; files: string[] } } {
  const file = path.join(here, '..', 'test-results', `e2e-http-${mode}.json`)
  return JSON.parse(fs.readFileSync(file, 'utf8'))
}

/** Copies fixture FLACs into a fresh temp directory and returns it. */
export function seedFlac(names: string[]): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'flacidal-e2e-case-'))
  for (const name of names) fs.copyFileSync(path.join(FIXTURES, name), path.join(dir, name))
  return dir
}

/** Points the server at `folder`, so each test owns its state. */
export async function setFolder(request: APIRequestContext, folder: string, token?: string) {
  const res = await request.post('/api/folder', {
    data: { folder },
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  })
  expect(res.ok(), await res.text()).toBeTruthy()
}

export const SIDEBAR_VIEWS = ['Home', 'Search', 'Queue', 'Files', 'History', 'Settings', 'Terminal', 'About']
export const TOOL_COUNT = 5
export const VIEWS = [...SIDEBAR_VIEWS, ...Array.from({ length: TOOL_COUNT }, (_, i) => `tool-${i}`)]

export async function openView(page: Page, view: string) {
  if (view.startsWith('tool-')) {
    await page.locator('.sidebar button[title="Tools"]').click()
    await page.locator('.flyout-item').nth(Number(view.slice(5))).click()
  } else {
    await page.locator(`.sidebar button[title="${view}"]`).click()
  }
  await page.waitForTimeout(250) // transition:fade is 150 ms
}

/** Collects console/page errors and every 4xx/5xx response. */
export function collectProblems(page: Page) {
  const errors: string[] = []
  const badResponses: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('console', (m) => {
    if (m.type() === 'error') errors.push(m.text())
  })
  page.on('response', (r) => {
    if (r.status() >= 400) badResponses.push(`${r.status()} ${new URL(r.url()).pathname}`)
  })
  return { errors, badResponses }
}
