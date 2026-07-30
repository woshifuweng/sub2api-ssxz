import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listAdminWithdrawals, reviewWithdrawal, appStore } = vi.hoisted(() => ({
  listAdminWithdrawals: vi.fn(),
  reviewWithdrawal: vi.fn(),
  appStore: {
    showSuccess: vi.fn(),
    showError: vi.fn(),
    showWarning: vi.fn()
  }
}))

vi.mock('@/api/reseller', () => ({
  default: { listAdminWithdrawals, reviewWithdrawal }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/admin/reseller/withdrawals' })
}))

import AdminWithdrawals from '../AdminWithdrawals.vue'

const pendingRequest = {
  id: 12,
  user_id: 7,
  user_email: 'agent@example.com',
  username: 'Agent Seven',
  amount: 10,
  method: 'manual',
  account_info: null,
  status: 'pending',
  requested_at: '2026-07-30T00:00:00Z',
  note: ''
}

function mountView() {
  return mount(AdminWithdrawals, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        AdminPageHeader: { template: '<header><slot name="actions" /></header>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: {
          props: ['data'],
          template: `
            <div>
              <div v-for="row in data" :key="row.id" class="request-row">
                <slot name="cell-applicant" :row="row" />
                <slot name="cell-actions" :row="row" />
              </div>
            </div>
          `
        },
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show" class="dialog"><slot /><slot name="footer" /></div>'
        },
        Pagination: true,
        LiquidButton: { template: '<button v-bind="$attrs"><slot /></button>' },
        WithdrawalStatusBadge: true,
        Icon: true,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
      }
    }
  })
}

function buttonByText(wrapper: ReturnType<typeof mountView>, text: string) {
  return wrapper.findAll('button').find((button) => button.text().trim() === text)
}

describe('AdminWithdrawals', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listAdminWithdrawals.mockResolvedValue({
      items: [pendingRequest],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    reviewWithdrawal.mockResolvedValue({})
  })

  it('approves a pending request and refreshes the list', async () => {
    const wrapper = mountView()
    await flushPromises()

    await buttonByText(wrapper, '批准')?.trigger('click')
    await flushPromises()

    expect(reviewWithdrawal).toHaveBeenCalledWith(12, { action: 'approve' })
    expect(listAdminWithdrawals).toHaveBeenCalledTimes(2)
    expect(appStore.showSuccess).toHaveBeenCalledWith('兑换申请已批准')
  })

  it('requires a reason before rejecting a request', async () => {
    const wrapper = mountView()
    await flushPromises()

    await buttonByText(wrapper, '拒绝')?.trigger('click')
    const confirm = buttonByText(wrapper, '确认拒绝')

    expect(confirm?.attributes('disabled')).toBeDefined()

    await wrapper.get('textarea').setValue('资料不完整')
    await buttonByText(wrapper, '确认拒绝')?.trigger('click')
    await flushPromises()

    expect(reviewWithdrawal).toHaveBeenCalledWith(12, {
      action: 'reject',
      reason: '资料不完整'
    })
    expect(appStore.showSuccess).toHaveBeenCalledWith('兑换申请已拒绝')
  })
})
