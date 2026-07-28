import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const authState = {
  user: {
    balance: 0
  }
}

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'balanceWarning.message': '余额不足，无法调用接口，请先兑换额度',
      'balanceWarning.action': '去兑换额度'
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
})
