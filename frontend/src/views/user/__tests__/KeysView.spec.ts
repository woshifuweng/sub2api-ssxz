import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, keysAPI, authAPI, usageAPI, userGroupsAPI, appStore, onboardingStore, clipboardCopy } = vi.hoisted(() => ({
  routeState: {
    path: '/app/keys'
  },
  keysAPI: {
    list: vi.fn(),
    reveal: vi.fn(),
    toggleStatus: vi.fn(),
    update: vi.fn(),
    create: vi.fn(),
    delete: vi.fn()
  },
  authAPI: {
    getPublicSettings: vi.fn()
  },
  usageAPI: {
    getDashboardApiKeysUsage: vi.fn()
  },
  userGroupsAPI: {
    getAvailable: vi.fn(),
    getUserGroupRates: vi.fn()
  },
  appStore: {
    showSuccess: vi.fn(),
    showError: vi.fn()
  },
  onboardingStore: {
    markKeysPageVisited: vi.fn(),
    isCurrentStep: vi.fn(),
    nextStep: vi.fn()
  },
  clipboardCopy: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key,
      locale: { value: 'zh-CN' }
    }
  }),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/api', () => ({
  keysAPI,
  authAPI,
  usageAPI,
  userGroupsAPI
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => onboardingStore
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: clipboardCopy
  })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main data-testid="app-section-shell"><slot /></main>'
  }
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="app-layout"><slot /></main>'
  }
}))

vi.mock('@/components/layout/TablePageLayout.vue', () => ({
  default: {
    name: 'TablePageLayout',
    template: `
      <section class="table-page-layout" :class="$attrs.class">
        <div data-testid="table-actions"><slot name="actions" /></div>
        <div data-testid="table-filters"><slot name="filters" /></div>
        <div data-testid="table-content"><slot name="table" /></div>
        <div data-testid="table-pagination"><slot name="pagination" /></div>
      </section>
    `
  }
}))

vi.mock('@/components/common/DataTable.vue', () => ({
  default: {
    name: 'DataTable',
    props: ['columns', 'data', 'loading'],
    template: `
      <div
        data-testid="data-table"
        :data-column-keys="columns.map((column) => column.key).join(',')"
      >
        <div v-for="row in data" :key="row.id" data-testid="data-row">
          <slot name="cell-key" :value="row.key" :row="row" />
          <slot name="cell-actions" :row="row" />
        </div>
        <slot v-if="!data.length" name="empty" />
      </div>
    `
  }
}))

vi.mock('@/components/common/Pagination.vue', () => ({
  default: {
    name: 'Pagination',
    template: '<nav data-testid="pagination" />'
  }
}))

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: ['show', 'title'],
    emits: ['close'],
    template: `
      <div v-if="show" data-testid="base-dialog">
        <h2>{{ title }}</h2>
        <slot />
        <slot name="footer" />
      </div>
    `
  }
}))

vi.mock('@/components/common/ConfirmDialog.vue', () => ({
  default: {
    name: 'ConfirmDialog',
    props: ['show', 'title', 'message', 'confirmText', 'cancelText', 'danger'],
    emits: ['confirm', 'cancel'],
    template: `
      <div v-if="show" data-testid="confirm-dialog">
        <h2>{{ title }}</h2>
        <p>{{ message }}</p>
        <button type="button" data-testid="confirm-cancel" @click="$emit('cancel')">{{ cancelText }}</button>
        <button type="button" data-testid="confirm-submit" @click="$emit('confirm')">{{ confirmText }}</button>
      </div>
    `
  }
}))

vi.mock('@/components/common/EmptyState.vue', () => ({
  default: {
    name: 'EmptyState',
    props: ['title', 'description', 'actionText'],
    template: '<div data-testid="empty-state">{{ title }} {{ description }} {{ actionText }}</div>'
  }
}))

vi.mock('@/components/common/Select.vue', () => ({
  default: {
    name: 'Select',
    inheritAttrs: false,
    props: {
      modelValue: { default: null },
      options: { type: Array, default: () => [] },
      searchable: { type: Boolean, default: false },
      searchPlaceholder: { type: String, default: '' }
    },
    emits: ['update:modelValue', 'change'],
    computed: {
      selectedOption() {
        return this.options.find((option: { value: unknown }) => option.value === this.modelValue) || null
      }
    },
    template: `
      <div
        v-bind="$attrs"
        data-component="select"
        :data-model-value="modelValue ?? ''"
        :data-searchable="String(Boolean(searchable))"
      >
        <button type="button" data-testid="select-trigger">
          <slot name="selected" :option="selectedOption">
            {{ selectedOption?.label || '' }}
          </slot>
        </button>
        <input v-if="searchable" type="search" :placeholder="searchPlaceholder" />
        <button
          v-for="option in options"
          :key="option.value"
          type="button"
          :data-option-value="option.value"
          @click="$emit('update:modelValue', option.value); $emit('change', option.value, option)"
        >
          <slot name="option" :option="option" :selected="option.value === modelValue">
            {{ option.label }}
          </slot>
        </button>
      </div>
    `
  }
}))

vi.mock('@/components/common/SearchInput.vue', () => ({
  default: {
    name: 'SearchInput',
    template: '<input data-testid="search-input" />'
  }
}))

vi.mock('@/components/keys/UseKeyModal.vue', () => ({
  default: {
    name: 'UseKeyModal',
    props: ['apiKey', 'allowedModels', 'baseUrl', 'keyStatus'],
    template: '<div data-testid="use-key-modal" :data-api-key="apiKey" :data-allowed-models="allowedModels?.join(\',\')" :data-base-url="baseUrl" :data-key-status="keyStatus" />'
  }
}))

vi.mock('@/components/common/GroupBadge.vue', () => ({
  default: {
    name: 'GroupBadge',
    props: ['name'],
    template: '<span data-testid="group-badge">{{ name }}</span>'
  }
}))

vi.mock('@/components/common/GroupOptionItem.vue', () => ({
  default: {
    name: 'GroupOptionItem',
    template: '<span data-testid="group-option-item" />'
  }
}))

vi.mock('@/components/account/ModelWhitelistSelector.vue', () => ({
  default: {
    name: 'ModelWhitelistSelector',
    template: '<div data-testid="model-whitelist-selector" />'
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span data-testid="icon" />'
  }
}))

import KeysView from '../KeysView.vue'

function mountView() {
  return mount(KeysView, {
    global: {
      stubs: {
        Teleport: true,
        SearchInput: true,
        Pagination: true,
        EmptyState: true,
        UseKeyModal: true,
        GroupOptionItem: true,
        ModelWhitelistSelector: true
      }
    }
  })
}

function apiKeyFixture(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    name: 'primary-key',
    key: 'sk-test...0001',
    status: 'active',
    group_id: 1,
    group_ids: [1],
    groups: [],
    group: null,
    allowed_models: [],
    ip_whitelist: [],
    ip_blacklist: [],
    quota: 0,
    quota_used: 0,
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
    expires_at: null,
    last_used_at: null,
    created_at: '2026-06-18T00:00:00Z',
    ...overrides
  }
}

function groupFixture(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    name: 'default',
    description: '',
    platform: null,
    subscription_type: null,
    rate_multiplier: 1,
    ...overrides
  }
}

describe('KeysView workbench surface', () => {
  beforeEach(() => {
    localStorage.clear()
    routeState.path = '/app/keys'
    keysAPI.list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    authAPI.getPublicSettings.mockResolvedValue({ api_base_url: 'https://example.test', site_name: 'SSXZ AI' })
    usageAPI.getDashboardApiKeysUsage.mockResolvedValue({})
    userGroupsAPI.getAvailable.mockResolvedValue([])
    userGroupsAPI.getUserGroupRates.mockResolvedValue({})
    onboardingStore.isCurrentStep.mockReturnValue(false)
    clipboardCopy.mockResolvedValue(true)
    vi.clearAllMocks()
  })

  it('uses the workbench shell and workbench table surface on /app/keys', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.find('.keys-page-surface--workbench').exists()).toBe(true)
    expect(wrapper.find('.keys-workbench-layout').exists()).toBe(true)
    expect(wrapper.find('.keys-workbench-layout--no-pagination').exists()).toBe(true)
    expect(wrapper.find('.keys-client-guide').exists()).toBe(false)
    expect(wrapper.find('.keys-client-callout').exists()).toBe(false)
    expect(wrapper.find('[data-testid="keys-base-url-row"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/docs"]').text()).toContain('home.viewDocs')
    expect(wrapper.text()).toContain('https://example.test/v1')

    await wrapper.get('[data-testid="keys-guide-copy-base-url"]').trigger('click')
    await flushPromises()

    expect(clipboardCopy).toHaveBeenCalledWith(
      'https://example.test/v1',
      'keys.workbenchGuide.baseUrlCopied'
    )
  })

  it('shows eight primary columns by default and keeps advanced columns in column settings', async () => {
    const wrapper = mountView()
    await flushPromises()

    const table = wrapper.get('[data-testid="data-table"]')
    expect(table.attributes('data-column-keys')).toBe(
      'name,key,group,usage,status,expires_at,created_at,actions'
    )

    await wrapper.get('[data-testid="keys-column-settings-trigger"]').trigger('click')

    const rateLimitOption = wrapper.get('[data-column-key="rate_limit"]')
    const lastUsedOption = wrapper.get('[data-column-key="last_used_at"]')
    expect(rateLimitOption.attributes('aria-checked')).toBe('false')
    expect(lastUsedOption.attributes('aria-checked')).toBe('false')

    await rateLimitOption.trigger('click')
    expect(table.attributes('data-column-keys')).toBe(
      'name,key,group,usage,rate_limit,status,expires_at,created_at,actions'
    )
  })

  it('keeps all row actions visible in one icon row', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const actions = wrapper.get('.keys-row-actions')
    const persistentButtons = actions.findAll(':scope > button')
    expect(persistentButtons).toHaveLength(5)
    expect(persistentButtons.map((button) => button.attributes('aria-label'))).toEqual([
      'keys.useKey',
      'keys.importToCcSwitch',
      'common.edit',
      'keys.disable',
      'common.delete'
    ])
    expect(persistentButtons.every((button) => button.get('.sr-only').exists())).toBe(true)
    expect(actions.find('.keys-more-actions').exists()).toBe(false)
  })

  it('keeps the pagination section when API keys exist', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.keys-workbench-layout--no-pagination').exists()).toBe(false)
  })

  it('keeps API key creation disabled until an available group exists', async () => {
    userGroupsAPI.getAvailable.mockResolvedValue([])

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    expect(createButton!.attributes('disabled')).toBeDefined()
    expect(createButton!.attributes('title')).toBe('keys.noAvailableGroups')

    await createButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('form#key-form').exists()).toBe(false)
  })

  it('does not duplicate v1 when the public API base URL already includes it', async () => {
    authAPI.getPublicSettings.mockResolvedValue({ api_base_url: 'https://example.test/v1', site_name: 'SSXZ AI' })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('https://example.test/v1')
    expect(wrapper.text()).not.toContain('/v1/v1')

    await wrapper.get('[data-testid="keys-guide-copy-base-url"]').trigger('click')
    await flushPromises()

    expect(clipboardCopy).toHaveBeenCalledWith(
      'https://example.test/v1',
      'keys.workbenchGuide.baseUrlCopied'
    )
  })

  it('keeps the legacy /keys surface on the legacy route', async () => {
    routeState.path = '/keys'
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
    expect(wrapper.find('.keys-page-surface--workbench').exists()).toBe(false)
    expect(wrapper.find('.keys-workbench-layout').exists()).toBe(false)
  })

  it('requires confirmation before disabling an active API key', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })
    keysAPI.toggleStatus.mockResolvedValue(apiKeyFixture({ status: 'inactive' }))

    const wrapper = mountView()
    await flushPromises()

    const disableButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.disable'))
    expect(disableButton).toBeTruthy()
    await disableButton!.trigger('click')

    expect(keysAPI.toggleStatus).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('keys.disableKeyTitle')
    expect(wrapper.text()).toContain('keys.disableConfirmMessage')

    await wrapper.get('[data-testid="confirm-submit"]').trigger('click')
    await flushPromises()

    expect(keysAPI.toggleStatus).toHaveBeenCalledWith(1, 'inactive')
    expect(keysAPI.list).toHaveBeenCalledTimes(2)
    expect(appStore.showSuccess).toHaveBeenCalledWith('keys.keyDisabledSuccess')
  })

  it('retrieves and copies an owned masked list key without storing it in the row', async () => {
    const maskedKey = 'sk-user-...1234'
    const fullKey = 'sk-user-revealed-full-key-1234'
    keysAPI.reveal.mockResolvedValue({ key: fullKey })
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture({ key: maskedKey })],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const copyButton = wrapper.get('button[title="keys.copyToClipboard"]')
    expect(copyButton.attributes('disabled')).toBeUndefined()
    await copyButton.trigger('click')

		await flushPromises()

    expect(keysAPI.reveal).toHaveBeenCalledWith(1)
    expect(clipboardCopy).toHaveBeenCalledWith(fullKey, 'keys.copied')
    expect(wrapper.html()).not.toContain(fullKey)
  })

  it('copies a full list key when one is available', async () => {
    const fullKey = 'sk-full-key-value-visible-once-1234'
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture({ key: fullKey })],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[title="keys.copyToClipboard"]').trigger('click')

    expect(clipboardCopy).toHaveBeenCalledWith(fullKey, 'keys.copied')
  })

  it('does not change API key status when the confirmation is cancelled', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const disableButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.disable'))
    await disableButton!.trigger('click')
    await wrapper.get('[data-testid="confirm-cancel"]').trigger('click')
    await flushPromises()

    expect(keysAPI.toggleStatus).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="confirm-dialog"]').exists()).toBe(false)
  })

  it('does not fake success or refresh data when disabling an API key fails', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })
    keysAPI.toggleStatus.mockRejectedValue(new Error('network unavailable'))

    const wrapper = mountView()
    await flushPromises()

    const disableButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.disable'))
    expect(disableButton).toBeTruthy()
    await disableButton!.trigger('click')
    await wrapper.get('[data-testid="confirm-submit"]').trigger('click')
    await flushPromises()

    expect(keysAPI.toggleStatus).toHaveBeenCalledWith(1, 'inactive')
    expect(keysAPI.list).toHaveBeenCalledTimes(1)
    expect(appStore.showError).toHaveBeenCalledWith('keys.failedToUpdateStatus')
    expect(appStore.showSuccess).not.toHaveBeenCalled()
  })

  it('requires confirmation before enabling an inactive API key', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture({ status: 'inactive' })],
      total: 1,
      pages: 1
    })
    keysAPI.toggleStatus.mockResolvedValue(apiKeyFixture({ status: 'active' }))

    const wrapper = mountView()
    await flushPromises()

    const enableButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.enable'))
    expect(enableButton).toBeTruthy()
    await enableButton!.trigger('click')

    expect(keysAPI.toggleStatus).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('keys.enableKeyTitle')
    expect(wrapper.text()).toContain('keys.enableConfirmMessage')

    await wrapper.get('[data-testid="confirm-submit"]').trigger('click')
    await flushPromises()

    expect(keysAPI.toggleStatus).toHaveBeenCalledWith(1, 'active')
    expect(keysAPI.list).toHaveBeenCalledTimes(2)
    expect(appStore.showSuccess).toHaveBeenCalledWith('keys.keyEnabledSuccess')
  })

  it('requires confirmation before deleting an API key', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })
    keysAPI.delete.mockResolvedValue({})

    const wrapper = mountView()
    await flushPromises()

    const deleteButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.delete'))
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')

    expect(keysAPI.delete).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('keys.deleteKey')
    expect(wrapper.text()).toContain('keys.deleteConfirmMessage')

    await wrapper.get('[data-testid="confirm-submit"]').trigger('click')
    await flushPromises()

    expect(keysAPI.delete).toHaveBeenCalledWith(1)
    expect(keysAPI.list).toHaveBeenCalledTimes(2)
    expect(appStore.showSuccess).toHaveBeenCalledWith('keys.keyDeletedSuccess')
    expect(wrapper.find('[data-testid="confirm-dialog"]').exists()).toBe(false)
  })

  it('keeps the delete confirmation visible and does not refresh when deleting an API key fails', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })
    keysAPI.delete.mockRejectedValue(new Error('cannot delete active key'))

    const wrapper = mountView()
    await flushPromises()

    const deleteButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.delete'))
    expect(deleteButton).toBeTruthy()
    await deleteButton!.trigger('click')
    await wrapper.get('[data-testid="confirm-submit"]').trigger('click')
    await flushPromises()

    expect(keysAPI.delete).toHaveBeenCalledWith(1)
    expect(keysAPI.list).toHaveBeenCalledTimes(1)
    expect(appStore.showError).toHaveBeenCalledWith('cannot delete active key')
    expect(appStore.showSuccess).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="confirm-dialog"]').exists()).toBe(true)
  })

  it('does not delete an API key when the confirmation is cancelled', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const deleteButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.delete'))
    await deleteButton!.trigger('click')
    await wrapper.get('[data-testid="confirm-cancel"]').trigger('click')
    await flushPromises()

    expect(keysAPI.delete).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="confirm-dialog"]').exists()).toBe(false)
  })

  it('refreshes API key data after resetting quota usage', async () => {
    const exhaustedKey = apiKeyFixture({
      id: 9,
      name: 'quota-key',
      status: 'quota_exhausted',
      quota: 10,
      quota_used: 12.5
    })
    const refreshedKey = apiKeyFixture({
      id: 9,
      name: 'quota-key',
      status: 'active',
      quota: 10,
      quota_used: 0
    })
    keysAPI.list
      .mockResolvedValueOnce({ items: [exhaustedKey], total: 1, pages: 1 })
      .mockResolvedValueOnce({ items: [refreshedKey], total: 1, pages: 1 })
    keysAPI.update.mockResolvedValue(refreshedKey)

    const wrapper = mountView()
    await flushPromises()

    const editButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const resetButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.reset'))
    expect(resetButton).toBeTruthy()
    await resetButton!.trigger('click')

    expect(keysAPI.update).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('keys.resetQuotaTitle')

    await wrapper.get('[data-testid="confirm-submit"]').trigger('click')
    await flushPromises()

    expect(keysAPI.update).toHaveBeenCalledWith(9, { reset_quota: true })
    expect(keysAPI.list).toHaveBeenCalledTimes(2)
    expect(appStore.showSuccess).toHaveBeenCalledWith('keys.quotaResetSuccess')
  })

  it('refreshes API key data after resetting rate-limit usage', async () => {
    const limitedKey = apiKeyFixture({
      id: 10,
      name: 'limited-key',
      rate_limit_1d: 25,
      usage_1d: 5
    })
    const refreshedKey = apiKeyFixture({
      id: 10,
      name: 'limited-key',
      rate_limit_1d: 25,
      usage_1d: 0
    })
    keysAPI.list
      .mockResolvedValueOnce({ items: [limitedKey], total: 1, pages: 1 })
      .mockResolvedValueOnce({ items: [refreshedKey], total: 1, pages: 1 })
    keysAPI.update.mockResolvedValue(refreshedKey)

    const wrapper = mountView()
    await flushPromises()

    const editButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const resetRateLimitButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.resetRateLimitUsage'))
    expect(resetRateLimitButton).toBeTruthy()
    await resetRateLimitButton!.trigger('click')

    expect(keysAPI.update).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('keys.resetRateLimitTitle')

    await wrapper.get('[data-testid="confirm-submit"]').trigger('click')
    await flushPromises()

    expect(keysAPI.update).toHaveBeenCalledWith(10, { reset_rate_limit_usage: true })
    expect(keysAPI.list).toHaveBeenCalledTimes(2)
    expect(appStore.showSuccess).toHaveBeenCalledWith('keys.rateLimitResetSuccess')
  })

  it('preselects the first available group when creating an API key', async () => {
    userGroupsAPI.getAvailable.mockResolvedValue([groupFixture()])

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="key-form-group-select"]').attributes('data-model-value')).toBe('1')
  })

  it('shows the real group name instead of a generic category', async () => {
    userGroupsAPI.getAvailable.mockResolvedValue([
      groupFixture({
        name: 'Claude Kiro高缓池',
        description: 'Kiro高缓·性价比',
        platform: 'anthropic'
      })
    ])

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Claude Kiro高缓池')
    expect(wrapper.text()).not.toContain('高级模型组')
  })

  it('uses a searchable single-select and submits only the newly selected group', async () => {
    userGroupsAPI.getAvailable.mockResolvedValue([
      groupFixture({
        id: 11,
        name: 'Claude 满血池(CCMAX)',
        description: '满血高质量·支持Fable5',
        platform: 'anthropic',
        rate_multiplier: 1.2
      }),
      groupFixture({
        id: 15,
        name: 'Codex池',
        description: '标准Codex·性价比',
        platform: 'openai',
        rate_multiplier: 0.8
      })
    ])
    keysAPI.create.mockResolvedValue(apiKeyFixture({ group_id: 15, group_ids: [15] }))

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    const groupSelect = wrapper.get('[data-testid="key-form-group-select"]')
    expect(groupSelect.attributes('data-searchable')).toBe('true')
    expect(groupSelect.attributes('data-model-value')).toBe('11')
    expect(groupSelect.text()).toContain('Claude 满血池(CCMAX)')
    expect(groupSelect.text()).toContain('满血高质量·支持Fable5')
    expect(groupSelect.text()).toContain('1.2x')
    expect(groupSelect.text()).toContain('Codex池')
    expect(groupSelect.text()).toContain('0.8x')
    expect(groupSelect.find('input[type="checkbox"]').exists()).toBe(false)

    const claudeRateBadge = groupSelect
      .get('[data-option-value="11"]')
      .findAll('span')
      .find((span) => span.text() === '1.2x')
    const codexRateBadge = groupSelect
      .get('[data-option-value="15"]')
      .findAll('span')
      .find((span) => span.text() === '0.8x')
    expect(claudeRateBadge?.classes()).toContain('bg-orange-50')
    expect(codexRateBadge?.classes()).toContain('bg-emerald-50')

    await groupSelect.get('[data-option-value="15"]').trigger('click')
    await wrapper.get('[data-tour="key-form-name"]').setValue('single-group-key')
    await wrapper.get('form#key-form').trigger('submit')
    await flushPromises()

    expect(keysAPI.create).toHaveBeenCalledWith(
      'single-group-key',
      15,
      [15],
      [],
      undefined,
      [],
      [],
      0,
      undefined,
      { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 },
      { idempotencyKey: expect.stringMatching(/^api-key-create-/) }
    )
  })

  it('reveals the full API key once after creation', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const createdKey = 'sk-created-full-key-only-shown-once'
    userGroupsAPI.getAvailable.mockResolvedValue([groupFixture()])
    keysAPI.create.mockResolvedValue(apiKeyFixture({
      id: 2,
      name: 'client-key',
      key: createdKey,
      group_id: 1,
      group_ids: [1],
      group: { platform: 'openai', allow_messages_dispatch: false }
    }))

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    await wrapper.get('[data-tour="key-form-name"]').setValue('client-key')
    await wrapper.get('form#key-form').trigger('submit')
    await flushPromises()

    expect(keysAPI.create).toHaveBeenCalledWith(
      'client-key',
      1,
      [1],
      [],
      undefined,
      [],
      [],
      0,
      undefined,
      { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 },
      { idempotencyKey: expect.stringMatching(/^api-key-create-/) }
    )
    expect(wrapper.find('[data-testid="created-key-reveal"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="created-key-value"]').element as HTMLInputElement).value).toBe(createdKey)
    expect(wrapper.text()).toContain('keys.createdKeyReveal.connectionTitle')
    expect(wrapper.text()).toContain('keys.workbenchGuide.baseUrlLabel')
    expect(wrapper.text()).toContain('keys.createdKeyReveal.modelHint')
    expect(wrapper.text()).toContain('keys.createdKeyReveal.readinessHint')
    expect(wrapper.text()).toContain('keys.createdKeyReveal.primaryActionHint')
    expect(wrapper.text()).toContain('https://example.test/v1')
    expect(wrapper.get('[data-testid="created-key-ccs-import"]').classes()).toContain('btn-primary')
    expect(wrapper.get('[data-testid="created-key-ack"]').classes()).toContain('btn-secondary')

    await wrapper.get('[data-testid="created-key-base-url-copy"]').trigger('click')
    await flushPromises()
    expect(clipboardCopy).toHaveBeenCalledWith(
      'https://example.test/v1',
      'keys.workbenchGuide.baseUrlCopied'
    )

    await wrapper.get('[data-testid="created-key-copy"]').trigger('click')
    await flushPromises()
    expect(clipboardCopy).toHaveBeenCalledWith(createdKey, 'keys.createdKeyReveal.fullKeyCopied')

    await wrapper.get('[data-testid="created-key-ccs-import"]').trigger('click')
    await flushPromises()
    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(deeplink).toMatch(/^ccswitch:\/\/v1\/import\?/)
    expect(params.get('apiKey')).toBe(createdKey)
    expect(params.get('homepage')).toBe('https://example.test')
    expect(params.get('endpoint')).toBe('https://example.test/v1')
    expect(params.get('model')).toBe('gpt-5.5')
    expect(params.get('enabled')).toBe('true')
    expect(params.get('usageBaseUrl')).toBe('https://example.test')

    await wrapper.get('[data-testid="created-key-ack"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="created-key-reveal"]').exists()).toBe(false)
    expect(wrapper.html()).not.toContain(createdKey)

    openSpy.mockRestore()
  })

  it('shows copy-ready quick start examples in the created key dialog', async () => {
    const createdKey = 'sk-created-full-key-only-shown-once'
    userGroupsAPI.getAvailable.mockResolvedValue([groupFixture()])
    keysAPI.create.mockResolvedValue(apiKeyFixture({
      id: 2,
      name: 'client-key',
      key: createdKey,
      group_id: 1,
      group_ids: [1],
      group: { platform: 'openai', allow_messages_dispatch: false }
    }))

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    await wrapper.get('[data-tour="key-form-name"]').setValue('client-key')
    await wrapper.get('form#key-form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('[data-testid="created-key-quick-start"]').exists()).toBe(true)
    const codeBlock = () => wrapper.get('[data-testid="created-key-example-code"]').text()

    expect(codeBlock()).toContain('curl -X POST "https://example.test/v1/chat/completions"')
    expect(codeBlock()).toContain(`Authorization: Bearer ${createdKey}`)
    expect(codeBlock()).toContain('"model": "gpt-5.5"')

    await wrapper.get('[data-testid="created-key-example-copy"]').trigger('click')
    await flushPromises()
    const curlCopied = String(clipboardCopy.mock.calls.at(-1)?.[0])
    expect(clipboardCopy).toHaveBeenLastCalledWith(
      expect.any(String),
      'keys.createdKeyReveal.quickStart.exampleCopied'
    )
    expect(curlCopied).toContain('curl -X POST "https://example.test/v1/chat/completions"')
    expect(curlCopied).toContain(createdKey)

    await wrapper.get('[data-testid="created-key-example-tab-python"]').trigger('click')
    await flushPromises()
    expect(codeBlock()).toContain('from openai import OpenAI')
    expect(codeBlock()).toContain('base_url="https://example.test/v1"')
    expect(codeBlock()).toContain(`api_key="${createdKey}"`)

    await wrapper.get('[data-testid="created-key-example-tab-cherry"]').trigger('click')
    await flushPromises()
    expect(codeBlock()).toContain('keys.createdKeyReveal.quickStart.cherryProvider')
    expect(codeBlock()).toContain(`keys.createdKeyReveal.quickStart.cherryApiKey${createdKey}`)

    await wrapper.get('[data-testid="created-key-example-copy"]').trigger('click')
    await flushPromises()
    const cherryCopied = String(clipboardCopy.mock.calls.at(-1)?.[0])
    expect(cherryCopied).toContain(createdKey)
    expect(cherryCopied).toContain('https://example.test/v1')
  })

  it('ignores duplicate create submits while the first request is pending', async () => {
    userGroupsAPI.getAvailable.mockResolvedValue([groupFixture()])
    let resolveCreate!: (value: unknown) => void
    keysAPI.create.mockImplementation(() => new Promise((resolve) => {
      resolveCreate = resolve
    }))

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    await wrapper.get('[data-tour="key-form-name"]').setValue('client-key')
    const form = wrapper.get('form#key-form')
    await form.trigger('submit')
    await form.trigger('submit')

    expect(keysAPI.create).toHaveBeenCalledTimes(1)
    expect(keysAPI.create).toHaveBeenCalledWith(
      'client-key',
      1,
      [1],
      [],
      undefined,
      [],
      [],
      0,
      undefined,
      { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 },
      { idempotencyKey: expect.stringMatching(/^api-key-create-/) }
    )

    resolveCreate(apiKeyFixture({
      id: 2,
      name: 'client-key',
      key: 'sk-created-once',
      group_id: 1,
      group_ids: [1]
    }))
    await flushPromises()
  })

  it('does not close the create dialog or reveal a key when API key creation fails', async () => {
    userGroupsAPI.getAvailable.mockResolvedValue([groupFixture()])
    keysAPI.create.mockRejectedValue({
      response: {
        data: {
          detail: 'quota is not available for this group'
        }
      }
    })

    const wrapper = mountView()
    await flushPromises()

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.createKey'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    await wrapper.get('[data-tour="key-form-name"]').setValue('client-key')
    await wrapper.get('form#key-form').trigger('submit')
    await flushPromises()

    expect(keysAPI.create).toHaveBeenCalledTimes(1)
    expect(keysAPI.list).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="base-dialog"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="created-key-reveal"]').exists()).toBe(false)
    expect(appStore.showError).toHaveBeenCalledWith('quota is not available for this group')
    expect(appStore.showSuccess).not.toHaveBeenCalled()
    expect(onboardingStore.nextStep).not.toHaveBeenCalled()
  })

  it('does not close the edit dialog or refresh data when API key update fails', async () => {
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture()],
      total: 1,
      pages: 1
    })
    keysAPI.update.mockRejectedValue({
      response: {
        data: {
          detail: 'API key name is already used'
        }
      }
    })

    const wrapper = mountView()
    await flushPromises()

    const editButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    await wrapper.get('[data-tour="key-form-name"]').setValue('renamed-key')
    await wrapper.get('form#key-form').trigger('submit')
    await flushPromises()

    expect(keysAPI.update).toHaveBeenCalledWith(1, expect.objectContaining({
      name: 'renamed-key'
    }))
    expect(keysAPI.list).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="base-dialog"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="created-key-reveal"]').exists()).toBe(false)
    expect(appStore.showError).toHaveBeenCalledWith('API key name is already used')
    expect(appStore.showSuccess).not.toHaveBeenCalled()
  })

  it('does not pass a masked list key into the usage modal', async () => {
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-user-...1234',
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const useKeyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.useKey'))
    expect(useKeyButton).toBeTruthy()
    await useKeyButton!.trigger('click')
    await flushPromises()

    const modal = wrapper.findComponent({ name: 'UseKeyModal' })
    expect(modal.exists()).toBe(true)
    expect(modal.attributes('apikey')).toBe('')
  })

  it('passes the current key model allowlist into the usage modal', async () => {
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          allowed_models: ['gpt-4.1', 'gpt-4o-mini'],
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const useKeyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.useKey'))
    expect(useKeyButton).toBeTruthy()
    await useKeyButton!.trigger('click')
    await flushPromises()

    const modal = wrapper.findComponent({ name: 'UseKeyModal' })
    expect(modal.exists()).toBe(true)
    expect(modal.attributes('apikey')).toBe('sk-full-key-value-visible-once-1234')
    expect(modal.attributes('allowedmodels')).toBe('gpt-4.1,gpt-4o-mini')
  })

  it('passes the current key status into the usage modal', async () => {
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          status: 'quota_exhausted',
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const useKeyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.useKey'))
    expect(useKeyButton).toBeTruthy()
    await useKeyButton!.trigger('click')
    await flushPromises()

    const modal = wrapper.findComponent({ name: 'UseKeyModal' })
    expect(modal.exists()).toBe(true)
    expect(modal.props('keyStatus')).toBe('quota_exhausted')
  })

  it('passes the user-facing Base URL to the usage modal instead of a loopback configured URL', async () => {
    authAPI.getPublicSettings.mockResolvedValue({
      api_base_url: 'http://127.0.0.1:8080',
      site_name: 'SSXZ AI'
    })
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const useKeyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.useKey'))
    expect(useKeyButton).toBeTruthy()
    await useKeyButton!.trigger('click')
    await flushPromises()

    const modal = wrapper.findComponent({ name: 'UseKeyModal' })
    expect(modal.exists()).toBe(true)
    expect(modal.attributes('baseurl')).toBe(window.location.origin)
    expect(modal.attributes('baseurl')).not.toContain('127.0.0.1')
  })

  it('uses the user-facing Base URL for CCS import when settings contain a loopback URL', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    authAPI.getPublicSettings.mockResolvedValue({
      api_base_url: 'http://127.0.0.1:8080',
      site_name: 'SSXZ AI'
    })
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    await importButton!.trigger('click')

    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('homepage')).toBe(window.location.origin)
    expect(params.get('endpoint')).toBe(`${window.location.origin}/v1`)
    expect(deeplink).not.toContain('127.0.0.1')

    openSpy.mockRestore()
  })

  it('imports OpenAI platform keys to CC Switch as Codex/OpenAI-compatible', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    await importButton!.trigger('click')

    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe('https://example.test/v1')
    expect(params.get('model')).toBe('gpt-5.5')
    expect(params.get('enabled')).toBe('true')
    expect(params.get('usageBaseUrl')).toBe('https://example.test')
    expect(atob(params.get('usageScript') || '')).toContain('"{{baseUrl}}/v1/usage"')

    openSpy.mockRestore()
  })

  it('imports Gemini platform keys to CC Switch as Gemini-compatible', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group: { platform: 'gemini', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    await importButton!.trigger('click')

    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('app')).toBe('gemini')
    expect(params.get('homepage')).toBe('https://example.test')
    expect(params.get('endpoint')).toBe('https://example.test')

    openSpy.mockRestore()
  })

  it('imports Anthropic platform keys to CC Switch as Claude-compatible', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group: { platform: 'anthropic', allow_messages_dispatch: true }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    await importButton!.trigger('click')

    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('app')).toBe('claude')
    expect(params.get('homepage')).toBe('https://example.test')
    expect(params.get('endpoint')).toBe('https://example.test')
    expect(params.get('model')).toBe('claude-opus-4-8')
    expect(params.get('enabled')).toBe('true')
    expect(params.get('usageBaseUrl')).toBe('https://example.test')

    openSpy.mockRestore()
  })

  it('asks which CC Switch client to import Antigravity platform keys into', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group: { platform: 'antigravity', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    await importButton!.trigger('click')
    await flushPromises()

    expect(openSpy).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('keys.ccsClientSelect.title')

    const claudeButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.ccsClientSelect.claudeCode'))
    expect(claudeButton).toBeTruthy()
    await claudeButton!.trigger('click')

    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('app')).toBe('claude')
    expect(params.get('endpoint')).toBe('https://example.test/antigravity')

    openSpy.mockRestore()
  })

  it('does not import mixed-platform keys to CC Switch', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group: { platform: 'openai', allow_messages_dispatch: false },
          groups: [
            { id: 1, name: 'openai', platform: 'openai' },
            { id: 2, name: 'gemini', platform: 'gemini' }
          ]
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    await importButton!.trigger('click')

    expect(openSpy).not.toHaveBeenCalled()
    expect(appStore.showError).toHaveBeenCalledWith('keys.noGroupFound')

    openSpy.mockRestore()
  })

  it('does not default unknown-platform keys to Claude when importing to CC Switch', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-full-key-value-visible-once-1234',
          group_id: null,
          group_ids: [],
          groups: [],
          group: null
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    await importButton!.trigger('click')

    expect(openSpy).not.toHaveBeenCalled()
    expect(appStore.showError).toHaveBeenCalledWith('keys.noGroupFound')

    openSpy.mockRestore()
  })

  it('retrieves a masked owned key and imports it to CC Switch in one click', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const fullKey = 'sk-user-revealed-full-key-1234'
    keysAPI.reveal.mockResolvedValue({ key: fullKey })
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-user-...1234',
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const importButton = wrapper.get('[data-testid="api-key-ccs-import"]')
    expect(importButton.attributes('disabled')).toBeUndefined()
    await importButton.trigger('click')
		await flushPromises()

    expect(keysAPI.reveal).toHaveBeenCalledWith(1)
    expect(openSpy).toHaveBeenCalledTimes(1)
    const deeplink = String(openSpy.mock.calls[0]?.[0])
    const params = new URLSearchParams(deeplink.split('?')[1])
    expect(params.get('apiKey')).toBe(fullKey)
    expect(params.get('model')).toBe('gpt-5.5')
    expect(params.get('enabled')).toBe('true')
    expect(wrapper.html()).not.toContain(fullKey)

    openSpy.mockRestore()
  })

  it('does not open CC Switch when retrieving a masked key fails', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    keysAPI.reveal.mockRejectedValue(new Error('request failed'))
    keysAPI.list.mockResolvedValue({
      items: [
        apiKeyFixture({
          key: 'sk-user-...1234',
          group: { platform: 'openai', allow_messages_dispatch: false }
        })
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="api-key-ccs-import"]').trigger('click')
    await flushPromises()

    expect(openSpy).not.toHaveBeenCalled()
    expect(appStore.showError).toHaveBeenCalledWith('keys.failedToReveal')

    openSpy.mockRestore()
  })
})
