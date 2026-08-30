import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'

function readPath(source: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((value, key) => (
    value && typeof value === 'object' ? (value as Record<string, unknown>)[key] : undefined
  ), source)
}

describe('reseller and docs locales', () => {
  it.each([
    'nav.reseller',
    'nav.resellerManager',
    'nav.resellerAdmin',
    'reseller.pages.dashboard.title',
    'reseller.pages.manager.title',
    'reseller.admin.withdrawals.title',
    'reseller.status.pending',
    'docs.title',
    'docs.guide.step1Title',
    'docs.guide.images.success',
  ])('defines %s in zh and en without exposing the key', (key) => {
    expect(readPath(zh, key)).toEqual(expect.any(String))
    expect(readPath(en, key)).toEqual(expect.any(String))
    expect(readPath(zh, key)).not.toBe(key)
    expect(readPath(en, key)).not.toBe(key)
  })
})
