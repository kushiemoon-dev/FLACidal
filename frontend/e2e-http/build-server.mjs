import { execFileSync } from 'node:child_process'
import { mkdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(here, '..', '..')
mkdirSync(path.join(here, '.bin'), { recursive: true })
execFileSync('go', ['build', '-o', path.join(here, '.bin', 'flacidal-server'), './cmd/server'], {
  cwd: repoRoot,
  stdio: 'inherit',
})
