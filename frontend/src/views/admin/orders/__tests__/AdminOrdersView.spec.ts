import { describe, expect, it, beforeEach, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AdminOrdersView from '../AdminOrdersView.vue'

const { getOrders, routeQuery, routerPush } = vi.hoisted(() => ({
  getOrders: vi.fn(),
  routeQuery: {} as Record<string, string>,
  routerPush: vi.fn()
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    getOrders,
    getOrder: vi.fn(),
    cancelOrder: vi.fn(),
    retryRecharge: vi.fn(),
    refundOrder: vi.fn()
  },
  default: {
    getOrders,
    getOrder: vi.fn(),
    cancelOrder: vi.fn(),
    retryRecharge: vi.fn(),
    refundOrder: vi.fn()
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('@/utils/apiError', () => ({
  extractI18nErrorMessage: () => 'error'
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery
  }),
  useRouter: () => ({
    push: routerPush
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const BaseDialogStub = { template: '<div><slot /></div>' }
const OrderTableStub = { template: '<div class="order-table-stub"><slot name="actions" :row="{ id: 1, user_id: 42, status: `COMPLETED` }" /></div>' }

describe('AdminOrdersView investigation filters', () => {
  beforeEach(() => {
    getOrders.mockReset()
    routerPush.mockReset()
    for (const key of Object.keys(routeQuery)) {
      delete routeQuery[key]
    }
    getOrders.mockResolvedValue({
      data: {
        items: [],
        total: 0
      }
    })
  })

  it('passes user_id route query to the admin orders API', async () => {
    routeQuery.user_id = '42'

    const wrapper = mount(AdminOrdersView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          OrderTable: OrderTableStub,
          Pagination: true,
          Select: true,
          Icon: true,
          AdminRefundDialog: true,
          OrderStatusBadge: true
        }
      }
    })

    await flushPromises()

    expect(getOrders).toHaveBeenCalledWith(expect.objectContaining({
      user_id: 42
    }))
    expect(wrapper.text()).toContain('#42')
  })

  it('can clear the user investigation filter without dropping other query params', async () => {
    routeQuery.user_id = '42'
    routeQuery.status = 'COMPLETED'

    const wrapper = mount(AdminOrdersView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          BaseDialog: BaseDialogStub,
          OrderTable: OrderTableStub,
          Pagination: true,
          Select: true,
          Icon: true,
          AdminRefundDialog: true,
          OrderStatusBadge: true
        }
      }
    })

    await flushPromises()

    const resetButton = wrapper.findAll('button').find((button) => button.text() === 'common.reset')
    expect(resetButton).toBeTruthy()
    await resetButton!.trigger('click')

    expect(routerPush).toHaveBeenCalledWith({
      path: '/admin/orders',
      query: { status: 'COMPLETED' }
    })
  })
})
