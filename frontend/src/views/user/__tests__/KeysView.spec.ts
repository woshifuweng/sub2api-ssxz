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
    template: '<div data-testid="data-table"><slot name="empty" /></div>'
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
    template: '<div data-testid="confirm-dialog" />'
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
        ConfirmDialog: true,
        EmptyState: true,
        UseKeyModal: true,
        GroupBadge: true,
        GroupOptionItem: true,
        ModelWhitelistSelector: true
      }
    }
  })
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
})
