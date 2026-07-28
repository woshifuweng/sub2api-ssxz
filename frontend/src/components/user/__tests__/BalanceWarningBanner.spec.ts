import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const authState = {
  user: {
    balance: 0
  }
}

const appState = {
  cachedPublicSettings: {
    balance_low_notify_enabled: true,
    balance_low_notify_threshold: 1
  }
}

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: { threshold?: string }) => ({
      'balanceWarning.message': '余额不足，无法调用接口，请先兑换额度',
      'balanceWarning.action': '去兑换额度',
      'lowBalanceWarning.message': `Balance below $${params?.threshold}`,
      'lowBalanceWarning.action': 'Recharge now'
    })[key] ?? key
  })
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import BalanceWarningBanner from '../BalanceWarningBanner.vue'

function mountBanner() {
  return mount(BalanceWarningBanner, {
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

describe('BalanceWarningBanner', () => {
  beforeEach(() => {
    authState.user.balance = 0
    appState.cachedPublicSettings.balance_low_notify_enabled = true
    appState.cachedPublicSettings.balance_low_notify_threshold = 1
  })

  it('shows the exchange warning and direct redeem link at zero balance', () => {
    const wrapper = mountBanner()

    expect(wrapper.get('[data-testid="balance-warning-banner"]').text()).toContain('余额不足，无法调用接口，请先兑换额度')
    expect(wrapper.get('a').attributes('href')).toBe('/app/redeem')
  })

  it('hides the warning once the account has a positive balance', () => {
    authState.user.balance = 1

    const wrapper = mountBanner()

    expect(wrapper.find('[data-testid="balance-warning-banner"]').exists()).toBe(false)
  })

  it('shows a low-balance warning with the purchase link', () => {
    authState.user.balance = 0.25

    const wrapper = mountBanner()

    expect(wrapper.get('[data-testid="low-balance-warning-banner"]').text()).toContain('Balance below $1.00')
    expect(wrapper.get('[data-testid="low-balance-warning-banner"] a').attributes('href')).toBe('/app/purchase')
  })

  it('hides the low-balance warning at zero, at the threshold, or when disabled', () => {
    authState.user.balance = 0
    expect(mountBanner().find('[data-testid="low-balance-warning-banner"]').exists()).toBe(false)

    authState.user.balance = 1
    expect(mountBanner().find('[data-testid="low-balance-warning-banner"]').exists()).toBe(false)

    authState.user.balance = 0.25
    appState.cachedPublicSettings.balance_low_notify_enabled = false
    expect(mountBanner().find('[data-testid="low-balance-warning-banner"]').exists()).toBe(false)
  })
})
