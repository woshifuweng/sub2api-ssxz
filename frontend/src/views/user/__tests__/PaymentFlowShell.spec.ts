import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeMock, routerMock } = vi.hoisted(() => ({
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
  usePaymentStore: () => ({
    pollOrderStatus: vi.fn(),
    fetchConfig: vi.fn(),
    config: {}
  })
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    cancelOrder: vi.fn(),
    getOrder: vi.fn(),
    resolveOrderPublicByResumeToken: vi.fn(),
    verifyOrderPublic: vi.fn()
  }
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
    template: '<main data-testid="payment-flow-shell"><slot /></main>'
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
  })
})
