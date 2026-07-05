import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { adminAPI, routerPush, showError, showSuccess } = vi.hoisted(() => ({
  adminAPI: {
    users: {
      list: vi.fn(),
      toggleStatus: vi.fn(),
      delete: vi.fn()
    },
    userAttributes: {
      listEnabledDefinitions: vi.fn(),
      getBatchUserAttributes: vi.fn()
    },
    dashboard: {
      getBatchUsersUsage: vi.fn()
    },
    groups: {
      getAll: vi.fn()
    }
  },
  routerPush: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, fallback?: unknown) => (typeof fallback === 'string' ? fallback : key)
    })
  }
})

import UsersView from '../UsersView.vue'

function mountView() {
  return mount(UsersView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<section><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>'
        },
        DataTable: {
          props: ['data'],
          template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-actions" :row="row" /></div></div>'
        },
        Pagination: true,
        ConfirmDialog: true,
        EmptyState: true,
        GroupBadge: true,
        Select: true,
        UserAttributesConfigModal: true,
        UserConcurrencyCell: true,
        UserCreateModal: true,
        UserEditModal: true,
        UserApiKeysModal: true,
        UserAllowedGroupsModal: true,
        UserBalanceModal: true,
        UserBalanceHistoryModal: true,
        GroupReplaceModal: true,
        Icon: true,
        Teleport: true
      }
    }
  })
}

describe('admin UsersView investigation links', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminAPI.userAttributes.listEnabledDefinitions.mockResolvedValue([])
    adminAPI.userAttributes.getBatchUserAttributes.mockResolvedValue({ attributes: {} })
    adminAPI.dashboard.getBatchUsersUsage.mockResolvedValue({ stats: {} })
    adminAPI.groups.getAll.mockResolvedValue([])
    adminAPI.users.list.mockResolvedValue({
      items: [
        {
          id: 42,
          email: 'customer@example.com',
          username: 'customer',
          role: 'user',
          status: 'active',
          balance: 8.26,
          concurrency: 0,
          allowed_groups: [],
          subscriptions: [],
          created_at: '2026-07-01T00:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
  })

  it('opens affiliate investigation from a user row', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('.action-menu-trigger').trigger('click', { clientX: 480, clientY: 240 })
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text().includes('推广返利'))!.trigger('click')

    expect(routerPush).toHaveBeenCalledWith({
      path: '/admin/affiliates',
      query: { search: 'customer@example.com' }
    })
  })

  it('opens redeem investigation from a user row', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('.action-menu-trigger').trigger('click', { clientX: 480, clientY: 240 })
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text().includes('兑换记录'))!.trigger('click')

    expect(routerPush).toHaveBeenCalledWith({
      path: '/admin/redeem',
      query: { search: 'customer@example.com' }
    })
  })
})
