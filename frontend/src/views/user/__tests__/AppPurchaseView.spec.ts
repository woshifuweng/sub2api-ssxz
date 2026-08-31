import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import AppPurchaseView from '../AppPurchaseView.vue'

const AppSectionShellStub = {
  template: '<section><slot /></section>',
}

describe('AppPurchaseView', () => {
  it('keeps the hosted shop for desktop and provides a safe mobile escape hatch', () => {
    const wrapper = mount(AppPurchaseView, {
      global: {
        stubs: {
          AppSectionShell: AppSectionShellStub,
        },
      },
    })

    const iframe = wrapper.get('iframe')
    expect(iframe.attributes('src')).toBe('https://pay.ldxp.cn/shop/VT7XKDFI')
    expect(iframe.attributes('loading')).toBe('lazy')

    const links = wrapper.findAll('a')
    expect(links).toHaveLength(2)
    for (const link of links) {
      expect(link.attributes('href')).toBe('https://pay.ldxp.cn/shop/VT7XKDFI')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toBe('noopener noreferrer')
    }
  })
})
