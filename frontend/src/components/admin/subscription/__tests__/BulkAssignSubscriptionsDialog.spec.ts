import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BulkAssignSubscriptionsDialog from '../BulkAssignSubscriptionsDialog.vue'
import { adminAPI } from '@/api/admin'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: { searchUsers: vi.fn() },
    subscriptions: { bulkAssign: vi.fn() }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

const BaseDialogStub = {
  props: ['show'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
}

const ConfirmDialogStub = {
  props: ['show', 'message'],
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" data-testid="confirmation">
      <span>{{ message }}</span>
      <button data-testid="confirm" @click="$emit('confirm')">confirm</button>
    </div>
  `
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: `
    <button
      type="button"
      data-testid="group-select"
      @click="$emit('update:modelValue', options[0].value)"
    >group</button>
  `
}

const groups = [
  {
    id: 11,
    name: 'Monthly Pro',
    description: 'Subscription group',
    platform: 'openai',
    rate_multiplier: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: 'subscription',
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null
  }
]

function mountDialog() {
  return mount(BulkAssignSubscriptionsDialog, {
    props: { show: true, groups: groups as any },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        Select: SelectStub,
        GroupBadge: true,
        GroupOptionItem: true,
        Icon: true
      }
    }
  })
}

describe('BulkAssignSubscriptionsDialog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(adminAPI.usage.searchUsers).mockReset()
    vi.mocked(adminAPI.subscriptions.bulkAssign).mockReset()
    vi.mocked(adminAPI.usage.searchUsers).mockResolvedValue([
      { id: 101, email: 'one@example.com' },
      { id: 202, email: 'two@example.com' }
    ])
    vi.mocked(adminAPI.subscriptions.bulkAssign).mockResolvedValue({
      success_count: 2,
      created_count: 1,
      reused_count: 1,
      failed_count: 0,
      subscriptions: [],
      errors: [],
      statuses: { '101': 'created', '202': 'reused' }
    })
  })

  it('searches and selects multiple users, then submits only after confirmation', async () => {
    const wrapper = mountDialog()

    await wrapper.get('[data-testid="bulk-user-search"]').setValue('example')
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()

    await wrapper.get('[data-testid="bulk-user-101"]').trigger('click')
    await wrapper.get('[data-testid="bulk-user-202"]').trigger('click')
    await wrapper.get('[data-testid="group-select"]').trigger('click')
    await wrapper.get('#bulk-assign-subscriptions-form').trigger('submit.prevent')

    expect(adminAPI.subscriptions.bulkAssign).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="confirmation"]').text()).toContain('"count":2')

    await wrapper.get('[data-testid="confirm"]').trigger('click')
    await flushPromises()

    expect(adminAPI.subscriptions.bulkAssign).toHaveBeenCalledTimes(1)
    expect(adminAPI.subscriptions.bulkAssign).toHaveBeenCalledWith({
      user_ids: [101, 202],
      group_id: 11,
      validity_days: 30
    })
    expect(wrapper.emitted('assigned')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
