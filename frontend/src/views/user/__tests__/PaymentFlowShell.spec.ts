import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { paymentAPI, paymentStore, routeMock, routerMock } = vi.hoisted(() => ({
  paymentAPI: {
    cancelOrder: vi.fn(),
    getOrder: vi.fn(),
    resolveOrderPublicByResumeToken: vi.fn(),
    verifyOrderPublic: vi.fn()
  },
  paymentStore: {
    pollOrderStatus: vi.fn(),
    fetchConfig: vi.fn(),
    config: {}
  },
  routeMock: {
    query: {}
  },
  routerMock: {
    push: vi.fn()
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a :href="to"><slot /></a>'
  },
  useRoute: () => routeMock,
  useRouter: () => routerMock
}))

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => paymentStore
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('@/api/payment', () => ({
  paymentAPI
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="legacy-app-layout"><slot /></main>'
  }
}))

vi.mock('@/components/payment/PaymentFlowShell.vue', () => ({
  default: {
    name: 'PaymentFlowShell',
    props: ['subtitle'],
    template: '<main data-testid="payment-flow-shell"><span data-testid="payment-flow-subtitle">{{ subtitle }}</span><slot /></main>'
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

vi.mock('qrcode', () => ({
  default: {
    toCanvas: vi.fn()
  }
}))

import PaymentQRCodeView from '../PaymentQRCodeView.vue'
import PaymentResultView from '../PaymentResultView.vue'
import StripePaymentView from '../StripePaymentView.vue'

describe('payment flow shell ownership', () => {
  beforeEach(() => {
    routeMock.query = {}
    routerMock.push.mockClear()
    paymentStore.pollOrderStatus.mockReset()
    paymentStore.fetchConfig.mockReset()
    paymentAPI.cancelOrder.mockReset()
    paymentAPI.getOrder.mockReset()
    paymentAPI.resolveOrderPublicByResumeToken.mockReset()
    paymentAPI.verifyOrderPublic.mockReset()
  })

  it('renders the QR payment page in the neutral payment shell instead of the legacy app layout', () => {
    const wrapper = mount(PaymentQRCodeView)

    expect(wrapper.find('[data-testid="payment-flow-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="legacy-app-layout"]').exists()).toBe(false)
  })

  it('renders non-popup Stripe payment in the neutral payment shell', async () => {
    routeMock.query = {}

    const wrapper = mount(StripePaymentView)
    await flushPromises()

    expect(wrapper.find('[data-testid="payment-flow-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="legacy-app-layout"]').exists()).toBe(false)
  })

  it('keeps Stripe popup mode unwrapped for external payment windows', async () => {
    routeMock.query = { method: 'alipay' }

    const wrapper = mount(StripePaymentView)
    await flushPromises()

    expect(wrapper.find('[data-testid="payment-flow-shell"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="legacy-app-layout"]').exists()).toBe(false)
  })

  it('renders the payment result page in the neutral payment shell', async () => {
    routeMock.query = {}

    const wrapper = mount(PaymentResultView)
    await flushPromises()

    expect(wrapper.find('[data-testid="payment-flow-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="legacy-app-layout"]').exists()).toBe(false)
  })

  it('shows an unavailable result state when no payment identifiers are present', async () => {
    routeMock.query = {}

    const wrapper = mount(PaymentResultView)
    await flushPromises()

    expect(wrapper.text()).toContain('payment.result.missingOrder')
    expect(wrapper.text()).toContain('payment.result.missingOrderHint')
    expect(wrapper.text()).not.toContain('payment.result.processing')
    expect(wrapper.text()).not.toContain('payment.result.failed')
    expect(wrapper.find('[data-testid="payment-flow-subtitle"]').text()).toBe('payment.result.missingOrder')
  })

  it('shows processing for a pending payment result instead of success', async () => {
    routeMock.query = { order_id: '42' }
    paymentStore.pollOrderStatus.mockResolvedValue({
      id: 42,
      out_trade_no: 'ORDER-42',
      amount: 10,
      pay_amount: 10,
      fee_rate: 0,
      payment_type: 'alipay',
      order_type: 'balance',
      status: 'PENDING',
      created_at: '2026-06-18T08:00:00Z',
      expires_at: '2026-06-18T08:30:00Z'
    })

    const wrapper = mount(PaymentResultView)
    await flushPromises()

    expect(wrapper.text()).toContain('payment.result.processing')
    expect(wrapper.text()).toContain('payment.result.processingHint')
    expect(wrapper.text()).not.toContain('payment.result.success')
    expect(wrapper.text()).not.toContain('payment.result.failed')
    wrapper.unmount()
  })

  it('shows failed for a failed payment result instead of success', async () => {
    routeMock.query = { order_id: '43' }
    paymentStore.pollOrderStatus.mockResolvedValue({
      id: 43,
      out_trade_no: 'ORDER-43',
      amount: 10,
      pay_amount: 10,
      fee_rate: 0,
      payment_type: 'alipay',
      order_type: 'balance',
      status: 'FAILED',
      created_at: '2026-06-18T08:00:00Z',
      expires_at: '2026-06-18T08:30:00Z'
    })

    const wrapper = mount(PaymentResultView)
    await flushPromises()

    expect(wrapper.text()).toContain('payment.result.failed')
    expect(wrapper.text()).not.toContain('payment.result.success')
    expect(wrapper.text()).not.toContain('payment.result.processing')
  })
})
