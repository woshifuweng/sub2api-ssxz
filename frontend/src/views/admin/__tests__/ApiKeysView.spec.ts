import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ApiKeysView from '../ApiKeysView.vue'

const { list, setEnabled, deleteApiKey, updateApiKeyGroup, getAll, searchUsers, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  setEnabled: vi.fn(),
  deleteApiKey: vi.fn(),
  updateApiKeyGroup: vi.fn(),
  getAll: vi.fn(),
  searchUsers: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    apiKeys: { list, setEnabled, deleteApiKey, updateApiKeyGroup },
    groups: { getAll },
    usage: { searchUsers },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
}))

const DataTableStub = {
  props: ['data', 'columns', 'loading'],
  emits: ['sort'],
  template: `
    <div data-testid="data-table">
      <button data-testid="sort-cost" @click="$emit('sort', 'total_actual_cost', 'asc')">sort</button>
      <div v-for="row in data" :key="row.id" class="row">
        <slot name="cell-user" :row="row" :value="row.user" />
        <slot name="cell-key" :row="row" :value="row.key" />
        <slot name="cell-group" :row="row" :value="row.group" />
        <slot name="cell-total_actual_cost" :row="row" :value="row.total_actual_cost" />
        <slot name="cell-status" :row="row" :value="row.status" />
        <slot name="cell-actions" :row="row" :value="row.id" />
      </div>
    </div>
  `,
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: `<select data-testid="select" :value="modelValue ?? ''" @change="$emit('update:modelValue', Number($event.target.value)); $emit('change')">
    <option v-for="option in options" :key="String(option.value)" :value="option.value ?? ''">{{ option.label }}</option>
  </select>`,
}

describe('admin ApiKeysView inventory actions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getAll.mockResolvedValue([
      { id: 12, name: 'Claude CCMAX 满血池', platform: 'anthropic', rate_multiplier: 1.2 },
      { id: 15, name: 'Codex 池', platform: 'openai', rate_multiplier: 0.8 },
    ])
    searchUsers.mockResolvedValue([])
    setEnabled.mockResolvedValue({})
    deleteApiKey.mockResolvedValue({ deleted: true })
    updateApiKeyGroup.mockResolvedValue({ api_key: {}, auto_granted_group_access: false })
    list.mockResolvedValue({
      items: [
        {
          id: 41,
          user: { id: 9, email: 'customer@example.com', username: 'customer', balance: 8.75 },
          key: 'sk-admin...7890',
          name: 'CC Switch',
          group: { id: 12, name: 'Claude CCMAX 满血池', platform: 'anthropic', rate_multiplier: 1.2 },
          status: 'active',
          quota: 10,
          quota_used: 1.25,
          last_used_at: '2026-07-14T08:00:00Z',
          expires_at: null,
          created_at: '2026-07-01T08:00:00Z',
          today_actual_cost: 0.3,
          last_30_days_actual_cost: 2.4,
          total_actual_cost: 6.8,
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
      summary: { total: 1, active: 1, inactive: 0, expired: 0, last_30_days_actual_cost: 2.4 },
    })
  })

  const mountView = () => mount(ApiKeysView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="actions"/><slot name="filters"/><slot name="table"/><slot name="pagination"/></div>',
        },
        DataTable: DataTableStub,
        Select: SelectStub,
        Pagination: true,
        Icon: true,
        GroupBadge: { props: ['name'], template: '<span>{{ name }}</span>' },
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<div v-if="show" data-testid="group-dialog"><h2>{{ title }}</h2><slot/><slot name="footer"/></div>',
        },
        ConfirmDialog: {
          props: ['show', 'title', 'message'],
          emits: ['confirm', 'cancel'],
          template: '<div v-if="show" data-testid="delete-dialog"><span>{{ message }}</span><button data-testid="confirm-delete" @click="$emit(\'confirm\')">confirm</button></div>',
        },
      },
    },
  })

  it('renders real user/group data and masked keys with only allowlisted admin actions', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('customer@example.com')
    expect(wrapper.text()).toContain('Claude CCMAX 满血池')
    expect(wrapper.text()).toContain('sk-admin...7890')
    expect(wrapper.text()).toContain('$6.80')
    expect(wrapper.text()).not.toContain('reveal')
    expect(wrapper.text()).not.toContain('导入')
    expect(wrapper.find('[data-action="disable"]').exists()).toBe(true)
    expect(wrapper.find('[data-action="change-group"]').exists()).toBe(true)
    expect(wrapper.find('[data-action="delete"]').exists()).toBe(true)
  })

  it('requires explicit confirmation before deleting a customer key', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-action="delete"]').trigger('click')
    expect(deleteApiKey).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="delete-dialog"]').text()).toContain('admin.apiKeyInventory')

    await wrapper.get('[data-testid="confirm-delete"]').trigger('click')
    await flushPromises()

    expect(deleteApiKey).toHaveBeenCalledWith(41)
  })

  it('uses the dedicated status endpoint for disable and refreshes the inventory', async () => {
    const wrapper = mountView()
    await flushPromises()
    list.mockClear()

    await wrapper.get('[data-action="disable"]').trigger('click')
    await flushPromises()

    expect(setEnabled).toHaveBeenCalledWith(41, false)
    expect(list).toHaveBeenCalledTimes(1)
  })

  it('changes exactly one group through the audited admin endpoint', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-action="change-group"]').trigger('click')
    await wrapper.get('[data-testid="group-action-select"]').setValue('15')
    await wrapper.get('[data-testid="save-group-change"]').trigger('click')
    await flushPromises()

    expect(updateApiKeyGroup).toHaveBeenCalledWith(41, 15)
  })

  it('uses zero summary values for an empty successful response', async () => {
    list.mockResolvedValueOnce({
      items: [], total: 0, page: 1, page_size: 20, pages: 1,
      summary: { total: 0, active: 0, inactive: 0, expired: 0, last_30_days_actual_cost: 0 },
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-testid="summary-total"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="summary-cost"]').text()).toContain('$0.00')
    expect(showError).not.toHaveBeenCalled()
  })

  it('requests an allowlisted server sort when a sortable column is selected', async () => {
    const wrapper = mountView()
    await flushPromises()
    list.mockClear()

    await wrapper.get('[data-testid="sort-cost"]').trigger('click')
    await flushPromises()

    expect(list).toHaveBeenCalledWith(
      expect.objectContaining({ sort_by: 'total_actual_cost', sort_order: 'asc' }),
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    )
  })
})
