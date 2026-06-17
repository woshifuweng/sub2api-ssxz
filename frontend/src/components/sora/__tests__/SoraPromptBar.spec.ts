import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SoraPromptBar from '../SoraPromptBar.vue'

const mocks = vi.hoisted(() => ({
  fetchActiveSubscriptions: vi.fn(),
  getModels: vi.fn(),
  getStorageStatus: vi.fn(),
  listKeys: vi.fn()
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/api/sora', () => ({
  default: {
    getModels: mocks.getModels,
    getStorageStatus: mocks.getStorageStatus
  }
}))

vi.mock('@/api/keys', () => ({
  default: {
    list: mocks.listKeys
  }
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({
    fetchActiveSubscriptions: mocks.fetchActiveSubscriptions
  })
}))

describe('SoraPromptBar storage status', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getModels.mockResolvedValue([
      {
        id: 'sora2',
        name: 'Sora 2',
        type: 'video',
        orientations: ['landscape'],
        durations: [10]
      }
    ])
    mocks.listKeys.mockResolvedValue({ items: [] })
    mocks.fetchActiveSubscriptions.mockResolvedValue([])
  })

  function mountPromptBar() {
    return mount(SoraPromptBar, {
      props: {
        activeTaskCount: 0,
        generating: false,
        maxConcurrentTasks: 3
      }
    })
  }

  it('does not show the missing storage badge when local storage is enabled', async () => {
    mocks.getStorageStatus.mockResolvedValue({
      local_enabled: true,
      s3_enabled: false,
      s3_healthy: false
    })

    const wrapper = mountPromptBar()
    await flushPromises()

    expect(wrapper.text()).not.toContain('sora.noStorageConfigured')
  })

  it('shows the missing storage badge when no storage backend is enabled', async () => {
    mocks.getStorageStatus.mockResolvedValue({
      local_enabled: false,
      s3_enabled: false,
      s3_healthy: false
    })

    const wrapper = mountPromptBar()
    await flushPromises()

    expect(wrapper.text()).toContain('sora.noStorageConfigured')
  })
})
