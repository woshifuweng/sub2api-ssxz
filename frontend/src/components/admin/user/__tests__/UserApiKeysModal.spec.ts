import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { adminAPI, showError, showSuccess } = vi.hoisted(() => ({
  adminAPI: {
    users: {
      getUserApiKeys: vi.fn()
    },
    groups: {
      getAll: vi.fn()
    },
    apiKeys: {
      updateApiKeyGroup: vi.fn()
    }
  },
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

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import UserApiKeysModal from '../UserApiKeysModal.vue'

const activeUser = {
  id: 42,
  email: 'customer@example.com',
  username: 'customer',
  role: 'user',
  status: 'active',
  balance: 20,
  concurrency: 1,
  allowed_groups: [],
  subscriptions: [],
  created_at: '2026-07-01T00:00:00Z'
}

const makeKey = (overrides: Record<string, unknown> = {}) => ({
  id: 100,
  user_id: 42,
  key: 'sk-test-secret-value-1234567890',
  name: 'client-key',
  group_id: 7,
  group_ids: [7],
  allowed_models: [],
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: '2026-07-07T10:00:00Z',
  quota: 0,
  quota_used: 0,
  expires_at: null,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  group: {
    id: 7,
    name: 'default',
    platform: 'openai',
    status: 'active',
    rate_multiplier: 1,
    subscription_type: 'standard'
  },
  groups: [],
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
  ...overrides
})

function mountModal(user = activeUser) {
  return mount(UserApiKeysModal, {
    props: {
      show: true,
      user
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<section v-if="show"><slot /></section>'
        },
        GroupBadge: {
          props: ['name'],
          template: '<span data-testid="group-badge">{{ name }}</span>'
        },
        GroupOptionItem: true,
        Teleport: true
      }
    }
  })
}

describe('UserApiKeysModal readiness summary', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminAPI.groups.getAll.mockResolvedValue([])
  })

  it('mounts safely while the modal is closed', () => {
    expect(() => mount(UserApiKeysModal, {
      props: {
        show: false,
        user: null
      },
      global: {
        stubs: {
          BaseDialog: true,
          GroupBadge: true,
          GroupOptionItem: true,
          Teleport: true
        }
      }
    })).not.toThrow()
  })

  it('marks an active funded grouped key as deliverable', async () => {
    adminAPI.users.getUserApiKeys.mockResolvedValue({
      items: [makeKey()],
      total: 1,
      page: 1,
      page_size: 10,
      pages: 1
    })

    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.find('[data-testid="api-key-readiness-summary"]').text()).toContain('可交付 1 个')
    expect(wrapper.find('[data-testid="api-key-readiness-label"]').text()).toContain('可交付')
    expect(wrapper.find('[data-testid="api-key-readiness-notes"]').text()).toContain('状态、余额、分组、额度看起来可用')
  })

  it('blocks handoff when the user balance is empty', async () => {
    adminAPI.users.getUserApiKeys.mockResolvedValue({
      items: [makeKey()],
      total: 1,
      page: 1,
      page_size: 10,
      pages: 1
    })

    const wrapper = mountModal({ ...activeUser, balance: 0 })
    await flushPromises()

    expect(wrapper.find('[data-testid="api-key-readiness-summary"]').text()).toContain('需处理 1 个')
    expect(wrapper.find('[data-testid="api-key-readiness-label"]').text()).toContain('需处理')
    expect(wrapper.find('[data-testid="api-key-readiness-notes"]').text()).toContain('账户余额不足')
  })

  it('surfaces key-level quota and model restrictions for operators', async () => {
    adminAPI.users.getUserApiKeys.mockResolvedValue({
      items: [
        makeKey({
          quota: 10,
          quota_used: 10,
          allowed_models: ['gpt-5.5'],
          last_used_at: null
        })
      ],
      total: 1,
      page: 1,
      page_size: 10,
      pages: 1
    })

    const wrapper = mountModal()
    await flushPromises()

    const notes = wrapper.find('[data-testid="api-key-readiness-notes"]').text()
    expect(wrapper.find('[data-testid="api-key-readiness-label"]').text()).toContain('需处理')
    expect(notes).toContain('Key 额度已用完')
    expect(notes).toContain('已限制可用模型')
    expect(notes).toContain('暂无调用记录')
  })
})
