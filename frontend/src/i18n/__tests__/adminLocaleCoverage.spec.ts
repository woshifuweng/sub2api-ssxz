import { readFileSync, readdirSync } from 'node:fs'
import { extname, join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { i18n, loadLocaleMessages } from '../index'

const adminSourceRoots = [
  'src/views/admin',
  'src/components/admin',
  'src/components/account',
  'src/components/layout',
]

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) {
      return entry.name === '__tests__' ? [] : sourceFiles(path)
    }
    return ['.ts', '.vue'].includes(extname(entry.name)) && !entry.name.includes('.spec.')
      ? [path]
      : []
  })
}

function staticTranslationKeys(): string[] {
  const keys = new Set<string>()
  const callPattern = /(?:\bt|\bi18n\.global\.t)\(\s*['"]([^'"]+)['"]/g

  for (const relativeRoot of adminSourceRoots) {
    for (const file of sourceFiles(resolve(process.cwd(), relativeRoot))) {
      const source = readFileSync(file, 'utf8')
      for (const match of source.matchAll(callPattern)) {
        // Prefixes such as `payment.status.` are completed dynamically at
        // runtime and are not standalone translation keys.
        if (!match[1].endsWith('.')) {
          keys.add(match[1])
        }
      }
    }
  }

  return [...keys].sort()
}

function hasPath(messages: Record<string, any>, key: string): boolean {
  return key.split('.').reduce<any>((current, part) => current?.[part], messages) !== undefined
}

describe('admin locale coverage', () => {
  it('defines every statically referenced admin translation in Chinese and English', async () => {
    await Promise.all([loadLocaleMessages('zh'), loadLocaleMessages('en')])
    const zh = i18n.global.getLocaleMessage('zh') as Record<string, any>
    const en = i18n.global.getLocaleMessage('en') as Record<string, any>
    const keys = staticTranslationKeys()
    const missing = keys.flatMap((key) => [
      ...(hasPath(zh, key) ? [] : [`zh:${key}`]),
      ...(hasPath(en, key) ? [] : [`en:${key}`]),
    ])

    expect(missing).toEqual([])
  })
})
