import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

/** Fingerprint of the real ~/.flacidal: every file with its mtime and size. */
export function snapshotRealDotDir() {
  const root = path.join(os.homedir(), '.flacidal')
  const out = []
  const walk = (dir) => {
    for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
      const p = path.join(dir, e.name)
      if (e.isDirectory()) walk(p)
      else {
        // lstat: a broken symlink must not crash the launcher.
        const st = fs.lstatSync(p)
        out.push(`${p}:${st.mtimeMs}:${st.size}`)
      }
    }
  }
  if (fs.existsSync(root)) walk(root)
  return { exists: fs.existsSync(root), files: out.sort() }
}
