import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, redeemAPI, authAPI, authStore, appStore, subscriptionStore } = vi.hoisted(() => ({
  routeState: {
    path: '/app/redeem'
  },
  redeemAPI: {
    getHistory: vi.fn(),
    redeem: vi.fn()
  },
  authAPI: {
    getPublicSettings: vi.fn()
  },
  authStore: {
    user: {
      balance: 49.4,
      concurrency: 10
    },
    refreshUser: vi.fn()
  },
  appStore: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  },
  subscriptionStore: {
    fetchActiveSubscriptions: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => params?.days ? `${key}:${params.days}` : key
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => subscriptionStore
}))

vi.mock('@/api', () => ({
  redeemAPI,
  authAPI
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => value
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
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import RedeemView from '../RedeemView.vue'

describe('RedeemView', () => {
  beforeEach(() => {
    routeState.path = '/app/redeem'
    redeemAPI.getHistory.mockReset()
    redeemAPI.redeem.mockReset()
    authAPI.getPublicSettings.mockReset()
    authStore.refreshUser.mockReset()
    appStore.showError.mockReset()
    appStore.showSuccess.mockReset()
    appStore.showWarning.mockReset()
    subscriptionStore.fetchActiveSubscriptions.mockReset()
    redeemAPI.getHistory.mockResolvedValue([])
    authAPI.getPublicSettings.mockResolvedValue({ contact_info: '' })
  })

  it('renders the redeem form inside the user workbench shell on /app/redeem', async () => {
    const wrapper = mount(RedeemView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('兑换码')
    expect(wrapper.text()).toContain('redeem.redeemCodeLabel')
    expect(redeemAPI.getHistory).toHaveBeenCalledTimes(1)
  })

  it('keeps the legacy layout when used outside the app workbench', async () => {
    routeState.path = '/redeem'

    const wrapper = mount(RedeemView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })
})
