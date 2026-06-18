import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore, authStore, paymentAPI } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: { payment_enabled: false as boolean }
  },
  authStore: {
    user: {
      balance: 49.4
    }
  },
  paymentAPI: {
    getMyOrders: vi.fn()
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

import AppOrdersView from '../AppOrdersView.vue'

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
    paymentAPI.getMyOrders.mockReset()
  })

  it('shows the workbench disabled state without calling order APIs when payment is off', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('订单记录')
    expect(wrapper.text()).toContain('充值 / 订阅暂未开启')
    expect(wrapper.text()).toContain('不会跳转到旧后台壳')
    expect(paymentAPI.getMyOrders).not.toHaveBeenCalled()
  })

  it('renders real order data inside the workbench shell when payment is enabled', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true }
    paymentAPI.getMyOrders.mockResolvedValue({
      data: {
        items: [
          {
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
            refund_amount: 0
          }
        ],
        total: 1,
        page: 1,
        page_size: 10,
        pages: 1
      }
    })

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('$49.40')
    expect(text).toContain('余额充值')
    expect(text).toContain('¥12.34')
    expect(text).toContain('支付宝')
    expect(text).toContain('COMPLETED')
    expect(text).toContain('ORDER-9')
    expect(paymentAPI.getMyOrders).toHaveBeenCalledWith({ page: 1, page_size: 10 })
  })
})
