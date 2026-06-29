import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { paymentAPI, paymentStore, subscriptionStore, authStore, appStore, routeMock, routerMock } = vi.hoisted(() => ({
  paymentAPI: {
    getCheckoutInfo: vi.fn()
  },
  paymentStore: {
    createOrder: vi.fn(),
    pollOrderStatus: vi.fn(),
    clearCurrentOrder: vi.fn()
  },
  subscriptionStore: {
    activeSubscriptions: [],
    fetchActiveSubscriptions: vi.fn()
  },
  authStore: {
    user: {
      username: 'test-user',
      balance: 49.4
    },
    refreshUser: vi.fn()
  },
  appStore: {
    showError: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn()
  },
  routeMock: {
    path: '/app/purchase',
    query: {}
  },
  routerMock: {
    resolve: vi.fn((route: { path: string }) => ({ href: route.path })),
    replace: vi.fn(),
    push: vi.fn()
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeMock,
  useRouter: () => routerMock
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => paymentStore
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => subscriptionStore
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/api/payment', () => ({
  paymentAPI
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

vi.mock('@/components/payment/AmountInput.vue', () => ({
  default: {
    name: 'AmountInput',
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: '<input data-testid="amount-input" :value="modelValue ?? \'\'" @input="$emit(\'update:modelValue\', Number($event.target.value))" />'
  }
}))

vi.mock('@/components/payment/PaymentMethodSelector.vue', () => ({
  default: {
    name: 'PaymentMethodSelector',
    template: '<div data-testid="payment-method-selector" />'
  }
}))

vi.mock('@/components/payment/SubscriptionPlanCard.vue', () => ({
  default: {
    name: 'SubscriptionPlanCard',
    template: '<article data-testid="subscription-plan-card" />'
  }
}))

vi.mock('@/components/payment/PaymentStatusPanel.vue', () => ({
  default: {
    name: 'PaymentStatusPanel',
    template: '<section data-testid="payment-status-panel" />'
  }
}))

import PaymentCheckoutContent from '../PaymentCheckoutContent.vue'

function checkoutInfo() {
  return {
    methods: {},
    global_min: 0,
    global_max: 0,
    plans: [],
    balance_disabled: false,
    balance_recharge_multiplier: 1,
    recharge_fee_rate: 0,
    help_text: '',
    help_image_url: '',
    stripe_publishable_key: ''
  }
}

function checkoutInfoWithAlipay() {
  return {
    ...checkoutInfo(),
    methods: {
      alipay: {
        single_min: 1,
        single_max: 1000,
        daily_limit: 0,
        fee_rate: 0,
        available: true
      }
    }
  }
}

function mountContent(variant?: 'legacy' | 'workspace') {
  return mount(PaymentCheckoutContent, {
    props: variant ? { variant } : undefined,
    global: {
      components: {
        RouterLink: {
          props: ['to'],
          setup(props) {
            return {
              href: typeof props.to === 'string' ? props.to : props.to.path
            }
          },
          template: '<a :href="href"><slot /></a>'
        }
      }
    }
  })
}

describe('PaymentCheckoutContent', () => {
  beforeEach(() => {
    paymentAPI.getCheckoutInfo.mockResolvedValue({ data: checkoutInfo() })
    subscriptionStore.fetchActiveSubscriptions.mockResolvedValue([])
    authStore.user.balance = 49.4
    routeMock.path = '/app/purchase'
    routeMock.query = {}
    routerMock.resolve.mockClear()
    routerMock.replace.mockClear()
    routerMock.push.mockClear()
    paymentStore.createOrder.mockReset()
    paymentStore.pollOrderStatus.mockReset()
    paymentStore.clearCurrentOrder.mockReset()
    subscriptionStore.fetchActiveSubscriptions.mockReset()
    subscriptionStore.fetchActiveSubscriptions.mockResolvedValue([])
    appStore.showError.mockClear()
    appStore.showWarning.mockClear()
    appStore.showInfo.mockClear()
    window.localStorage.clear()
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockReturnValue({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn()
      })
    })
  })

  it('uses app-shell links and hides technical shortcuts in workspace mode', async () => {
    const wrapper = mountContent('workspace')
    await flushPromises()

    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    const text = wrapper.text()
    expect(hrefs).toContain('/app/usage')
    expect(hrefs).toContain('/app/keys')
    expect(hrefs).not.toContain('/available-channels')
    expect(hrefs).not.toContain('/redeem')
    expect(text).toContain('所有消费都会记录在用量明细中')
    expect(text).not.toContain('模型 token')
    expect(text).not.toContain('模型倍率')
    expect(text).not.toContain('Images API')
    expect(text).not.toContain('上游账号')
    expect(paymentAPI.getCheckoutInfo).toHaveBeenCalledTimes(1)
  })

  it('keeps legacy shortcuts for the legacy purchase route', async () => {
    const wrapper = mountContent()
    await flushPromises()

    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).toContain('/usage')
    expect(hrefs).toContain('/available-channels')
    expect(hrefs).toContain('/redeem')
    expect(hrefs).toContain('/keys')
  })

  it('does not expose create-order actions when checkout has no payment methods or plans', async () => {
    paymentAPI.getCheckoutInfo.mockResolvedValue({
      data: {
        ...checkoutInfo(),
        methods: {},
        plans: []
      }
    })

    const wrapper = mountContent('workspace')
    await flushPromises()

    expect(wrapper.text()).toContain('payment.notAvailable')
    expect(wrapper.text()).not.toContain('payment.createOrder')

    const subscriptionTab = wrapper
      .findAll('button')
      .find((button) => button.text().includes('payment.tabSubscribe'))
    expect(subscriptionTab).toBeTruthy()

    await subscriptionTab?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('payment.noPlans')
    expect(wrapper.text()).not.toContain('payment.createOrder')
    expect(paymentStore.createOrder).not.toHaveBeenCalled()
  })

  it('keeps the user in select state when create order fails', async () => {
    paymentAPI.getCheckoutInfo.mockResolvedValue({
      data: checkoutInfoWithAlipay()
    })
    paymentStore.createOrder.mockRejectedValue({
      reason: 'PAYMENT_GATEWAY_ERROR',
      message: 'payment gateway unavailable'
    })

    const wrapper = mountContent('workspace')
    await flushPromises()

    await wrapper.find('[data-testid="amount-input"]').setValue('10')
    await flushPromises()

    const submit = wrapper
      .findAll('button')
      .find((button) => button.text().includes('payment.createOrder'))
    expect(submit).toBeTruthy()

    await submit?.trigger('click')
    await flushPromises()

    expect(paymentStore.createOrder).toHaveBeenCalledWith(expect.objectContaining({
      amount: 10,
      order_type: 'balance',
      payment_type: 'alipay'
    }))
    expect(appStore.showError).toHaveBeenCalledWith(expect.stringContaining('payment.errors.alipayDesktopUnavailable'))
    expect(wrapper.find('[data-testid="payment-status-panel"]').exists()).toBe(false)
    expect(window.localStorage.getItem('payment.recovery.current')).toBeNull()
  })
})
