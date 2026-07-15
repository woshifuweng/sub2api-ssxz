import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import BrandLogo from '../BrandLogo.vue'
import { DEFAULT_SITE_LOGO, normalizeSiteLogo, resolveCustomSiteLogo } from '@/utils/brand'

describe('BrandLogo', () => {
  it('renders the theme-adaptive transparent silhouette for shared navigation marks', () => {
    const wrapper = mount(BrandLogo, {
      props: { variant: 'mark', size: '46px', theme: 'dark' }
    })

    expect(wrapper.attributes('style')).toContain('--brand-logo-size: 46px')
    expect(wrapper.classes()).toContain('brand-logo--mark')
    expect(wrapper.classes()).toContain('brand-logo--theme-dark')
    expect(wrapper.get('.brand-logo__mark').exists()).toBe(true)
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('SSXZ')
  })

  it('reuses the approved animated and static assets for the homepage artwork', () => {
    const wrapper = mount(BrandLogo, {
      props: { variant: 'animated', size: '100%', theme: 'light' }
    })
    const images = wrapper.findAll('img')

    expect(images).toHaveLength(2)
    expect(images[0].attributes('src')).toBe('/brand/ssxz-cat-dog-line-draw.svg')
    expect(images[1].attributes('src')).toBe('/brand/ssxz-cat-dog-static.svg')
    expect(images.every((image) => image.attributes('style')?.includes('color-scheme: light'))).toBe(true)
  })

  it('normalizes the legacy default asset without discarding an explicit custom logo', () => {
    expect(normalizeSiteLogo('/logo.png?v=legacy')).toBe(DEFAULT_SITE_LOGO)
    expect(resolveCustomSiteLogo('/logo.png')).toBe('')
    expect(normalizeSiteLogo('/uploads/customer-brand.svg')).toBe('/uploads/customer-brand.svg')
    expect(resolveCustomSiteLogo('/uploads/customer-brand.svg')).toBe('/uploads/customer-brand.svg')
  })
})
