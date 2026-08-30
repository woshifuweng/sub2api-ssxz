import { beforeEach, describe, expect, it } from 'vitest'
import {
  DEFAULT_SITE_LOGO,
  DEFAULT_SITE_NAME,
  normalizeSiteLogo,
  normalizeSiteName,
  updateFavicon,
} from '@/utils/branding'

describe('updateFavicon', () => {
  beforeEach(() => {
    document.head.innerHTML = '<link rel="icon" href="/logo.svg">'
  })

  it('replaces the default favicon with the configured logo', () => {
    updateFavicon('https://example.com/custom-logo.png')

    const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
    expect(link?.href).toBe('https://example.com/custom-logo.png')
  })

  it('ignores unsafe logo URLs', () => {
    updateFavicon('javascript:alert(1)')

    const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
    expect(link?.getAttribute('href')).toBe('/logo.svg')
  })
})

describe('SSXZ default branding', () => {
  it('uses the SSXZ name and static logo for official empty/default settings', () => {
    expect(DEFAULT_SITE_NAME).toBe('SSXZ AI')
    expect(DEFAULT_SITE_LOGO).toBe('/brand/ssxz-cat-dog-static.svg')
    expect(normalizeSiteName()).toBe(DEFAULT_SITE_NAME)
    expect(normalizeSiteName('Sub2API')).toBe(DEFAULT_SITE_NAME)
    expect(normalizeSiteLogo()).toBe(DEFAULT_SITE_LOGO)
  })

  it('preserves explicitly customized branding', () => {
    expect(normalizeSiteName('Customer Gateway')).toBe('Customer Gateway')
    expect(normalizeSiteLogo('https://example.com/customer.svg')).toBe(
      'https://example.com/customer.svg',
    )
  })
})
