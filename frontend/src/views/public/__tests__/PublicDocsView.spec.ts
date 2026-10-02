import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import PublicDocsView from '../PublicDocsView.vue'

const authState = vi.hoisted(() => ({
  isAuthenticated: false,
  isAdmin: false,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState,
}))

const stubs = {
  RouterLink: { template: '<a><slot /></a>' },
  BrandLogo: { template: '<span />' },
  PageHeader: { template: '<div><slot /></div>' },
  CcSwitchGuide: { template: '<div />' },
  FoundationProvider: { template: '<div><slot /></div>' },
  FoundationCard: { template: '<div><slot /></div>' },
  FoundationButton: { template: '<button><slot /></button>' },
}

function mountDocs() {
  return mount(PublicDocsView, { global: { stubs } })
}

describe('PublicDocsView', () => {
  it('shows public login actions to signed-out visitors', () => {
    authState.isAuthenticated = false
    authState.isAdmin = false
    const wrapper = mountDocs()

    expect(wrapper.find('.public-docs-login').text()).toBe('登录')
    expect(wrapper.find('.public-docs-cta').text()).toBe('开始使用')
  })

  it('does not show a misleading login action to signed-in users', () => {
    authState.isAuthenticated = true
    authState.isAdmin = false
    const wrapper = mountDocs()

    expect(wrapper.find('.public-docs-login').exists()).toBe(false)
    expect(wrapper.find('.public-docs-cta').text()).toBe('进入控制台')
  })
})
