import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { PaymentOrder } from '@/types/payment'

const { appStore, authStore, paymentAPI, redeemAPI } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {
      payment_enabled: false as boolean,
      purchase_subscription_enabled: false as boolean
    },
    showSuccess: vi.fn(),
    showError: vi.fn()
  },
  authStore: {
    user: {
      balance: 49.4
    }
  },
  paymentAPI: {
    getMyOrders: vi.fn(),
    getRefundEligibleProviders: vi.fn(),
    cancelOrder: vi.fn(),
    requestRefund: vi.fn()
  },
  redeemAPI: {
    getHistory: vi.fn()
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/api/payment', () => ({
  paymentAPI
}))

vi.mock('@/api/redeem', () => ({
  default: redeemAPI,
  redeemAPI
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: `
      <main data-testid="app-section-shell">
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
        <slot />
      </main>
    `
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

vi.mock('@/components/payment/OrderStatusBadge.vue', () => ({
  default: {
    name: 'OrderStatusBadge',
    props: ['status'],
    template: '<span data-testid="order-status">{{ status }}</span>'
  }
}))

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: ['show', 'title', 'width'],
    emits: ['close'],
    template: `
      <section v-if="show" data-testid="base-dialog">
        <h2>{{ title }}</h2>
        <slot />
        <slot name="footer" />
      </section>
    `
  }
}))

import AppOrdersView from '../AppOrdersView.vue'

function order(overrides: Partial<PaymentOrder> = {}): PaymentOrder {
  return {
    id: 9,
    user_id: 8,
    amount: 12.34,
    pay_amount: 12.34,
    fee_rate: 0,
    payment_type: 'alipay',
    out_trade_no: 'ORDER-9',
    status: 'COMPLETED',
    order_type: 'balance',
    created_at: '2026-06-18T08:00:00Z',
    expires_at: '2026-06-18T08:30:00Z',
    refund_amount: 0,
    ...overrides
  }
}

function ordersResponse(items: PaymentOrder[], total = items.length) {
  return {
    data: {
      items,
      total,
      page: 1,
      page_size: 10,
      pages: Math.ceil(total / 10)
    }
  }
}

function mountView() {
  return mount(AppOrdersView, {
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

describe('AppOrdersView', () => {
  beforeEach(() => {
    appStore.cachedPublicSettings = {
      payment_enabled: false,
      purchase_subscription_enabled: false
    }
    authStore.user.balance = 49.4
    appStore.showSuccess.mockReset()
    appStore.showError.mockReset()
    paymentAPI.getMyOrders.mockReset()
    paymentAPI.getRefundEligibleProviders.mockReset()
    paymentAPI.cancelOrder.mockReset()
    paymentAPI.requestRefund.mockReset()
    redeemAPI.getHistory.mockReset()
    paymentAPI.getMyOrders.mockResolvedValue(ordersResponse([]))
    paymentAPI.getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: [] } })
    paymentAPI.cancelOrder.mockResolvedValue({})
    paymentAPI.requestRefund.mockResolvedValue({})
    redeemAPI.getHistory.mockResolvedValue([])
  })

  it('shows billing records copy and the payment-off notice when payment is disabled', async () => {
    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('账单记录')
    expect(text).toContain('当前可用方式：兑换码')
    expect(text).toContain('通过兑换码补充账户额度')
    expect(text).toContain('使用兑换码')
    expect(text).toContain('暂无账单记录')
    expect(text).not.toContain('联系管理员')
    expect(text).not.toContain('新版工作台')
    expect(text).not.toContain('真实订单接口')
    expect(text).not.toContain('旧版订单页')
    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).not.toContain('/app/purchase')
    expect(hrefs).toContain('/app/redeem')
    expect(redeemAPI.getHistory).toHaveBeenCalledTimes(1)
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
    expect(paymentAPI.getRefundEligibleProviders).not.toHaveBeenCalled()
  })

  it('renders redeem history as billing records when payment is disabled', async () => {
    redeemAPI.getHistory.mockResolvedValue([
      {
        id: 7,
        code: 'CODE-777',
        type: 'balance',
        value: 25,
        status: 'used',
        used_at: '2026-06-20T09:30:00Z',
        created_at: '2026-06-01T00:00:00Z'
      },
      {
        id: 8,
        code: 'SUB-888',
        type: 'subscription',
        value: 0,
        status: 'used',
        used_at: '2026-06-21T09:30:00Z',
        created_at: '2026-06-01T00:00:00Z',
        validity_days: 30,
        group: { id: 2, name: 'Pro 订阅' }
      }
    ])

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="billing-records-table"]').exists()).toBe(true)
    expect(text).toContain('兑换码入账')
    expect(text).toContain('+$25.00 额度')
    expect(text).toContain('CODE-777')
    expect(text).toContain('订阅开通')
    expect(text).toContain('Pro 订阅（30 天）')
    expect(wrapper.find('[data-testid="status-filter"]').exists()).toBe(false)
    expect(redeemAPI.getHistory).toHaveBeenCalledTimes(1)
  })

  it('keeps existing order history visible while payment actions are disabled', async () => {
    paymentAPI.getMyOrders.mockResolvedValue(ordersResponse([
      order({ id: 31, status: 'PENDING', out_trade_no: 'ORDER-OFF' })
    ]))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('历史充值订单')
    expect(wrapper.text()).toContain('ORDER-OFF')
    expect(wrapper.find('[data-testid="cancel-order-31"]').exists()).toBe(false)
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
    expect(paymentAPI.getRefundEligibleProviders).not.toHaveBeenCalled()
  })

  it('surfaces a legacy-order load error instead of silently hiding history when payment is disabled', async () => {
    paymentAPI.getMyOrders.mockRejectedValue(new Error('orders down'))

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="legacy-orders-error"]').exists()).toBe(true)
    expect(text).toContain('历史充值订单暂时无法加载')
    expect(text).toContain('暂无账单记录')
    expect(text).not.toContain('orders down')
  })

  it('keeps the configured legacy purchase entry from the disabled order state', async () => {
    appStore.cachedPublicSettings = {
      payment_enabled: false,
      purchase_subscription_enabled: true
    }

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('当前账号可查看已有账户记录')
    expect(text).toContain('已有订单和账户变化会保留在账户记录中')
    expect(text).toContain('查看补充额度方式')
    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).toContain('/app/purchase')
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
    expect(paymentAPI.getRefundEligibleProviders).not.toHaveBeenCalled()
  })

  it('renders real order data inside the workbench shell when payment is enabled', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true, purchase_subscription_enabled: false }
    paymentAPI.getMyOrders.mockResolvedValue(ordersResponse([
      order({ provider_instance_id: 'provider-1' })
    ]))
    paymentAPI.getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: ['provider-1'] } })

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('$49.40')
    expect(text).toContain('余额充值')
    expect(text).toContain('支付 ¥12.34')
    expect(text).toContain('到账 $12.34 额度')
    expect(text).toContain('支付宝')
    expect(text).toContain('COMPLETED')
    expect(text).toContain('ORDER-9')
    expect(text).toContain('申请退款')
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
    expect(paymentAPI.getRefundEligibleProviders).toHaveBeenCalledTimes(1)
  })

  it('lets the user cancel a pending order from the app orders page', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true, purchase_subscription_enabled: false }
    paymentAPI.getMyOrders.mockResolvedValue(ordersResponse([
      order({ id: 18, status: 'PENDING', out_trade_no: 'ORDER-18' })
    ]))

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="cancel-order-18"]').trigger('click')
    expect(wrapper.text()).toContain('确认取消订单')

    await wrapper.get('[data-testid="confirm-cancel-order"]').trigger('click')
    await flushPromises()

    expect(paymentAPI.cancelOrder).toHaveBeenCalledWith(18)
    expect(appStore.showSuccess).toHaveBeenCalledWith('订单已取消')
    expect(paymentAPI.getMyOrders).toHaveBeenLastCalledWith({ page: 1, page_size: 10 })
  })

  it('lets the user request a refund when the provider allows user refunds', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true, purchase_subscription_enabled: false }
    paymentAPI.getMyOrders.mockResolvedValue(ordersResponse([
      order({ id: 22, provider_instance_id: 'provider-2', out_trade_no: 'ORDER-22' })
    ]))
    paymentAPI.getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: ['provider-2'] } })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="request-refund-22"]').trigger('click')
    await wrapper.get('[data-testid="refund-reason"]').setValue('paid by mistake')
    await wrapper.get('[data-testid="confirm-refund-request"]').trigger('click')
    await flushPromises()

    expect(paymentAPI.requestRefund).toHaveBeenCalledWith(22, { reason: 'paid by mistake' })
    expect(appStore.showSuccess).toHaveBeenCalledWith('退款申请已提交')
    expect(paymentAPI.getMyOrders).toHaveBeenLastCalledWith({ page: 1, page_size: 10 })
  })

  it('supports status filtering and pagination', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true, purchase_subscription_enabled: false }
    paymentAPI.getMyOrders.mockResolvedValue(ordersResponse([
      order({ id: 1, status: 'PENDING' })
    ], 25))

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="status-filter"]').setValue('PENDING')
    await flushPromises()
    expect(paymentAPI.getMyOrders).toHaveBeenLastCalledWith({ page: 1, page_size: 10, status: 'PENDING' })

    await wrapper.get('[data-testid="next-page"]').trigger('click')
    await flushPromises()
    expect(paymentAPI.getMyOrders).toHaveBeenLastCalledWith({ page: 2, page_size: 10, status: 'PENDING' })
  })

  it('shows an empty state when the user has no orders', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true, purchase_subscription_enabled: false }

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('暂无账户记录')
    expect(text).toContain('完成充值、兑换或额度调整后')
    expect(text).toContain('补充额度')
    expect(text).toContain('使用兑换码')
    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).toContain('/app/purchase')
    expect(hrefs).toContain('/app/redeem')
    expect(text).not.toContain('真实订单接口')
    expect(text).not.toContain('旧版订单页')
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
  })

  it('shows a user-facing load error when orders cannot be loaded', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true, purchase_subscription_enabled: false }
    paymentAPI.getMyOrders.mockRejectedValue(new Error('network down'))

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('账户记录暂时无法加载')
    expect(text).toContain('账户记录正在更新，请稍后刷新')
    expect(text).not.toContain('联系管理员')
    expect(text).not.toContain('network down')
    expect(text).not.toContain('真实订单接口')
    expect(text).not.toContain('旧版订单页')
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
  })
})
