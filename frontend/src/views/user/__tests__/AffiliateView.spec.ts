import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, userAPI, appStore, authStore, clipboard } = vi.hoisted(() => ({
  routeState: {
    path: '/app/affiliate'
  },
  userAPI: {
    getAffiliateDetail: vi.fn(),
    transferAffiliateQuota: vi.fn()
  },
  appStore: {
    showError: vi.fn(),
    showSuccess: vi.fn()
  },
  authStore: {
    refreshUser: vi.fn()
  },
  clipboard: {
    copyToClipboard: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('@/api/user', () => ({
  default: userAPI
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => clipboard
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: () => 'safe error'
}))

vi.mock('@/utils/format', () => ({
  formatCurrency: (value: number) => `$${value.toFixed(2)}`,
  formatDateTime: () => '2026-06-19 12:00:00'
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="app-layout"><slot /></main>'
  }
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main data-testid="app-section-shell"><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot /></main>'
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    template: '<span />'
  }
}))

import AffiliateView from '../AffiliateView.vue'

describe('AffiliateView', () => {
  beforeEach(() => {
    routeState.path = '/app/affiliate'
    userAPI.getAffiliateDetail.mockReset()
    userAPI.transferAffiliateQuota.mockReset()
    appStore.showError.mockReset()
    appStore.showSuccess.mockReset()
    authStore.refreshUser.mockReset()
    clipboard.copyToClipboard.mockReset()
    userAPI.getAffiliateDetail.mockResolvedValue({
      aff_code: 'INVITE123',
      effective_rebate_rate_percent: 12.5,
      aff_count: 2,
      aff_quota: 3,
      aff_history_quota: 5,
      aff_frozen_quota: 0,
      invitees: []
    })
  })

  it('renders inside the user workbench shell on /app/affiliate', async () => {
    const wrapper = mount(AffiliateView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('邀请返利')
    expect(wrapper.text()).toContain('INVITE123')
    expect(userAPI.getAffiliateDetail).toHaveBeenCalledTimes(1)
  })

  it('keeps the legacy layout when used outside the app workbench', async () => {
    routeState.path = '/affiliate'

    const wrapper = mount(AffiliateView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })
})
