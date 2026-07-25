import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('default locale', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
    Object.defineProperty(navigator, 'language', {
      configurable: true,
      value: 'en-US'
    })
  })

  it('defaults new visitors to Chinese even on a non-Chinese browser', async () => {
    const { i18n } = await import('../index')

    expect(i18n.global.locale.value).toBe('zh')
  })

  it('respects an explicit saved English preference', async () => {
    localStorage.setItem('sub2api_locale', 'en')
    const { i18n } = await import('../index')

    expect(i18n.global.locale.value).toBe('en')
  })
})
