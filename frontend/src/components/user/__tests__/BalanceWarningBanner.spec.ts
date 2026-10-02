import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import BalanceWarningBanner from '@/components/user/BalanceWarningBanner.vue'

const authState = {
  user: {
    balance: 7,
    balance_notify_threshold: 5,
  },
}

const appState = {
  cachedPublicSettings: {
    balance_low_notify_enabled: true,
    balance_low_notify_threshold: 10,
  },
}

vi.mock('@/stores/auth', () => ({ useAuthStore: () => authState }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appState }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, values?: Record<string, unknown>) => values?.threshold ? `${key}:${values.threshold}` : key }) }))

describe('BalanceWarningBanner', () => {
  it('uses the saved personal threshold instead of the higher system default', () => {
    const wrapper = mount(BalanceWarningBanner, {
      global: {
        stubs: {
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.find('[data-testid="low-balance-warning-banner"]').exists()).toBe(false)
  })

  it('falls back to the system threshold when no personal threshold is saved', () => {
    authState.user.balance_notify_threshold = null
    const wrapper = mount(BalanceWarningBanner, {
      global: {
        stubs: {
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.find('[data-testid="low-balance-warning-banner"]').exists()).toBe(true)
  })
})
