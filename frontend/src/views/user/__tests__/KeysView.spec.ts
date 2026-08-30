import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

import type { ApiKey } from '@/types'
import KeysView from '../KeysView.vue'

const {
  listKeys,
  getPublicSettings,
  getDashboardApiKeysUsage,
  getAvailableGroups,
  getUserGroupRates,
  getAvailableChannels,
  showError,
  showSuccess,
  copyToClipboard,
  isCurrentStep,
  nextStep,
} = vi.hoisted(() => ({
  listKeys: vi.fn(),
  getPublicSettings: vi.fn(),
  getDashboardApiKeysUsage: vi.fn(),
  getAvailableGroups: vi.fn(),
  getUserGroupRates: vi.fn(),
  getAvailableChannels: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  copyToClipboard: vi.fn(),
  isCurrentStep: vi.fn(),
  nextStep: vi.fn(),
}))

const messages: Record<string, string> = {
  'common.actions': 'Actions',
  'common.name': 'Name',
  'common.refresh': 'Refresh',
  'common.status': 'Status',
  'keys.apiKey': 'API Key',
  'keys.allGroups': 'All Groups',
  'keys.allStatus': 'All Status',
  'keys.columnSettings': 'Column Settings',
  'keys.createKey': 'Create API Key',
  'keys.created': 'Created',
  'keys.expiresAt': 'Expires',
  'keys.group': 'Group',
  'keys.id': 'ID',
  'keys.currentConcurrency': 'Current Concurrency',
  'keys.lastUsedAt': 'Last Used',
  'keys.lastUsedIP': 'Last Used IP',
  'keys.rateLimitColumn': 'Rate Limit',
  'keys.searchPlaceholder': 'Search name or key...',
  'keys.status.active': 'Active',
  'keys.status.expired': 'Expired',
  'keys.status.inactive': 'Inactive',
  'keys.status.quota_exhausted': 'Quota exhausted',
  'keys.usage': 'Usage',
}

vi.mock('@/api', () => ({
  keysAPI: {
    list: listKeys,
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    toggleStatus: vi.fn(),
  },
  authAPI: {
    getPublicSettings,
  },
  usageAPI: {
    getDashboardApiKeysUsage,
  },
  userGroupsAPI: {
    getAvailable: getAvailableGroups,
    getUserGroupRates,
  },
  userChannelsAPI: {
    getAvailable: getAvailableChannels,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep,
    nextStep,
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/keys' }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const createApiKey = (): ApiKey => ({
  id: 1,
  user_id: 1,
  key: 'sk-test-key',
  name: 'test-key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  last_used_ip: null,
  quota: 0,
  quota_used: 0,
  expires_at: null,
  created_at: '2026-06-27T00:00:00Z',
  updated_at: '2026-06-27T00:00:00Z',
  current_concurrency: 3,
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
})

const AppLayoutStub = {
  template: '<div><slot /></div>',
}

const TablePageLayoutStub = {
  template: `
    <div>
      <slot name="filters" />
      <slot name="actions" />
      <slot name="table" />
      <slot name="pagination" />
    </div>
  `,
}

const DataTableStub = {
  name: 'DataTable',
  props: ['columns', 'data'],
  emits: ['sort'],
  template: `
    <div>
      <div data-test="columns">{{ columns.map((col) => col.key).join(',') }}</div>
      <div data-test="columns-meta">{{ JSON.stringify(columns.map((col) => ({ key: col.key, sortable: !!col.sortable }))) }}</div>
      <button data-test="sort-name" @click="$emit('sort', 'name', 'asc')">
        Sort Name
      </button>
      <div v-for="row in data" :key="row.id">
        <div
          v-if="columns.some((col) => col.key === 'id')"
          data-test="key-id"
        >
          <slot name="cell-id" :value="row.id" :row="row" />
        </div>
        <slot name="cell-name" :value="row.name" :row="row" />
        <div data-test="current-concurrency">
          <slot name="cell-current_concurrency" :value="row.current_concurrency" :row="row" />
        </div>
        <div data-test="actions">
          <slot name="cell-actions" :row="row" />
        </div>
        <div
          v-if="columns.some((col) => col.key === 'last_used_ip')"
          data-test="last-used-ip"
        >
          <slot name="cell-last_used_ip" :value="row.last_used_ip" :row="row" />
        </div>
      </div>
      <slot name="empty" />
    </div>
  `,
}

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"></select>',
}

const SearchInputStub = {
  name: 'SearchInput',
  props: ['modelValue'],
  emits: ['update:modelValue', 'search'],
  template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
}

const PaginationStub = {
  name: 'Pagination',
  props: ['page', 'total', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: `
    <div>
      <button data-test="page-size-50" @click="$emit('update:pageSize', 50)">50</button>
    </div>
  `,
}

const IconStub = {
  props: ['name'],
  template: '<span data-test="icon">{{ name }}</span>',
}

const mountView = async () => {
  const wrapper = mount(KeysView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: PaginationStub,
        BaseDialog: true,
        ConfirmDialog: true,
        EmptyState: true,
        Select: SelectStub,
        SearchInput: SearchInputStub,
        Icon: IconStub,
        UseKeyModal: true,
        EndpointPopover: true,
        BalanceWarningBanner: true,
        GroupBadge: true,
        GroupOptionItem: true,
        LiquidButton: {
          template: '<button v-bind="$attrs"><slot /></button>',
        },
        Teleport: true,
      },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

const visibleColumnKeys = (wrapper: VueWrapper) =>
  wrapper.get('[data-test="columns"]').text().split(',').filter(Boolean)

const visibleColumnMeta = (wrapper: VueWrapper): Array<{ key: string; sortable: boolean }> =>
  JSON.parse(wrapper.get('[data-test="columns-meta"]').text())

const getButtonByText = (wrapper: VueWrapper, text: string) => {
  const button = wrapper.findAll('button').find((item) => item.text().includes(text))
  if (!button) {
    throw new Error(`Button not found: ${text}`)
  }
  return button
}

describe('user KeysView column settings', () => {
  beforeEach(() => {
    localStorage.clear()

    listKeys.mockReset()
    getPublicSettings.mockReset()
    getDashboardApiKeysUsage.mockReset()
    getAvailableGroups.mockReset()
    getUserGroupRates.mockReset()
    getAvailableChannels.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    copyToClipboard.mockReset()
    isCurrentStep.mockReset()
    nextStep.mockReset()

    listKeys.mockResolvedValue({
      items: [createApiKey()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getPublicSettings.mockResolvedValue({})
    getDashboardApiKeysUsage.mockResolvedValue({ stats: {} })
    getAvailableGroups.mockResolvedValue([])
    getUserGroupRates.mockResolvedValue({})
    getAvailableChannels.mockResolvedValue([])
    isCurrentStep.mockReturnValue(false)
  })

  it('uses the current default API key columns with low-frequency columns hidden', async () => {
    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toEqual([
      'name',
      'key',
      'group',
      'usage',
      'status',
      'expires_at',
      'created_at',
      'actions',
    ])
    expect(visibleColumnKeys(wrapper)).not.toContain('rate_limit')
    expect(visibleColumnKeys(wrapper)).not.toContain('last_used_at')
  })

  it('loads every bindable group for key creation instead of only groups exposed by channels', async () => {
    getAvailableGroups.mockResolvedValue([
      {
        id: 24,
        name: '特惠分组',
        description: 'Special offer text group',
        platform: 'openai',
        rate_multiplier: 0.08,
        is_exclusive: false,
        status: 'active',
        subscription_type: 'standard',
      },
      {
        id: 18,
        name: 'GPT-Image（生成图片1 2k）',
        description: 'Image generation group',
        platform: 'openai',
        rate_multiplier: 1,
        is_exclusive: false,
        status: 'active',
        subscription_type: 'standard',
      },
    ])
    getAvailableChannels.mockResolvedValue([])

    const wrapper = await mountView()

    expect(getAvailableGroups).toHaveBeenCalledTimes(1)
    expect(getAvailableChannels).not.toHaveBeenCalled()
    expect((wrapper.vm as unknown as { groupOptions: Array<{ label: string }> }).groupOptions)
      .toEqual(expect.arrayContaining([
        expect.objectContaining({ label: '特惠分组' }),
        expect.objectContaining({ label: 'GPT-Image（生成图片1 2k）' }),
      ]))
  })

  it('shows a hidden column when toggled and persists the preference', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    await getButtonByText(wrapper, 'Rate Limit').trigger('click')
    await nextTick()

    expect(visibleColumnKeys(wrapper)).toContain('rate_limit')
    expect(localStorage.getItem('ssxz-api-key-hidden-columns-v1')).toBe(
      JSON.stringify(['last_used_at'])
    )
  })

  it('restores column preferences from localStorage on mount', async () => {
    localStorage.setItem('ssxz-api-key-hidden-columns-v1', JSON.stringify(['group', 'created_at']))

    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toEqual([
      'name',
      'key',
      'usage',
      'rate_limit',
      'status',
      'expires_at',
      'last_used_at',
      'actions',
    ])
  })

  it('does not include always-visible columns in the toggleable menu', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    await nextTick()

    const columnMenuText = wrapper.get('[data-testid="keys-column-settings-menu"]').text()
    expect(columnMenuText).toContain('Group')
    expect(columnMenuText).toContain('Usage')
    expect(columnMenuText).toContain('Rate Limit')
    expect(columnMenuText).not.toContain('Name')
    expect(columnMenuText).not.toContain('Actions')
  })

  it('marks the key name as sortable', async () => {
    const wrapper = await mountView()

    expect(visibleColumnMeta(wrapper).find((column) => column.key === 'name')?.sortable).toBe(true)
  })

  it('imports Claude keys with separate model mappings instead of a single Opus default', async () => {
    listKeys.mockResolvedValue({
      items: [{
        ...createApiKey(),
        group: {
          id: 11,
          platform: 'anthropic',
        },
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = await mountView()

    await wrapper.get('[data-testid="api-key-ccs-import"]').trigger('click')

    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('model')).toBe('claude-fable-5')
    expect(params.get('haikuModel')).toBe('claude-3-5-haiku')
    expect(params.get('sonnetModel')).toBe('claude-sonnet-5')
    expect(params.get('opusModel')).toBe('claude-opus-4-8')
    expect(JSON.parse(atob(params.get('config') || ''))).toEqual({
      env: { ANTHROPIC_DEFAULT_FABLE_MODEL: 'claude-fable-5' },
    })
  })

  it('does not import Claude model aliases that a restricted key cannot use', async () => {
    listKeys.mockResolvedValue({
      items: [{
        ...createApiKey(),
        allowed_models: ['claude-opus-4-8'],
        group: {
          id: 11,
          platform: 'anthropic',
        },
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = await mountView()

    await wrapper.get('[data-testid="api-key-ccs-import"]').trigger('click')

    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('model')).toBe('claude-opus-4-8')
    expect(params.get('haikuModel')).toBeNull()
    expect(params.get('sonnetModel')).toBeNull()
    expect(params.get('config')).toBeNull()
  })
})
