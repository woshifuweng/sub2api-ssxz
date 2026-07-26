import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { paymentAPI, paymentStore, appStore } = vi.hoisted(() => ({
  paymentAPI: {
    cancelOrder: vi.fn()
  },
  paymentStore: {
    pollOrderStatus: vi.fn()
  },
  appStore: {
    showError: vi.fn()
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => paymentStore
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

vi.mock('qrcode', () => ({
  default: {
    toCanvas: vi.fn()
  }
}))

import PaymentStatusPanel from '../PaymentStatusPanel.vue'

function order(status: 'RECHARGING' | 'COMPLETED') {
  return {
    id: 42,
    user_id: 7,
    amount: 10,
    pay_amount: 10,
    fee_rate: 0,
    payment_type: 'alipay',
    out_trade_no: 'ORDER-42',
    status,
    order_type: 'balance',
    created_at: '2026-06-18T08:00:00Z',
    expires_at: '2026-06-18T08:30:00Z',
    refund_amount: 0
  }
}

describe('PaymentStatusPanel', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-18T08:00:00Z'))
    paymentStore.pollOrderStatus.mockReset()
    paymentAPI.cancelOrder.mockReset()
    appStore.showError.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows paid fulfillment as settling and emits success only after completed', async () => {
    paymentStore.pollOrderStatus
      .mockResolvedValueOnce(order('RECHARGING'))
      .mockResolvedValueOnce(order('COMPLETED'))

    const wrapper = mount(PaymentStatusPanel, {
      props: {
        orderId: 42,
        qrCode: 'https://pay.example.test/qr',
        expiresAt: '2026-06-18T08:30:00Z',
        paymentType: 'alipay',
        orderType: 'balance'
      }
    })

    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(wrapper.text()).toContain('payment.result.settling')
    expect(wrapper.text()).toContain('payment.result.settlingHint')
    expect(wrapper.text()).not.toContain('payment.result.success')
    expect(wrapper.emitted('success')).toBeUndefined()

    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(wrapper.text()).toContain('payment.result.success')
    expect(wrapper.emitted('success')).toHaveLength(1)
  })

  it('does not show the QR fallback after the page enters the background', async () => {
    const originalLocation = window.location
    const originalHidden = Object.getOwnPropertyDescriptor(document, 'hidden')
    let hidden = false
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { assign: vi.fn() },
    })
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      get: () => hidden,
    })

    const wrapper = mount(PaymentStatusPanel, {
      props: {
        orderId: 42,
        amount: 88,
        payAmount: 88,
        qrCode: 'https://qr.alipay.com/dynamic-order-42',
        expiresAt: '2099-01-01T12:30:00Z',
        paymentType: 'alipay',
        orderType: 'balance',
        outTradeNo: 'sub2_20260420abcd1234',
        mobileAlipayDeepLink: true,
      },
      global: { stubs: { Icon: true } },
    })

    await flushPromises()
    hidden = true
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(2200)
    await flushPromises()

    expect(wrapper.find('[data-test="alipay-qr-fallback"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('payment.qr.alipayContinueInApp')

    wrapper.unmount()
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
    if (originalHidden) Object.defineProperty(document, 'hidden', originalHidden)
  })
})
