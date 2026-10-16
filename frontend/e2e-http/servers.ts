import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// Single source of truth shared with start-server.mjs.
const cfg = JSON.parse(fs.readFileSync(path.join(path.dirname(fileURLToPath(import.meta.url)), 'servers.json'), 'utf8'))

export const OPEN_PORT: number = cfg.openPort
export const TOKEN_PORT: number = cfg.tokenPort
export const OPEN_URL = `http://127.0.0.1:${OPEN_PORT}`
export const TOKEN_URL = `http://127.0.0.1:${TOKEN_PORT}`
export const TOKEN: string = cfg.token
