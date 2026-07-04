import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { PaymentOrder } from '@/types/payment'

const { appStore, authStore, paymentAPI } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: { payment_enabled: false as boolean },
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
    appStore.cachedPublicSettings = { payment_enabled: false }
    authStore.user.balance = 49.4
    appStore.showSuccess.mockReset()
    appStore.showError.mockReset()
    paymentAPI.getMyOrders.mockReset()
    paymentAPI.getRefundEligibleProviders.mockReset()
    paymentAPI.cancelOrder.mockReset()
    paymentAPI.requestRefund.mockReset()
    paymentAPI.getMyOrders.mockResolvedValue(ordersResponse([]))
    paymentAPI.getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: [] } })
    paymentAPI.cancelOrder.mockResolvedValue({})
    paymentAPI.requestRefund.mockResolvedValue({})
  })

  it('shows the workbench disabled state without calling order APIs when payment is off', async () => {
    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('订单记录')
    expect(text).toContain('充值 / 订阅暂未开启')
    expect(text).toContain('当前暂未开放充值或订阅入口')
    expect(text).not.toContain('联系管理员')
    expect(text).not.toContain('新版工作台')
    expect(text).not.toContain('真实订单接口')
    expect(text).not.toContain('旧版订单页')
    expect(paymentAPI.getMyOrders).not.toHaveBeenCalled()
    expect(paymentAPI.getRefundEligibleProviders).not.toHaveBeenCalled()
  })

  it('renders real order data inside the workbench shell when payment is enabled', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true }
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
    appStore.cachedPublicSettings = { payment_enabled: true }
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
    appStore.cachedPublicSettings = { payment_enabled: true }
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
    appStore.cachedPublicSettings = { payment_enabled: true }
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
    appStore.cachedPublicSettings = { payment_enabled: true }

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('暂无订单记录')
    expect(text).toContain('完成充值或购买订阅后')
    expect(text).toContain('去充值')
    expect(text).toContain('使用兑换码')
    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).toContain('/app/purchase')
    expect(hrefs).toContain('/app/redeem')
    expect(text).not.toContain('真实订单接口')
    expect(text).not.toContain('旧版订单页')
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
  })

  it('shows a user-facing load error when orders cannot be loaded', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true }
    paymentAPI.getMyOrders.mockRejectedValue(new Error('network down'))

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('订单记录暂时无法加载')
    expect(text).toContain('订单记录同步中，请稍后刷新重试')
    expect(text).not.toContain('联系管理员')
    expect(text).not.toContain('network down')
    expect(text).not.toContain('真实订单接口')
    expect(text).not.toContain('旧版订单页')
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
  })
})
