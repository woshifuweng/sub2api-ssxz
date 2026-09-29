import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

import type { ApiKey } from '@/types'
import { keysAPI } from '@/api'
import KeysView from '../KeysView.vue'

enableAutoUnmount(afterEach)

const {
  listKeys,
  updateKey,
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
  updateKey: vi.fn(),
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
  'admin.users.columnSettings': 'Column Settings',
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

const HIDDEN_COLUMNS_STORAGE_KEY_V1 = 'ssxz-api-key-hidden-columns-v1'
const HIDDEN_COLUMNS_STORAGE_KEY_V2 = 'ssxz-api-key-hidden-columns-v2'
const LEGACY_HIDE_EXPIRATION_SUMMARY_KEY = 'ssxz-api-key-hide-expiration-summary-legacy-v1'

vi.mock('@/api', () => ({
  keysAPI: {
    list: listKeys,
    create: vi.fn(),
    update: updateKey,
    delete: vi.fn(),
    toggleStatus: vi.fn(),
    reveal: vi.fn(),
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
  useRoute: () => ({ path: '/app/keys' }),
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
  props: { columns: Array, data: Array, selectedKeys: Array, selectable: Boolean },
  emits: ['sort', 'update:selectedKeys'],
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
    AppSectionShell: { template: '<main data-test="ssxz-shell"><slot /></main>' },
    TablePageLayout: TablePageLayoutStub,
    Pagination: PaginationStub,
    BaseDialog: {
      props: ['show'],
      emits: ['close'],
      template: '<div v-if="show"><button type="button" data-test="close-dialog" @click="$emit(\'close\')">close</button><slot /><slot name="footer" /></div>',
    },
    ConfirmDialog: true,
    EmptyState: true,
    Select: SelectStub,
    SearchInput: SearchInputStub,
    Icon: IconStub,
    UseKeyModal: true,
    BulkEditKeysModal: true,
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
    updateKey.mockReset()
    vi.mocked(keysAPI.create).mockReset()
    vi.mocked(keysAPI.reveal).mockReset()
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
    expect(visibleColumnKeys(wrapper)).not.toContain('expires_at')
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

    expect(visibleColumnKeys(wrapper)).not.toContain('expires_at')
    expect(cell.text()).toContain('Active')
    expect(cell.text()).toContain('2030')
    expect(localStorage.getItem(LEGACY_HIDE_EXPIRATION_SUMMARY_KEY)).toBeNull()
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
    expect(source).not.toContain('overflow-x: clip')

    wrapper.unmount()
    viewport.remove()
  })

  it('lets the three English mobile page actions wrap within a 390px container without clipping', async () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/KeysView.vue'), 'utf8')
    stubMobileMatchMedia()
    const viewport = document.createElement('div')
    viewport.style.width = '390px'
    document.body.appendChild(viewport)
    const wrapper = await mountView({ attachTo: viewport })

    const actions = wrapper.get('.keys-page-actions')
    expect(viewport.style.width).toBe('390px')
    expect(actions.element.children).toHaveLength(3)
    expect(actions.text()).toContain('Column Settings')
    expect(actions.text()).toContain('Create API Key')
    expect(source).toMatch(
      /@media \(max-width: 767px\)[\s\S]*\.keys-page-actions\s*\{[\s\S]*width:\s*100%[\s\S]*flex-wrap:\s*wrap[\s\S]*justify-content:\s*flex-end/
    )
    expect(source).toMatch(
      /\.keys-page-actions > :not\(:first-child\)[\s\S]*max-width:\s*100%[\s\S]*flex:\s*1 1 10rem/
    )
    expect(source).not.toContain('overflow-x: clip')

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
    expect(localStorage.getItem(HIDDEN_COLUMNS_STORAGE_KEY_V2)).toBe(
      JSON.stringify(['expires_at', 'last_used_at', 'id', 'current_concurrency', 'last_used_ip'])
    )
  })

  it('keeps expiration as a default-hidden sortable column that can be enabled', async () => {
    const items = [
        { ...createApiKey(), id: 1, expires_at: '2030-01-02T03:04:05Z' },
        { ...createApiKey(), id: 2, expires_at: '2029-01-02T03:04:05Z' },
    ]
    listKeys.mockImplementation(async (_page, _size, filters) => ({
      items: filters.sort_by === 'expires_at' && filters.sort_order === 'asc' ? [...items].reverse() : items,
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
    }))
    const wrapper = await mountView({ realDataTable: true })
    const table = wrapper.findComponent({ name: 'DataTable' })

    expect(wrapper.find('th.keys-col-expires').exists()).toBe(false)

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    await getButtonByText(wrapper, 'Expires').trigger('click')
    await nextTick()

    const expirationHeader = wrapper.get('th.keys-col-expires')
    expect(expirationHeader.text()).toContain('Expires')
    expect(expirationHeader.attributes('aria-sort')).toBe('none')
    expect(wrapper.findAll('td.keys-col-expires').map((cell) => cell.text())).toEqual([
      expect.stringContaining('2030'),
      expect.stringContaining('2029'),
    ])

    await expirationHeader.trigger('click')
    await flushPromises()
    expect(listKeys).toHaveBeenLastCalledWith(1, expect.any(Number), expect.objectContaining({ sort_by: 'expires_at', sort_order: 'asc' }), expect.any(Object))
    expect(expirationHeader.attributes('aria-sort')).toBe('ascending')
    expect(wrapper.findAll('td.keys-col-expires').map((cell) => cell.text())).toEqual([
      expect.stringContaining('2029'),
      expect.stringContaining('2030'),
    ])

    await expirationHeader.trigger('click')
    await flushPromises()
    expect(listKeys).toHaveBeenLastCalledWith(1, expect.any(Number), expect.objectContaining({ sort_by: 'expires_at', sort_order: 'desc' }), expect.any(Object))
    expect(expirationHeader.attributes('aria-sort')).toBe('descending')
    expect(wrapper.findAll('td.keys-col-expires').map((cell) => cell.text())).toEqual([
      expect.stringContaining('2030'),
      expect.stringContaining('2029'),
    ])
    expect(table.classes()).toContain('keys-data-table--optional-columns-1')
  })

  it('expands the table by the number of enabled optional columns', async () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/KeysView.vue'), 'utf8')
    const wrapper = await mountView({ realDataTable: true })
    const table = wrapper.findComponent({ name: 'DataTable' })

    expect(table.classes()).toContain('keys-data-table--optional-columns-0')

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    await getButtonByText(wrapper, 'Rate Limit').trigger('click')
    await nextTick()
    expect(table.classes()).toContain('keys-data-table--optional-columns-1')

    await getButtonByText(wrapper, 'Last Used').trigger('click')
    await nextTick()
    expect(table.classes()).toContain('keys-data-table--optional-columns-2')

    await getButtonByText(wrapper, 'Expires').trigger('click')
    await nextTick()
    expect(table.classes()).toContain('keys-data-table--optional-columns-3')

    expect(source).toMatch(/keys-data-table--optional-columns-1[\s\S]*min-width:\s*79rem/)
    expect(source).toMatch(/keys-data-table--optional-columns-2[\s\S]*min-width:\s*89rem/)
    expect(source).toMatch(/keys-data-table--optional-columns-3[\s\S]*min-width:\s*99rem/)
  })

  it('restores column preferences from localStorage on mount', async () => {
    localStorage.setItem(HIDDEN_COLUMNS_STORAGE_KEY_V2, JSON.stringify(['group', 'created_at']))

    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toEqual([
      'id',
      'name',
      'current_concurrency',
      'usage',
      'rate_limit',
      'status',
      'expires_at',
      'last_used_at',
      'last_used_ip',
      'actions',
    ])
  })

  it('migrates an old preference that hid status but kept expiration visible', async () => {
    listKeys.mockResolvedValue({
      items: [{ ...createApiKey(), expires_at: '2030-01-02T03:04:05Z' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    localStorage.setItem(HIDDEN_COLUMNS_STORAGE_KEY_V1, JSON.stringify(['status']))

    const wrapper = await mountView({ realDataTable: true })

    expect(wrapper.get('td.keys-col-status').text()).toContain('Active')
    expect(wrapper.get('td.keys-col-status').text()).not.toContain('2030')
    expect(wrapper.get('td.keys-col-expires').text()).toContain('2030')
    expect(localStorage.getItem(HIDDEN_COLUMNS_STORAGE_KEY_V2)).toBe(JSON.stringify([]))
    expect(localStorage.getItem(LEGACY_HIDE_EXPIRATION_SUMMARY_KEY)).toBe('true')
  })

  it('migrates an old hidden expiration preference without exposing its summary', async () => {
    listKeys.mockResolvedValue({
      items: [{ ...createApiKey(), expires_at: '2030-01-02T03:04:05Z' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    localStorage.setItem(HIDDEN_COLUMNS_STORAGE_KEY_V1, JSON.stringify(['expires_at']))

    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toContain('status')
    expect(visibleColumnKeys(wrapper)).not.toContain('expires_at')
    expect(wrapper.get('[data-test="status-expiry-cell"]').text()).not.toContain('2030')
    expect(localStorage.getItem(HIDDEN_COLUMNS_STORAGE_KEY_V2)).toBe(
      JSON.stringify(['expires_at'])
    )
    expect(localStorage.getItem(LEGACY_HIDE_EXPIRATION_SUMMARY_KEY)).toBe('true')
  })

  it('clears the legacy summary marker when expiration is toggled and never duplicates the date', async () => {
    listKeys.mockResolvedValue({
      items: [{ ...createApiKey(), expires_at: '2030-01-02T03:04:05Z' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    localStorage.setItem(HIDDEN_COLUMNS_STORAGE_KEY_V1, JSON.stringify(['expires_at']))
    const wrapper = await mountView({ realDataTable: true })

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    await getButtonByText(wrapper, 'Expires').trigger('click')
    await nextTick()

    expect(wrapper.get('td.keys-col-status').text()).not.toContain('2030')
    expect(wrapper.get('td.keys-col-expires').text()).toContain('2030')
    expect(wrapper.get('tbody').text().match(/2030/g)).toHaveLength(1)
    expect(localStorage.getItem(LEGACY_HIDE_EXPIRATION_SUMMARY_KEY)).toBeNull()

    await getButtonByText(wrapper, 'Expires').trigger('click')
    await nextTick()

    expect(wrapper.find('td.keys-col-expires').exists()).toBe(false)
    expect(wrapper.get('td.keys-col-status').text()).toContain('2030')
    expect(wrapper.get('tbody').text().match(/2030/g)).toHaveLength(1)
    expect(localStorage.getItem(LEGACY_HIDE_EXPIRATION_SUMMARY_KEY)).toBeNull()
  })

  it('does not include always-visible columns in the toggleable menu', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    await nextTick()

    const columnMenuText = wrapper.get('[data-testid="keys-column-settings-menu"]').text()
    expect(columnMenuText).toContain('Group')
    expect(columnMenuText).toContain('Usage')
    expect(columnMenuText).toContain('Rate Limit')
    expect(columnMenuText).toContain('Expires')
    expect(columnMenuText).not.toContain('Name')
    expect(columnMenuText).not.toContain('Status / Expires')
    expect(columnMenuText).not.toContain('Actions')
  })

  it('marks the key name as sortable', async () => {
    const wrapper = await mountView()

    expect(visibleColumnMeta(wrapper).find((column) => column.key === 'name')?.sortable).toBe(true)
  })

  it('exposes upstream diagnostic columns and bulk edit in the real app shell', async () => {
    const wrapper = await mountView({ realDataTable: true })
    expect(wrapper.find('[data-test="ssxz-shell"]').exists()).toBe(true)
    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')
    for (const label of ['ID', 'Current Concurrency', 'Last Used IP']) {
      await getButtonByText(wrapper, label).trigger('click')
    }
    expect(wrapper.get('td.keys-col-id').text()).toBe('1')
    expect(wrapper.get('td.keys-col-concurrency').text()).toBe('3')
    expect(wrapper.get('td.keys-col-last-ip').text()).toBe('-')
    await wrapper.get('th.keys-col-concurrency').trigger('click')
    await flushPromises()
    expect(listKeys).toHaveBeenLastCalledWith(1, expect.any(Number), expect.objectContaining({ sort_by: 'current_concurrency', sort_order: 'asc' }), expect.any(Object))
    await wrapper.get('[data-test="select-row"]').setValue(true)
    await wrapper.get('[data-test="bulk-edit-keys"]').trigger('click')
    expect(wrapper.findComponent({ name: 'BulkEditKeysModal' }).props('show')).toBe(true)
    expect(wrapper.findComponent({ name: 'BulkEditKeysModal' }).props('selectedKeys')).toHaveLength(1)
    await wrapper.get('[data-test="page-size-50"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="bulk-edit-keys"]').exists()).toBe(false)
  })

  it('keeps every custom group when editing a multi-group key', async () => {
    getAvailableGroups.mockResolvedValue([24, 18].map((id) => ({ id, name: `Custom ${id}`, platform: 'openai', subscription_type: 'standard', status: 'active', rate_multiplier: 1 })))
    listKeys.mockResolvedValue({ items: [{ ...createApiKey(), group_id: 24, group_ids: [24, 18] }], total: 1, pages: 1 })
    const wrapper = await mountView()
    await getButtonByText(wrapper, 'common.edit').trigger('click')
    await wrapper.get('[data-tour="key-form-name"]').setValue('Renamed')
    await wrapper.get('#key-form').trigger('submit')
    await flushPromises()
    expect(updateKey).toHaveBeenCalledWith(1, expect.objectContaining({ name: 'Renamed', group_id: 24, group_ids: [24, 18] }))
  })

  it.each(['quota_exhausted', 'expired'] as const)('does not disable a %s key when only its name changes', async (status) => {
    getAvailableGroups.mockResolvedValue([{ id: 24, name: 'Custom', platform: 'openai', subscription_type: 'standard', status: 'active', rate_multiplier: 1 }])
    listKeys.mockResolvedValue({ items: [{ ...createApiKey(), status, group_id: 24 }], total: 1, pages: 1 })
    const wrapper = await mountView()
    await getButtonByText(wrapper, 'common.edit').trigger('click')
    await wrapper.get('#key-form').trigger('submit')
    await flushPromises()
    expect(updateKey).toHaveBeenCalledOnce()
    expect(updateKey.mock.calls[0][1]).not.toHaveProperty('status')
  })

  it('aborts an outstanding list request when leaving the page', async () => {
    let resolveList!: (value: unknown) => void
    listKeys.mockReturnValue(new Promise((resolve) => { resolveList = resolve }))
    const wrapper = await mountView()
    const signal = listKeys.mock.calls[0][3].signal as AbortSignal
    wrapper.unmount()
    expect(signal.aborted).toBe(true)
    resolveList({ items: [createApiKey()], total: 1, pages: 1 })
    await flushPromises()
    expect(getDashboardApiKeysUsage).not.toHaveBeenCalled()
  })

  it('does not reveal a key into a closed dialog', async () => {
    listKeys.mockResolvedValue({ items: [{ ...createApiKey(), key: '[redacted]' }], total: 1, pages: 1 })
    let resolveReveal!: (value: { key: string }) => void
    vi.mocked(keysAPI.reveal).mockReturnValue(new Promise((resolve) => { resolveReveal = resolve }))
    const wrapper = await mountView()
    await wrapper.get('[aria-label="keys.useKey"]').trigger('click')
    const dialog = wrapper.findComponent({ name: 'UseKeyModal' })
    dialog.vm.$emit('close')
    resolveReveal({ key: 'sk-secret' })
    await flushPromises()
    expect(dialog.props('show')).toBe(false)
    expect(dialog.props('apiKey')).not.toBe('sk-secret')
  })

  it('ignores stale plaintext after reopening the same key', async () => {
    listKeys.mockResolvedValue({ items: [{ ...createApiKey(), key: '[redacted]' }], total: 1, pages: 1 })
    let resolveOld!: (value: { key: string }) => void
    vi.mocked(keysAPI.reveal)
      .mockReturnValueOnce(new Promise((resolve) => { resolveOld = resolve }))
      .mockResolvedValueOnce({ key: 'sk-current' })
    const wrapper = await mountView()
    await wrapper.get('[aria-label="keys.useKey"]').trigger('click')
    const dialog = wrapper.findComponent({ name: 'UseKeyModal' })
    dialog.vm.$emit('close')
    await wrapper.get('[aria-label="keys.useKey"]').trigger('click')
    await flushPromises()
    resolveOld({ key: 'sk-stale' })
    await flushPromises()
    expect(dialog.props('apiKey')).toBe('sk-current')
  })

  it('keeps only failed bulk-edit rows selected and reconciles refreshed results', async () => {
    listKeys.mockResolvedValue({ items: [createApiKey(), { ...createApiKey(), id: 2 }], total: 2, pages: 1 })
    const wrapper = await mountView({ realDataTable: true })
    await wrapper.get('[data-test="select-all"]').setValue(true)
    const dialog = wrapper.findComponent({ name: 'BulkEditKeysModal' })
    dialog.vm.$emit('updated', [1])
    await flushPromises()
    expect(dialog.props('selectedKeys').map((key: ApiKey) => key.id)).toEqual([2])
    listKeys.mockResolvedValue({ items: [createApiKey()], total: 1, pages: 1 })
    await wrapper.get('[aria-label="Refresh"]').trigger('click')
    await flushPromises()
    expect(dialog.props('selectedKeys')).toEqual([])
    expect(wrapper.find('[data-test="bulk-edit-keys"]').exists()).toBe(false)
  })

  it('clears bulk selection on filters and cannot select stale rows while loading', async () => {
    const wrapper = await mountView({ realDataTable: true })
    await wrapper.get('[data-test="select-all"]').setValue(true)
    listKeys.mockReturnValue(new Promise(() => {}))
    wrapper.findAllComponents({ name: 'Select' })[0].vm.$emit('update:modelValue', 24)
    await nextTick()
    wrapper.findComponent({ name: 'DataTable' }).vm.$emit('selectionChange', [1])
    await nextTick()
    expect(wrapper.find('[data-test="bulk-edit-keys"]').exists()).toBe(false)
    expect(listKeys).toHaveBeenLastCalledWith(1, expect.any(Number), expect.objectContaining({ group_id: 24 }), expect.any(Object))
  })

  it('shows usage-load failures without retaining old usage totals', async () => {
    getDashboardApiKeysUsage.mockResolvedValueOnce({ stats: { 1: { total_actual_cost: 123.45 } } })
    const wrapper = await mountView({ realDataTable: true })
    expect(wrapper.get('td.keys-col-usage').text()).toContain('123.45')
    getDashboardApiKeysUsage.mockRejectedValueOnce(new Error('unavailable'))
    await wrapper.get('[aria-label="Refresh"]').trigger('click')
    await flushPromises()
    expect(showError).toHaveBeenCalledWith('keys.failedToLoad')
    expect(wrapper.get('td.keys-col-usage').text()).not.toContain('123.45')
  })

  it('closes column choices with Escape and restores keyboard focus', async () => {
    const host = document.createElement('div')
    document.body.append(host)
    const wrapper = await mountView({ attachTo: host })
    const trigger = wrapper.get('[data-testid="keys-column-settings-trigger"]')
    await trigger.trigger('click')
    const column = wrapper.get('[data-column-key="id"]')
    expect(column.attributes('role')).toBe('checkbox')
    expect(column.attributes('aria-checked')).toBe('false')
    await column.trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[data-testid="keys-column-settings-menu"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
    wrapper.unmount()
    host.remove()
  })

  it('imports Grok using the upstream grokbuild client while respecting model restrictions', async () => {
    listKeys.mockResolvedValue({ items: [{ ...createApiKey(), group: { id: 11, platform: 'grok' }, allowed_models: ['grok-custom'] }], total: 1, pages: 1 })
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = await mountView()
    await wrapper.get('[data-testid="api-key-ccs-import"]').trigger('click')
    await flushPromises()
    const params = new URLSearchParams(String(openSpy.mock.calls[0]?.[0]).split('?')[1])
    expect(params.get('app')).toBe('grokbuild')
    expect(params.get('endpoint')).toMatch(/\/v1$/)
    expect(params.get('model')).toBe('grok-custom')
  })

  it('does not guess a CC Switch client for mixed custom providers', async () => {
    listKeys.mockResolvedValue({ items: [{ ...createApiKey(), groups: [{ id: 1, platform: 'anthropic' }, { id: 5, platform: 'deepseek' }] }], total: 1, pages: 1 })
    const wrapper = await mountView()
    const button = wrapper.get<HTMLButtonElement>('[data-testid="api-key-ccs-import"]')
    expect(button.element.disabled).toBe(true)
    await button.trigger('click')
    expect(keysAPI.reveal).not.toHaveBeenCalled()
  })

  it('cannot let a stale list response replace a newer filtered result', async () => {
    let resolveOld!: (value: unknown) => void
    listKeys.mockReturnValueOnce(new Promise((resolve) => { resolveOld = resolve }))
    const wrapper = await mountView({ realDataTable: true })
    listKeys.mockResolvedValueOnce({ items: [{ ...createApiKey(), name: 'new-result' }], total: 1, pages: 1 })
    wrapper.findAllComponents({ name: 'Select' })[0].vm.$emit('update:modelValue', 24)
    await flushPromises()
    resolveOld({ items: [{ ...createApiKey(), name: 'stale-result' }], total: 1, pages: 1 })
    await flushPromises()
    expect(wrapper.get('tbody').text()).toContain('new-result')
    expect(wrapper.get('tbody').text()).not.toContain('stale-result')
  })

  it('keeps a quota reset active status when saving the open edit form', async () => {
    getAvailableGroups.mockResolvedValue([{ id: 24, name: 'Custom', platform: 'openai', subscription_type: 'standard', status: 'active', rate_multiplier: 1 }])
    const key = { ...createApiKey(), group_id: 24, quota: 10, quota_used: 10 }
    listKeys.mockResolvedValueOnce({ items: [{ ...key, status: 'quota_exhausted' }], total: 1, pages: 1 })
    updateKey.mockResolvedValue({ ...key, status: 'active', quota_used: 0 })
    listKeys.mockResolvedValue({ items: [{ ...key, status: 'active', quota_used: 0 }], total: 1, pages: 1 })
    const wrapper = await mountView()
    await getButtonByText(wrapper, 'common.edit').trigger('click')
    await wrapper.get('[title="keys.resetQuotaUsed"]').trigger('click')
    const confirm = wrapper.findAllComponents({ name: 'ConfirmDialog' }).find((dialog) => dialog.props('show'))!
    confirm.vm.$emit('confirm')
    await flushPromises()
    await wrapper.get('#key-form').trigger('submit')
    await flushPromises()
    expect(updateKey).toHaveBeenLastCalledWith(1, expect.objectContaining({ status: 'active' }))
  })

  it.each(['before reset', 'during reset', 'during refresh'] as const)('preserves an explicit inactive choice made %s', async (timing) => {
    const key = { ...createApiKey(), group_id: 24, quota: 10, quota_used: 10 }
    const recoveredKey = { ...key, quota_used: 0 }
    getAvailableGroups.mockResolvedValue([{ id: 24, name: 'Custom', platform: 'openai', subscription_type: 'standard', status: 'active', rate_multiplier: 1 }])
    listKeys.mockResolvedValueOnce({ items: [key], total: 1, pages: 1 })
    let resolveReset!: (value: ApiKey) => void
    let resolveRefresh!: (value: unknown) => void
    updateKey.mockReturnValueOnce(new Promise((resolve) => { resolveReset = resolve }))
    listKeys.mockReturnValueOnce(new Promise((resolve) => { resolveRefresh = resolve }))
    const wrapper = await mountView()
    await getButtonByText(wrapper, 'common.edit').trigger('click')
    const statusSelect = wrapper.findAllComponents({ name: 'Select' }).find((select) => select.attributes('aria-label') === 'keys.statusLabel' && select.props('modelValue') === 'active')!
    const chooseInactive = async () => {
      statusSelect.vm.$emit('update:modelValue', 'inactive')
      await nextTick()
    }
    if (timing === 'before reset') await chooseInactive()
    await wrapper.get('[title="keys.resetQuotaUsed"]').trigger('click')
    wrapper.findAllComponents({ name: 'ConfirmDialog' }).find((dialog) => dialog.props('show'))!.vm.$emit('confirm')
    if (timing === 'during reset') await chooseInactive()
    resolveReset(recoveredKey)
    await flushPromises()
    if (timing === 'during refresh') await chooseInactive()
    expect(statusSelect.props('modelValue')).toBe('inactive')
    resolveRefresh({ items: [recoveredKey], total: 1, pages: 1 })
    await flushPromises()
    expect(statusSelect.props('modelValue')).toBe('inactive')
    updateKey.mockResolvedValue({ ...recoveredKey, status: 'inactive' })
    await wrapper.get('#key-form').trigger('submit')
    await flushPromises()
    expect(updateKey).toHaveBeenLastCalledWith(1, expect.objectContaining({ status: 'inactive' }))
  })

  it('preserves an explicitly chosen inactive status on a quota-exhausted key', async () => {
    const recoveredKey = { ...createApiKey(), group_id: 24, quota: 10, quota_used: 0 }
    getAvailableGroups.mockResolvedValue([{ id: 24, name: 'Custom', platform: 'openai', subscription_type: 'standard', status: 'active', rate_multiplier: 1 }])
    listKeys.mockResolvedValueOnce({ items: [{ ...recoveredKey, status: 'quota_exhausted', quota_used: 10 }], total: 1, pages: 1 })
    listKeys.mockResolvedValue({ items: [recoveredKey], total: 1, pages: 1 })
    updateKey.mockResolvedValue(recoveredKey)
    const wrapper = await mountView()
    await getButtonByText(wrapper, 'common.edit').trigger('click')
    const statusSelect = wrapper.findAllComponents({ name: 'Select' }).find((select) => select.props('modelValue') === 'inactive')!
    statusSelect.vm.$emit('update:modelValue', 'active')
    await nextTick()
    statusSelect.vm.$emit('update:modelValue', 'inactive')
    await nextTick()
    await wrapper.get('[title="keys.resetQuotaUsed"]').trigger('click')
    wrapper.findAllComponents({ name: 'ConfirmDialog' }).find((dialog) => dialog.props('show'))!.vm.$emit('confirm')
    await flushPromises()
    expect(statusSelect.props('modelValue')).toBe('inactive')
    await wrapper.get('#key-form').trigger('submit')
    await flushPromises()
    expect(updateKey).toHaveBeenLastCalledWith(1, expect.objectContaining({ status: 'inactive' }))
  })

  it('does not restore a closed edit dialog when its rate-limit reset finishes', async () => {
    listKeys.mockResolvedValue({ items: [{ ...createApiKey(), group_id: 24, rate_limit_5h: 10 }], total: 1, pages: 1 })
    let resolveReset!: (value: ApiKey) => void
    updateKey.mockReturnValue(new Promise((resolve) => { resolveReset = resolve }))
    const wrapper = await mountView()
    await getButtonByText(wrapper, 'common.edit').trigger('click')
    await getButtonByText(wrapper, 'keys.resetRateLimitUsage').trigger('click')
    const confirm = wrapper.findAllComponents({ name: 'ConfirmDialog' }).find((dialog) => dialog.props('show'))!
    confirm.vm.$emit('confirm')
    await wrapper.get('[data-test="close-dialog"]').trigger('click')
    resolveReset(createApiKey())
    await flushPromises()
    expect(wrapper.find('#key-form').exists()).toBe(false)
    expect(showError).not.toHaveBeenCalled()
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

  describe('create provider selection', () => {
    const platforms = ['anthropic', 'openai', 'kimi', 'zhipu', 'deepseek', 'minimax', 'gemini', 'grok', 'antigravity', 'composite', 'opencode_go']
    const availableGroups = platforms.map((platform, index) => ({
      id: index + 1,
      // Deliberately ambiguous names: classification must follow the platform.
      name: `Shared group ${index + 1}`,
      platform,
      rate_multiplier: 1,
      subscription_type: 'standard',
      status: 'active',
    }))
    const groupSelect = (wrapper: VueWrapper) => wrapper.findComponent('[data-tour="key-form-group"]')
    const optionIds = (wrapper: VueWrapper) => groupSelect(wrapper).props('options').map((option: { value: number }) => option.value)
    const chooseProvider = (wrapper: VueWrapper, value: string) => wrapper.get(`input[name="key-provider"][value="${value}"]`).setValue()
    const openCreate = async () => {
      const wrapper = await mountView()
      await wrapper.get('[data-tour="keys-create-btn"]').trigger('click')
      return wrapper
    }

    beforeEach(() => {
      getAvailableGroups.mockResolvedValue(availableGroups)
    })

    it('classifies all configured platforms and retains the complete table filter', async () => {
      const wrapper = await openCreate()
      expect(wrapper.findAll('input[name="key-provider"]')).toHaveLength(4)
      expect(optionIds(wrapper)).toEqual([1])
      await chooseProvider(wrapper, 'openai')
      expect(optionIds(wrapper)).toEqual([2])
      await chooseProvider(wrapper, 'domestic')
      expect(optionIds(wrapper)).toEqual([3, 4, 5, 6])
      await chooseProvider(wrapper, 'other')
      expect(optionIds(wrapper)).toEqual([7, 8, 9, 10, 11])
      expect(wrapper.findAllComponents({ name: 'Select' })[0].props('options')).toHaveLength(13)
    })

    it('keeps multi-icon provider cards contained inside their card', async () => {
      const wrapper = await openCreate()
      const domesticCard = wrapper.get('input[name="key-provider"][value="domestic"]').element.closest('label')
      const iconRow = domesticCard?.querySelector<HTMLElement>('.key-provider-icons')

      expect(iconRow).not.toBeNull()
      expect(iconRow?.className).toContain('min-w-0')
      expect(iconRow?.className).toContain('max-w-full')
      expect(iconRow?.className).toContain('flex-wrap')
    })

    it('clears the previous group on provider change and submits only the newly selected group', async () => {
      const wrapper = await openCreate()
      await wrapper.get('[data-tour="key-form-name"]').setValue('My key')
      await groupSelect(wrapper).vm.$emit('update:modelValue', 1)
      await chooseProvider(wrapper, 'domestic')
      expect(groupSelect(wrapper).props('modelValue')).toBeNull()
      await wrapper.get('#key-form').trigger('submit')
      expect(keysAPI.create).not.toHaveBeenCalled()
      expect(showError).toHaveBeenCalledWith('keys.groupRequired')

      await groupSelect(wrapper).vm.$emit('update:modelValue', 5)
      vi.mocked(keysAPI.create).mockResolvedValue({ ...createApiKey(), group_id: 5 })
      await wrapper.get('#key-form').trigger('submit')
      await flushPromises()
      expect(keysAPI.create).toHaveBeenCalledOnce()
      expect(vi.mocked(keysAPI.create).mock.calls[0].slice(0, 2)).toEqual(['My key', 5])
    })

    it('defaults to a provider with available groups and disables empty categories', async () => {
      getAvailableGroups.mockResolvedValue([availableGroups[5]])
      const wrapper = await openCreate()
      expect(wrapper.get<HTMLInputElement>('input[value="domestic"]').element.checked).toBe(true)
      expect(wrapper.get<HTMLInputElement>('input[value="anthropic"]').element.disabled).toBe(true)
      expect(optionIds(wrapper)).toEqual([6])
    })

    it('shows the empty state when no groups are available', async () => {
      getAvailableGroups.mockResolvedValue([])
      const wrapper = await openCreate()
      expect(wrapper.get('[data-tour="key-form-provider"]').text()).toContain('common.noGroupsAvailable')
      expect(optionIds(wrapper)).toEqual([])
      expect(wrapper.findAll<HTMLInputElement>('input[name="key-provider"]').every((input) => input.element.disabled)).toBe(true)
    })

    it('selects an available provider when groups arrive after opening', async () => {
      let resolveGroups!: (value: typeof availableGroups) => void
      getAvailableGroups.mockReturnValue(new Promise((resolve) => { resolveGroups = resolve }))
      const wrapper = await openCreate()
      resolveGroups([availableGroups[1]])
      await flushPromises()
      expect(wrapper.get<HTMLInputElement>('input[value="openai"]').element.checked).toBe(true)
      expect(optionIds(wrapper)).toEqual([2])
    })

    it('resets provider and group when reopening create, and preserves edit options', async () => {
      const wrapper = await openCreate()
      await chooseProvider(wrapper, 'domestic')
      await groupSelect(wrapper).vm.$emit('update:modelValue', 5)
      await wrapper.get('[data-test="close-dialog"]').trigger('click')
      await wrapper.get('[data-tour="keys-create-btn"]').trigger('click')
      expect(optionIds(wrapper)).toEqual([1])
      expect(groupSelect(wrapper).props('modelValue')).toBeNull()
      await wrapper.get('[data-test="close-dialog"]').trigger('click')
      await getButtonByText(wrapper, 'common.edit').trigger('click')
      expect(wrapper.find('[data-tour="key-form-provider"]').exists()).toBe(false)
      expect(optionIds(wrapper)).toHaveLength(11)
    })

    it('creates a multi-group key only from compatible standard groups', async () => {
      getAvailableGroups.mockResolvedValue([...availableGroups, { ...availableGroups[0], id: 12, name: 'Custom Anthropic' }])
      const wrapper = await openCreate()
      await wrapper.get('[data-tour="key-form-name"]').setValue('Multi group')
      groupSelect(wrapper).vm.$emit('update:modelValue', 1)
      await nextTick()
      await wrapper.get('[data-test="key-group-12"]').setValue(true)
      vi.mocked(keysAPI.create).mockResolvedValue({ ...createApiKey(), group_ids: [1, 12] })
      await wrapper.get('#key-form').trigger('submit')
      await flushPromises()
      expect(vi.mocked(keysAPI.create).mock.calls[0].slice(0, 3)).toEqual(['Multi group', 1, [1, 12]])
    })

    it('preserves compatible secondary groups and rejects incompatible primary changes', async () => {
      getAvailableGroups.mockResolvedValue([...availableGroups, ...[12, 13].map((id) => ({ ...availableGroups[0], id, name: `Custom ${id}` }))])
      listKeys.mockResolvedValue({ items: [{ ...createApiKey(), group_id: 1, group_ids: [1, 12] }], total: 1, pages: 1 })
      const wrapper = await mountView()
      await getButtonByText(wrapper, 'common.edit').trigger('click')
      expect(groupSelect(wrapper).props('options').find((option: { value: number }) => option.value === 2).disabled).toBe(true)
      groupSelect(wrapper).vm.$emit('update:modelValue', 2)
      await nextTick()
      expect(groupSelect(wrapper).props('modelValue')).toBe(1)
      groupSelect(wrapper).vm.$emit('update:modelValue', 13)
      await nextTick()
      await wrapper.get('#key-form').trigger('submit')
      await flushPromises()
      expect(updateKey).toHaveBeenCalledWith(1, expect.objectContaining({ group_id: 13, group_ids: [13, 12] }))
    })

    it.each([
      { platform: 'deepseek', subscription_type: 'standard', status: 'active' },
      { platform: 'anthropic', subscription_type: 'subscription', status: 'active' },
      { platform: 'anthropic', subscription_type: 'standard', status: 'inactive' },
    ])('disables incompatible or unavailable extra groups: %j', async (invalidGroup) => {
      getAvailableGroups.mockResolvedValue([...availableGroups, { ...availableGroups[0], ...invalidGroup, id: 12 }])
      const wrapper = await openCreate()
      groupSelect(wrapper).vm.$emit('update:modelValue', 1)
      await nextTick()
      const checkbox = wrapper.get<HTMLInputElement>('[data-test="key-group-12"]')
      expect(checkbox.element.disabled).toBe(true)
      await checkbox.trigger('click')
      expect(checkbox.element.checked).toBe(false)
      expect(keysAPI.create).not.toHaveBeenCalled()
    })

    it.each([
      { platform: 'deepseek', subscription_type: 'standard', status: 'active' },
      { platform: 'anthropic', subscription_type: 'subscription', status: 'active' },
      null,
    ])('rejects invalid existing multi-group selections before saving: %j', async (invalidGroup) => {
      getAvailableGroups.mockResolvedValue(invalidGroup ? [...availableGroups, { ...availableGroups[0], ...invalidGroup, id: 12 }] : availableGroups)
      listKeys.mockResolvedValue({ items: [{ ...createApiKey(), group_id: 1, group_ids: [1, 12] }], total: 1, pages: 1 })
      const wrapper = await mountView()
      await getButtonByText(wrapper, 'common.edit').trigger('click')
      expect(wrapper.get<HTMLButtonElement>('[data-tour="key-form-submit"]').element.disabled).toBe(true)
      await wrapper.get('#key-form').trigger('submit')
      expect(updateKey).not.toHaveBeenCalled()
      expect(showError).toHaveBeenCalled()
    })

    it('allows subscription groups alone but never combines them with other groups', async () => {
      getAvailableGroups.mockResolvedValue([{ ...availableGroups[0], subscription_type: 'subscription' }, { ...availableGroups[0], id: 12 }])
      const wrapper = await openCreate()
      groupSelect(wrapper).vm.$emit('update:modelValue', 1)
      await nextTick()
      expect(wrapper.get<HTMLInputElement>('[data-test="key-group-12"]').element.disabled).toBe(true)
      await wrapper.get('[data-tour="key-form-name"]').setValue('Subscription key')
      vi.mocked(keysAPI.create).mockResolvedValue({ ...createApiKey(), group_id: 1 })
      await wrapper.get('#key-form').trigger('submit')
      await flushPromises()
      expect(vi.mocked(keysAPI.create).mock.calls[0].slice(0, 3)).toEqual(['Subscription key', 1, [1]])
    })

    it('does not treat the domestic provider category as a single backend platform', async () => {
      const wrapper = await openCreate()
      await chooseProvider(wrapper, 'domestic')
      groupSelect(wrapper).vm.$emit('update:modelValue', 5)
      await nextTick()
      expect(wrapper.get<HTMLInputElement>('[data-test="key-group-6"]').element.disabled).toBe(true)
      expect(wrapper.get<HTMLInputElement>('[data-test="key-group-3"]').element.disabled).toBe(true)
    })

    it('rejects inactive and unknown primary group values even if a selector emits them', async () => {
      getAvailableGroups.mockResolvedValue([...availableGroups, { ...availableGroups[0], id: 12, status: 'inactive' }])
      const wrapper = await openCreate()
      expect(groupSelect(wrapper).props('options').find((option: { value: number }) => option.value === 12).disabled).toBe(true)
      for (const id of [12, 999]) {
        groupSelect(wrapper).vm.$emit('update:modelValue', id)
        await nextTick()
        expect(groupSelect(wrapper).props('modelValue')).toBeNull()
        await wrapper.get('#key-form').trigger('submit')
        expect(keysAPI.create).not.toHaveBeenCalled()
      }
    })

    it('lets an unavailable existing binding be explicitly cleared and replaced', async () => {
      listKeys.mockResolvedValue({ items: [{ ...createApiKey(), group_id: 1, group_ids: [1, 999] }], total: 1, pages: 1 })
      const wrapper = await mountView()
      await getButtonByText(wrapper, 'common.edit').trigger('click')
      expect(groupSelect(wrapper).attributes()).toHaveProperty('clearable')
      expect(wrapper.get<HTMLButtonElement>('[data-tour="key-form-submit"]').element.disabled).toBe(true)
      groupSelect(wrapper).vm.$emit('update:modelValue', null)
      await nextTick()
      groupSelect(wrapper).vm.$emit('update:modelValue', 2)
      await nextTick()
      await wrapper.get('#key-form').trigger('submit')
      await flushPromises()
      expect(updateKey).toHaveBeenCalledWith(1, expect.objectContaining({ group_id: 2, group_ids: [2] }))
    })

    it('prevents duplicate submissions and closing a saving dialog', async () => {
      let resolveCreate!: (value: ApiKey) => void
      vi.mocked(keysAPI.create).mockReturnValue(new Promise((resolve) => { resolveCreate = resolve }))
      const wrapper = await openCreate()
      await wrapper.get('[data-tour="key-form-name"]').setValue('Pending key')
      groupSelect(wrapper).vm.$emit('update:modelValue', 1)
      await wrapper.get('#key-form').trigger('submit')
      await wrapper.get('#key-form').trigger('submit')
      expect(wrapper.get<HTMLFieldSetElement>('#key-form > fieldset').element.disabled).toBe(true)
      expect(wrapper.get<HTMLButtonElement>('[data-tour="key-form-submit"]').element.disabled).toBe(true)
      await wrapper.get('[data-test="close-dialog"]').trigger('click')
      expect(wrapper.find('#key-form').exists()).toBe(true)
      expect(keysAPI.create).toHaveBeenCalledOnce()
      resolveCreate(createApiKey())
      await flushPromises()
      expect(wrapper.find('#key-form').exists()).toBe(false)
    })

    it('does not reopen dialogs or refresh after creation finishes on an unmounted page', async () => {
      let resolveCreate!: (value: ApiKey) => void
      vi.mocked(keysAPI.create).mockReturnValue(new Promise((resolve) => { resolveCreate = resolve }))
      const wrapper = await openCreate()
      await wrapper.get('[data-tour="key-form-name"]').setValue('Pending key')
      groupSelect(wrapper).vm.$emit('update:modelValue', 1)
      await wrapper.get('#key-form').trigger('submit')
      wrapper.unmount()
      resolveCreate(createApiKey())
      await flushPromises()
      expect(listKeys).toHaveBeenCalledOnce()
      expect(showSuccess).not.toHaveBeenCalled()
    })
  })
})
