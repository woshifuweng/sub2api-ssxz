import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/common/BrandLogo.vue', () => ({
  default: {
    name: 'BrandLogo',
    template: '<span data-testid="brand-logo" />'
  }
}))

vi.mock('@/components/foundation', () => ({
  FoundationProvider: {
    name: 'FoundationProvider',
    props: ['theme'],
    template: '<div data-testid="public-docs-foundation"><slot /></div>'
  },
  FoundationButton: {
    name: 'FoundationButton',
    template: '<button><slot /></button>'
  },
  FoundationCard: {
    name: 'FoundationCard',
    template: '<section data-testid="public-docs-card"><slot /></section>'
  }
}))

import PublicDocsView from '../PublicDocsView.vue'

function mountView() {
  return mount(PublicDocsView, {
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>'
        }
      }
    }
  })
}

describe('PublicDocsView', () => {
  it('publishes the same seven-image guide without requiring a customer session', () => {
    const wrapper = mountView()
    const guide = wrapper.get('[data-testid="cc-switch-guide"]')

    expect(wrapper.get('[data-testid="public-docs-page"]').exists()).toBe(true)
    expect(guide.text()).toContain('用 CC Switch 一键接入 SSXZ')
    expect(guide.findAll('img')).toHaveLength(7)
    expect(wrapper.get('a[href="/home"]').text()).toContain('SSXZ')
    expect(wrapper.get('a[href="/login"]').exists()).toBe(true)
    expect(wrapper.get('a[href="/register"]').exists()).toBe(true)
  })

  it('does not import authenticated stores or private API clients', () => {
    const source = readFileSync('src/views/public/PublicDocsView.vue', 'utf-8')

    expect(source).not.toMatch(/from ['"]@\/api\//)
    expect(source).not.toMatch(/from ['"]@\/stores\/(auth|user)/)
    expect(source).not.toContain('useAuthStore')
    expect(source).not.toContain('useUserStore')
    expect(source).toContain('<CcSwitchGuide />')
  })
})
