import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
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
  'keys.noExpiration': 'No expiration',
  'keys.rateLimitColumn': 'Rate Limit',
  'keys.searchPlaceholder': 'Search name or key...',
  'keys.status.active': 'Active',
  'keys.status.disabled': 'Disabled',
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
        <div
          v-if="columns.some((col) => col.key === 'name')"
          data-test="name-key-cell"
        >
          <slot name="cell-name" :value="row.name" :row="row" />
        </div>
        <div data-test="current-concurrency">
          <slot name="cell-current_concurrency" :value="row.current_concurrency" :row="row" />
        </div>
        <div
          v-if="columns.some((col) => col.key === 'status')"
          data-test="status-expiry-cell"
        >
          <slot name="cell-status" :value="row.status" :row="row" />
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

const stubDesktopMatchMedia = () => {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: true,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

const stubMobileMatchMedia = () => {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

const mountView = async ({
  realDataTable = false,
  attachTo,
}: {
  realDataTable?: boolean
  attachTo?: HTMLElement
} = {}) => {
  const stubs: Record<string, unknown> = {
    AppLayout: AppLayoutStub,
    TablePageLayout: TablePageLayoutStub,
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
  }
  if (!realDataTable) stubs.DataTable = DataTableStub

  const wrapper = mount(KeysView, {
    attachTo,
    global: {
      stubs,
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
    stubDesktopMatchMedia()

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
      'group',
      'usage',
      'status',
      'created_at',
      'actions',
    ])
    expect(visibleColumnKeys(wrapper)).not.toContain('rate_limit')
    expect(visibleColumnKeys(wrapper)).not.toContain('last_used_at')
  })

  it('combines the key name and masked API key while keeping copy available', async () => {
    const plaintextKey = 'sk-test-key-1234567890'
    listKeys.mockResolvedValue({
      items: [{ ...createApiKey(), key: plaintextKey }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    const wrapper = await mountView()
    const cell = wrapper.get('[data-test="name-key-cell"]')

    expect(cell.text()).toContain('test-key')
    expect(cell.text()).toContain('sk-test-...7890')

    await cell.get('button').trigger('click')
    expect(copyToClipboard.mock.calls[0]?.[0]).toBe(plaintextKey)
  })

  it('combines status and expiration in the default status column', async () => {
    listKeys.mockResolvedValue({
      items: [{ ...createApiKey(), expires_at: '2030-01-02T03:04:05Z' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    const wrapper = await mountView()
    const cell = wrapper.get('[data-test="status-expiry-cell"]')

    expect(cell.text()).toContain('Active')
    expect(cell.text()).toContain('2030')
  })

  it('renders every combined default field in the native 390px mobile cards without a wide table', async () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/KeysView.vue'), 'utf8')
    stubMobileMatchMedia()
    const viewport = document.createElement('div')
    viewport.style.width = '390px'
    document.body.appendChild(viewport)
    const wrapper = await mountView({ realDataTable: true, attachTo: viewport })

    expect(viewport.style.width).toBe('390px')
    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.findAll('[data-field]').map((field) => field.attributes('data-field'))).toEqual([
      'name',
      'group',
      'usage',
      'status',
      'created_at',
    ])
    expect(wrapper.get('[data-field="name"]').text()).toContain('Name / API Key')
    expect(wrapper.get('[data-field="name"]').text()).toContain('sk-test-key')
    expect(wrapper.get('[data-field="status"]').text()).toContain('Status / Expires')
    expect(wrapper.get('[data-field="status"]').text()).toContain('No expiration')
    expect(wrapper.findAll('[data-field]').every((field) => field.classes().includes('min-w-0'))).toBe(true)
    expect(source).toMatch(/@media \(max-width: 767px\)[\s\S]*\.keys-page-surface\s*\{[\s\S]*overflow-x:\s*clip/)

    wrapper.unmount()
    viewport.remove()
  })

  it('keeps the default desktop table and its six physical columns within about 70rem', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/KeysView.vue'), 'utf8')
    const defaultColumnClasses = [
      'keys-col-name',
      'keys-col-group',
      'keys-col-usage',
      'keys-col-status',
      'keys-col-created',
      'keys-actions-column'
    ]
    const columnWidths = defaultColumnClasses.map((className) => {
      const match = source.match(
        new RegExp(`:deep\\(\\.${className}\\)\\s*\\{[\\s\\S]*?width:\\s*([\\d.]+)rem`)
      )
      expect(match, `missing width for ${className}`).not.toBeNull()
      return Number(match?.[1])
    })
    const tableMinWidth = Number(
      source.match(/:deep\(table\)\s*\{[\s\S]*?min-width:\s*([\d.]+)rem/)?.[1]
    )

    expect(tableMinWidth).toBeLessThanOrEqual(70)
    expect(columnWidths.reduce((total, width) => total + width, 0)).toBeLessThanOrEqual(70)
    expect(source).toContain('keys-name-key-cell')
    expect(source).toContain('keys-status-expiry-cell')
    expect(source).toContain('keys-group-cell')
    expect(source).toMatch(/:deep\(\.table-scroll-container \.table-wrapper\)[\s\S]*overflow-x:\s*auto/)
    expect(source).toMatch(/:deep\(\.table-scroll-container td\)[\s\S]*overflow:\s*hidden/)
    expect(source).toContain(':sticky-first-column="false"')
    expect(source).toContain(':sticky-actions-column="false"')
  })

  it('renders the backend disabled status as a translated label', async () => {
    listKeys.mockResolvedValue({
      items: [{ ...createApiKey(), status: 'disabled' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = await mountView()

    expect((wrapper.vm as unknown as { statusFilterOptions: Array<{ value: string; label: string }> }).statusFilterOptions)
      .toContainEqual({ value: 'disabled', label: 'Disabled' })
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

  it('expands the table only when optional rate-limit or last-used columns are enabled', async () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/KeysView.vue'), 'utf8')
    const wrapper = await mountView()
    const table = wrapper.findComponent({ name: 'DataTable' })

    expect(table.classes()).not.toContain('keys-data-table--rate-limit-visible')
    expect(table.classes()).not.toContain('keys-data-table--last-used-visible')

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    await getButtonByText(wrapper, 'Rate Limit').trigger('click')
    await getButtonByText(wrapper, 'Last Used').trigger('click')
    await nextTick()

    expect(table.classes()).toContain('keys-data-table--rate-limit-visible')
    expect(table.classes()).toContain('keys-data-table--last-used-visible')
    expect(source).toMatch(/keys-data-table--rate-limit-visible[\s\S]*min-width:\s*79rem/)
    expect(source).toMatch(/keys-data-table--last-used-visible[\s\S]*min-width:\s*79rem/)
    expect(source).toMatch(/keys-data-table--rate-limit-visible\.keys-data-table--last-used-visible[\s\S]*min-width:\s*89rem/)
  })

  it('restores column preferences from localStorage on mount', async () => {
    localStorage.setItem('ssxz-api-key-hidden-columns-v1', JSON.stringify(['group', 'created_at']))

    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toEqual([
      'name',
      'usage',
      'rate_limit',
      'status',
      'last_used_at',
      'actions',
    ])
  })

  it('safely migrates old hidden-column settings after combined columns replace key and expiration', async () => {
    localStorage.setItem(
      'ssxz-api-key-hidden-columns-v1',
      JSON.stringify(['key', 'expires_at', 'rate_limit', 'last_used_at', 'removed_column'])
    )

    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toEqual([
      'name',
      'group',
      'usage',
      'status',
      'created_at',
      'actions',
    ])
    expect(localStorage.getItem('ssxz-api-key-hidden-columns-v1')).toBe(
      JSON.stringify(['rate_limit', 'last_used_at'])
    )
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
