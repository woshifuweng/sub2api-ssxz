// Read-only merge gate: native features and regression tests must not disappear
// silently while applying SSXZ branding. Reachability is separately tested by
// router/feature-access and real component interaction tests.
import { execFileSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'

const root = fileURLToPath(new URL('../../', import.meta.url))
// Pin the reviewed commit: a moved or missing tag must not silently change the gate.
const ref = process.argv[2] || 'aea725f2ea644d5592d0bbb1d63b607efa7e200a'
if (!/^[A-Za-z0-9][A-Za-z0-9._/-]*$/.test(ref)) throw new Error('Invalid upstream ref')
const revision = execFileSync('git', ['rev-parse', '--verify', `${ref}^{commit}`], { cwd: root, encoding: 'utf8' }).trim()
const upstreamFiles = execFileSync('git', ['ls-tree', '-r', '--name-only', revision, '--', 'frontend/src'], { cwd: root, encoding: 'utf8' })
  .trim().split(/\r?\n/).filter(Boolean)
const missing = upstreamFiles.filter(path => !existsSync(resolve(root, path)))
const tests = upstreamFiles.filter(path => /\.(spec|test)\.[cm]?[jt]sx?$/.test(path))
console.log(`Upstream ${ref} (${revision}): ${upstreamFiles.length} frontend source files, ${tests.length} test files`)
if (missing.length) {
  console.error('Native files are missing; restore them or document/review an explicit replacement before updating this gate:')
  missing.forEach(path => console.error(`  ${path}`))
  process.exitCode = 1
} else {
  console.log('PASS: no upstream frontend source or tests silently omitted. This is NOT functional or production acceptance.')
}
