// Helpers for the extension e2e: a throwaway HTTPS server that serves an
// extension zip, and a restartable headless server on a fixed data folder.
import { execFileSync, spawn } from 'node:child_process'
import { mkdirSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import https from 'node:https'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { crc32 } from 'node:zlib'

const here = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(here, '..', '..')

/** Builds a zip in memory (stored, no compression) from [name, string] pairs. */
export function buildZip(entries) {
  const locals = []
  const centrals = []
  let offset = 0
  for (const [name, text] of entries) {
    const nameBuf = Buffer.from(name)
    const data = Buffer.from(text)
    const crc = crc32(data)
    const local = Buffer.alloc(30)
    local.writeUInt32LE(0x04034b50, 0)
    local.writeUInt16LE(20, 4)
    local.writeUInt32LE(crc, 14)
    local.writeUInt32LE(data.length, 18)
    local.writeUInt32LE(data.length, 22)
    local.writeUInt16LE(nameBuf.length, 26)
    locals.push(local, nameBuf, data)

    const central = Buffer.alloc(46)
    central.writeUInt32LE(0x02014b50, 0)
    central.writeUInt16LE(20, 4)
    central.writeUInt16LE(20, 6)
    central.writeUInt32LE(crc, 16)
    central.writeUInt32LE(data.length, 20)
    central.writeUInt32LE(data.length, 24)
    central.writeUInt16LE(nameBuf.length, 28)
    central.writeUInt32LE(offset, 42)
    centrals.push(central, nameBuf)
    offset += local.length + nameBuf.length + data.length
  }
  const centralSize = centrals.reduce((n, b) => n + b.length, 0)
  const end = Buffer.alloc(22)
  end.writeUInt32LE(0x06054b50, 0)
  end.writeUInt16LE(entries.length, 8)
  end.writeUInt16LE(entries.length, 10)
  end.writeUInt32LE(centralSize, 12)
  end.writeUInt32LE(offset, 16)
  return Buffer.concat([...locals, ...centrals, end])
}

/** Starts an HTTPS server on 127.0.0.1 with a fresh self-signed certificate. */
export async function startZipServer(dir, port, zip) {
  const key = path.join(dir, 'key.pem')
  const cert = path.join(dir, 'cert.pem')
  execFileSync(
    'openssl',
    ['req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1', '-subj', '/CN=127.0.0.1',
      '-addext', 'subjectAltName=IP:127.0.0.1', '-keyout', key, '-out', cert],
    { stdio: 'ignore' },
  )
  const server = https.createServer({ key: readFileSync(key), cert: readFileSync(cert) }, (req, res) => {
    if (req.url === '/ext.zip') {
      res.writeHead(200, { 'Content-Type': 'application/zip' })
      res.end(zip)
    } else {
      res.writeHead(404).end()
    }
  })
  await new Promise((resolve) => server.listen(port, '127.0.0.1', resolve))
  return { certPath: cert, url: `https://127.0.0.1:${port}`, stop: () => new Promise((resolve) => server.close(resolve)) }
}

/**
 * Starts the headless server on `root` (HOME and downloads live there, so a
 * second call with the same root is a restart on the same data).
 */
export async function startServer({ root, token, port, certPath }) {
  const home = path.join(root, 'home')
  const downloadDir = path.join(root, 'downloads')
  mkdirSync(home, { recursive: true })
  mkdirSync(downloadDir, { recursive: true })
  const env = {
    PATH: process.env.PATH ?? '',
    HOME: home,
    DOWNLOAD_FOLDER: downloadDir,
    PORT: String(port),
    LISTEN_HOST: '127.0.0.1',
    FRONTEND_DIST_DIR: path.join(repoRoot, 'frontend', 'dist'),
    // Dead proxy for anything external; the local extension host bypasses it.
    HTTP_PROXY: 'http://127.0.0.1:9',
    HTTPS_PROXY: 'http://127.0.0.1:9',
    NO_PROXY: '127.0.0.1,localhost',
    SSL_CERT_FILE: certPath,
    AUTH_TOKEN: token,
  }
  const child = spawn(path.join(here, '.bin', 'flacidal-server'), [], { env, stdio: 'ignore' })
  const exited = new Promise((resolve) => child.once('exit', resolve))
  for (let i = 0; i < 100; i++) {
    try {
      const res = await fetch(`http://127.0.0.1:${port}/api/health`)
      if (res.ok) break
    } catch {
      // not listening yet
    }
    if (child.exitCode !== null) throw new Error('server exited during startup')
    await new Promise((r) => setTimeout(r, 200))
  }
  return {
    async stop() {
      child.kill('SIGTERM')
      await exited
    },
  }
}

export function makeTempRoot() {
  return mkdtempSync(path.join(os.tmpdir(), 'flacidal-e2e-ext-'))
}

export function removeTempRoot(root) {
  rmSync(root, { recursive: true, force: true })
}
