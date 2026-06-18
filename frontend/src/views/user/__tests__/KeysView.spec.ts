import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, keysAPI, authAPI, usageAPI, userGroupsAPI, appStore, onboardingStore } = vi.hoisted(() => ({
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
    markKeysPageVisited: vi.fn()
  }
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
    copyToClipboard: vi.fn()
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
    template: '<div data-testid="base-dialog"><slot /></div>'
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
    template: '<div data-testid="use-key-modal" />'
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
        BaseDialog: true,
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

describe('KeysView workbench surface', () => {
  beforeEach(() => {
    routeState.path = '/app/keys'
    keysAPI.list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    authAPI.getPublicSettings.mockResolvedValue({ api_base_url: 'https://example.test', site_name: 'SSXZ AI' })
    usageAPI.getDashboardApiKeysUsage.mockResolvedValue({})
    userGroupsAPI.getAvailable.mockResolvedValue([])
    userGroupsAPI.getUserGroupRates.mockResolvedValue({})
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
    expect(wrapper.text()).toContain('把 SSXZ AI 接到你常用的客户端')
    expect(wrapper.text()).toContain('CC Switch')
    expect(wrapper.text()).toContain('Cherry Studio')
    expect(wrapper.text()).toContain('Chatbox')
    expect(wrapper.text()).toContain('https://example.test')
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
})
