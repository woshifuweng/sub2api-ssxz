import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppHeader alignment', () => {
  it('keeps the bordered header itself at the same 4rem height as the sidebar header', () => {
    expect(componentSource).toContain('<header class="glass sticky top-0 z-30 h-16 border-b')
    expect(componentSource).toContain('<div class="flex h-full items-center')
    expect(componentSource).not.toContain('<div class="flex h-16 items-center')
  })
})
