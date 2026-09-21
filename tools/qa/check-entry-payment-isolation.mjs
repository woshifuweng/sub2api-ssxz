import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../backend/internal/web/dist')
const html = readFileSync(resolve(root, 'index.html'), 'utf8')
const entry = html.match(/<script[^>]+src="([^"]+\.js)"/)
assert(entry, 'Built entry script missing')
const visited = new Set()
function inspect(file) {
  assert(file.startsWith(root), 'Dependency leaves build directory')
  if (visited.has(file)) return
  visited.add(file)
  const source = readFileSync(file, 'utf8')
  assert(!/static(?:-demo)?\.airwallex\.com|checkout(?:-demo)?\.airwallex\.com/.test(source),
    `Optional payment SDK is eager on every page: ${file}`)
  for (const match of source.matchAll(/\bimport\s*(?:[^(';"\n]*?\bfrom\s*)?["']([^"']+)["']/g)) {
    if (match[1].startsWith('.')) inspect(resolve(dirname(file), match[1]))
  }
}
inspect(resolve(root, entry[1].replace(/^\//, '')))
console.log(`Payment SDK isolated from ${visited.size} entry modules`)
