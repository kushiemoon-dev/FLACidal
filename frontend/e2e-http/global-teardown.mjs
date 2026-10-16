// Safety net for a launcher killed before its own cleanup ran. It runs before
// the web servers are stopped, so the removal is repeated (harmlessly) by
// start-server.mjs once the server has exited.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export default function globalTeardown() {
  const here = path.dirname(fileURLToPath(import.meta.url))
  for (const mode of ['open', 'token']) {
    const file = path.join(here, '..', 'test-results', `e2e-http-${mode}.json`)
    try {
      const { root } = JSON.parse(fs.readFileSync(file, 'utf8'))
      if (root) fs.rmSync(root, { recursive: true, force: true })
    } catch {
      // already cleaned
    }
    fs.rmSync(file, { force: true })
  }
}
