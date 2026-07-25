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
})
