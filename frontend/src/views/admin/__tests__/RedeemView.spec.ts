import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { redeemAPI, groupsAPI, showError, showSuccess, routeQuery } = vi.hoisted(() => ({
  redeemAPI: {
    list: vi.fn(),
    exportCodes: vi.fn(),
    generate: vi.fn(),
    delete: vi.fn(),
    batchDelete: vi.fn()
  },
  groupsAPI: {
    getAll: vi.fn()
  },
  showError: vi.fn(),
  showSuccess: vi.fn(),
  routeQuery: {} as Record<string, string>
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    redeem: redeemAPI,
    groups: groupsAPI
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showInfo: vi.fn()
  })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn().mockResolvedValue(true)
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, fallback?: string) => fallback || key
    })
  }
})

import RedeemView from '../RedeemView.vue'

function mountView() {
  return mount(RedeemView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<section><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>'
        },
        DataTable: { props: ['data'], template: '<div />' },
        Pagination: true,
        ConfirmDialog: true,
        Select: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Icon: true,
        Teleport: true
      }
    }
  })
}

describe('admin RedeemView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    for (const key of Object.keys(routeQuery)) {
      delete routeQuery[key]
    }
    redeemAPI.list.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    groupsAPI.getAll.mockResolvedValue([])
  })

  it('uses route query as an investigation search keyword', async () => {
    routeQuery.search = 'customer@example.com'

    mountView()
    await flushPromises()

    expect(redeemAPI.list).toHaveBeenCalledWith(
      1,
      20,
      {
        type: '',
        status: '',
        search: 'customer@example.com'
      },
      expect.objectContaining({
        signal: expect.any(AbortSignal)
      })
    )
  })
})
