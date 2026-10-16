// Starts the real headless server on throwaway directories for the HTTP e2e
// suite. Usage: node start-server.mjs <open|token>
import { spawn } from 'node:child_process'
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { snapshotRealDotDir } from './dotdir.mjs'

const here = path.dirname(fileURLToPath(import.meta.url))
const cfg = JSON.parse(readFileSync(path.join(here, 'servers.json'), 'utf8'))
const MODES = {
  open: { port: cfg.openPort, token: '' },
  token: { port: cfg.tokenPort, token: cfg.token },
}

const mode = process.argv[2]
if (!MODES[mode]) {
  console.error('usage: start-server.mjs <open|token>')
  process.exit(2)
}

const repoRoot = path.resolve(here, '..', '..')
const root = mkdtempSync(path.join(os.tmpdir(), `flacidal-e2e-${mode}-`))
const home = path.join(root, 'home')
const downloadDir = path.join(root, 'downloads')
mkdirSync(home)
mkdirSync(downloadDir)

const infoDir = path.join(here, '..', 'test-results')
mkdirSync(infoDir, { recursive: true })
const infoFile = path.join(infoDir, `e2e-http-${mode}.json`)
writeFileSync(infoFile, JSON.stringify({ root, downloadDir, home, realDotDir: snapshotRealDotDir() }))

// Minimal environment: nothing from the developer's shell leaks in. The dead
// proxy makes any outbound call fail fast; if the Core HTTP client ignores
// these variables, the attempts are harmless to the verified flows (the health
// probe is deferred by 10 s).
const env = {
  PATH: process.env.PATH ?? '',
  HOME: home,
  DOWNLOAD_FOLDER: downloadDir,
  PORT: String(MODES[mode].port),
  LISTEN_HOST: '127.0.0.1',
  FRONTEND_DIST_DIR: path.join(repoRoot, 'frontend', 'dist'),
  HTTP_PROXY: 'http://127.0.0.1:9',
  HTTPS_PROXY: 'http://127.0.0.1:9',
  NO_PROXY: '127.0.0.1,localhost',
}
if (MODES[mode].token) env.AUTH_TOKEN = MODES[mode].token

const child = spawn(path.join(here, '.bin', 'flacidal-server'), [], { env, stdio: 'inherit' })

let cleaned = false
function cleanup() {
  if (cleaned) return
  cleaned = true
  rmSync(root, { recursive: true, force: true })
  rmSync(infoFile, { force: true })
}

// Playwright signals the whole process group, so the server already got the
// signal. Forwarding it would be a second one, which the server treats as
// "kill now" and skips its graceful shutdown.
process.on('SIGTERM', () => {})
process.on('SIGINT', () => {})
child.on('exit', (code, signal) => {
  cleanup()
  process.exit(code ?? (signal ? 1 : 0))
})
process.on('exit', cleanup)
