import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserOrdersView from '../UserOrdersView.vue'

const {
  appStore,
  getMyOrders,
  getRefundEligibleProviders,
  cancelOrder,
  requestRefund,
  push,
} = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: { payment_enabled: false as boolean },
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
  getMyOrders: vi.fn(),
  getRefundEligibleProviders: vi.fn(),
  cancelOrder: vi.fn(),
  requestRefund: vi.fn(),
  push: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getMyOrders,
    getRefundEligibleProviders,
    cancelOrder,
    requestRefund,
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

const messages: Record<string, string> = {
  'purchase.notEnabledTitle': 'Feature not enabled',
  'purchase.notEnabledDesc': 'Recharge and order features are not enabled.',
  'nav.buySubscription': 'Recharge / Subscription',
  'common.all': 'All',
  'common.refresh': 'Refresh',
  'common.cancel': 'Cancel',
  'common.processing': 'Processing',
  'common.success': 'Success',
  'common.error': 'Error',
  'payment.status.pending': 'Pending',
  'payment.status.completed': 'Completed',
  'payment.status.failed': 'Failed',
  'payment.status.refunded': 'Refunded',
  'payment.result.backToRecharge': 'Back to recharge',
  'payment.orders.cancel': 'Cancel order',
  'payment.orders.requestRefund': 'Request refund',
  'payment.orders.orderId': 'Order ID',
  'payment.orders.amount': 'Amount',
  'payment.confirmCancel': 'Confirm cancel',
  'payment.refundReason': 'Refund reason',
  'payment.refundReasonPlaceholder': 'Tell us why',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const AppLayoutStub = { template: '<main><slot /></main>' }
const OrderTableStub = {
  props: ['orders', 'loading'],
  template: '<section data-testid="orders-table"></section>',
}
const BaseDialogStub = { template: '<section><slot /><slot name="footer" /></section>' }
const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: '<div />',
}
const PaginationStub = { template: '<nav />' }
const IconStub = { template: '<span />' }

function mountView() {
  return mount(UserOrdersView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        OrderTable: OrderTableStub,
        BaseDialog: BaseDialogStub,
        Select: SelectStub,
        Pagination: PaginationStub,
        Icon: IconStub,
      },
    },
  })
}

describe('UserOrdersView payment disabled state', () => {
  beforeEach(() => {
    appStore.cachedPublicSettings = { payment_enabled: false }
    appStore.showError.mockReset()
    appStore.showSuccess.mockReset()
    getMyOrders.mockReset()
    getRefundEligibleProviders.mockReset()
    cancelOrder.mockReset()
    requestRefund.mockReset()
    push.mockReset()
  })

  it('shows a safe disabled state without calling order APIs when payment is off', async () => {
    const wrapper = mountView()

    await flushPromises()

    expect(wrapper.text()).toContain('Feature not enabled')
    expect(wrapper.text()).toContain('Recharge and order features are not enabled.')
    expect(getMyOrders).not.toHaveBeenCalled()
    expect(getRefundEligibleProviders).not.toHaveBeenCalled()
  })

  it('loads orders normally when payment is enabled', async () => {
    appStore.cachedPublicSettings = { payment_enabled: true }
    getMyOrders.mockResolvedValue({ data: { items: [], total: 0 } })
    getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: [] } })

    const wrapper = mountView()

    await flushPromises()

    expect(wrapper.find('[data-testid="orders-table"]').exists()).toBe(true)
    expect(getMyOrders).toHaveBeenCalledWith({
      page: 1,
      page_size: 20,
      status: undefined,
    })
    expect(getRefundEligibleProviders).toHaveBeenCalledTimes(1)
  })
})
