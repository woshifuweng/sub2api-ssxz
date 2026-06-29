import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, keysAPI, authAPI, usageAPI, userGroupsAPI, appStore, onboardingStore, clipboardCopy } = vi.hoisted(() => ({
  routeState: {
    path: '/app/keys'
  },
  keysAPI: {
    list: vi.fn(),
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
      t: (key: string) => key
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
      <div data-testid="data-table">
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
    template: '<select data-testid="select" />'
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
    props: ['apiKey'],
    template: '<div data-testid="use-key-modal" :data-api-key="apiKey" />'
  }
}))

vi.mock('@/components/common/GroupBadge.vue', () => ({
  default: {
    name: 'GroupBadge',
    template: '<span data-testid="group-badge" />'
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
        Select: true,
        Pagination: true,
        EmptyState: true,
        UseKeyModal: true,
        GroupBadge: true,
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
    expect(wrapper.text()).toContain('keys.clientAccessTitle')
    expect(wrapper.text()).toContain('keys.workbenchGuide.title')
    expect(wrapper.text()).toContain('CC Switch')
    expect(wrapper.text()).toContain('Cherry Studio')
    expect(wrapper.text()).toContain('Chatbox')
    expect(wrapper.text()).toContain('https://example.test')

    await wrapper.get('[data-testid="keys-guide-copy-base-url"]').trigger('click')
    await flushPromises()

    expect(clipboardCopy).toHaveBeenCalledWith(
      'https://example.test',
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
  })

  it('does not copy the masked list key value', async () => {
    const maskedKey = 'sk-user-...1234'
    keysAPI.list.mockResolvedValue({
      items: [apiKeyFixture({ key: maskedKey })],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const copyButton = wrapper.get('button[title="keys.fullKeyRequiredForImport"]')
    expect(copyButton.attributes('disabled')).toBeDefined()
    await copyButton.trigger('click')

    expect(clipboardCopy).not.toHaveBeenCalled()
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

    expect((wrapper.get('input[type="checkbox"]').element as HTMLInputElement).checked).toBe(true)
  })

  it('reveals the full API key once after creation', async () => {
    const createdKey = 'sk-created-full-key-only-shown-once'
    userGroupsAPI.getAvailable.mockResolvedValue([groupFixture()])
    keysAPI.create.mockResolvedValue(apiKeyFixture({
      id: 2,
      name: 'client-key',
      key: createdKey,
      group_id: 1,
      group_ids: [1]
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
      { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }
    )
    expect(wrapper.find('[data-testid="created-key-reveal"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="created-key-value"]').element as HTMLInputElement).value).toBe(createdKey)
    expect(wrapper.text()).toContain('keys.createdKeyReveal.connectionTitle')
    expect(wrapper.text()).toContain('keys.workbenchGuide.baseUrlLabel')
    expect(wrapper.text()).toContain('keys.createdKeyReveal.modelHint')
    expect(wrapper.text()).toContain('https://example.test')

    await wrapper.get('[data-testid="created-key-base-url-copy"]').trigger('click')
    await flushPromises()
    expect(clipboardCopy).toHaveBeenCalledWith(
      'https://example.test',
      'keys.workbenchGuide.baseUrlCopied'
    )

    await wrapper.get('[data-testid="created-key-copy"]').trigger('click')
    await flushPromises()
    expect(clipboardCopy).toHaveBeenCalledWith(createdKey, 'keys.createdKeyReveal.fullKeyCopied')

    await wrapper.get('[data-testid="created-key-ack"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="created-key-reveal"]').exists()).toBe(false)
    expect(wrapper.html()).not.toContain(createdKey)
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

  it('keeps CCS import disabled for masked list keys', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
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

    const importButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('keys.importToCcSwitch'))
    expect(importButton).toBeTruthy()
    expect(importButton!.attributes('disabled')).toBeDefined()

    await importButton!.trigger('click')
    expect(openSpy).not.toHaveBeenCalled()

    openSpy.mockRestore()
  })
})
