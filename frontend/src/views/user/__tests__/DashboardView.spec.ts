import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore, authStore, usageAPI } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {
      channel_monitor_enabled: true
    }
  },
  authStore: {
    user: {
      email: 'user@example.com',
      balance: 42.35
    },
    isSimpleMode: false,
    refreshUser: vi.fn()
  },
  usageAPI: {
    getDashboardStats: vi.fn(),
    getDashboardTrend: vi.fn(),
    getDashboardModels: vi.fn(),
    getByDateRange: vi.fn()
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/api/usage', () => ({
  usageAPI
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="app-layout"><slot /></main>'
  }
}))

vi.mock('@/components/common/LoadingSpinner.vue', () => ({
  default: {
    name: 'LoadingSpinner',
    template: '<div data-testid="loading-spinner" />'
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name'],
    template: '<span class="icon-stub" :data-icon="name" />'
  }
}))

vi.mock('@/components/user/dashboard/UserDashboardStats.vue', () => ({
  default: {
    name: 'UserDashboardStats',
    props: ['stats', 'balance', 'isSimple'],
    template: '<section data-testid="dashboard-stats-child">{{ stats.total_requests }} {{ balance }} {{ String(isSimple) }}</section>'
  }
}))

vi.mock('@/components/user/dashboard/UserDashboardCharts.vue', () => ({
  default: {
    name: 'UserDashboardCharts',
    props: ['trend', 'models', 'loading'],
    template: '<section data-testid="dashboard-charts-child">{{ trend.length }} {{ models.length }} {{ String(loading) }}</section>'
  }
}))

vi.mock('@/components/user/dashboard/UserDashboardRecentUsage.vue', () => ({
  default: {
    name: 'UserDashboardRecentUsage',
    props: ['data', 'loading'],
    template: '<section data-testid="dashboard-recent-usage-child">{{ data.length }} {{ String(loading) }}</section>'
  }
}))

vi.mock('@/components/user/dashboard/UserDashboardQuickActions.vue', () => ({
  default: {
    name: 'UserDashboardQuickActions',
    template: '<section data-testid="dashboard-quick-actions-child" />'
  }
}))

import DashboardView from '../DashboardView.vue'

function dashboardStatsFixture() {
  return {
    total_api_keys: 3,
    active_api_keys: 2,
    total_requests: 12345,
    total_input_tokens: 1000,
    total_output_tokens: 800,
    total_cache_creation_tokens: 0,
    total_cache_read_tokens: 0,
    total_tokens: 1800,
    total_cost: 9.99,
    total_actual_cost: 8.75,
    today_requests: 27,
    today_input_tokens: 120,
    today_output_tokens: 80,
    today_cache_creation_tokens: 0,
    today_cache_read_tokens: 0,
    today_tokens: 200,
    today_cost: 1.5,
    today_actual_cost: 1.25,
    average_duration_ms: 900,
    rpm: 3,
    tpm: 120
  }
}

function mountView() {
  return mount(DashboardView, {
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>'
        }
      }
    }
  })
}

describe('DashboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authStore.user.email = 'user@example.com'
    authStore.user.balance = 42.35
    authStore.isSimpleMode = false
    appStore.cachedPublicSettings = {
      channel_monitor_enabled: true
    }
    authStore.refreshUser.mockResolvedValue(authStore.user)
    usageAPI.getDashboardStats.mockResolvedValue(dashboardStatsFixture())
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [
        {
          date: '2026-07-02',
          requests: 27,
          input_tokens: 120,
          output_tokens: 80,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          total_tokens: 200,
          cost: 1.5,
          actual_cost: 1.25
        }
      ],
      start_date: '2026-06-26',
      end_date: '2026-07-02',
      granularity: 'day'
    })
    usageAPI.getDashboardModels.mockResolvedValue({
      models: [
        {
          model: 'gpt-4o-mini',
          requests: 27,
          total_tokens: 200,
          total_cost: 1.5,
          total_actual_cost: 1.25
        }
      ],
      start_date: '2026-06-26',
      end_date: '2026-07-02'
    })
    usageAPI.getByDateRange.mockResolvedValue({
      items: [
        {
          id: 1,
          request_id: 'req-1',
          model: 'gpt-4o-mini',
          input_tokens: 120,
          output_tokens: 80,
          total_cost: 1.5,
          actual_cost: 1.25,
          created_at: '2026-07-02T08:00:00Z'
        }
      ],
      total: 1,
      pages: 1
    })
  })

  it('renders the real operating entry with balance, usage, channel, order, and API key links', async () => {
    const wrapper = mountView()
    await flushPromises()

    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('user@example.com')
    expect(wrapper.text()).toContain('$42.35')
    expect(wrapper.text()).toContain('$1.25')
    expect(wrapper.text()).toContain('27')
    expect(hrefs).toContain('/app/keys')
    expect(hrefs).toContain('/app/usage')
    expect(hrefs).toContain('/app/channel-status')
    expect(hrefs).toContain('/app/purchase')
    expect(hrefs).toContain('/app/orders')
    expect(hrefs).toContain('/app/chat')
    expect(wrapper.find('[data-testid="dashboard-stats-child"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-charts-child"]').text()).toContain('1 1')
    expect(wrapper.find('[data-testid="dashboard-recent-usage-child"]').text()).toContain('1')
    expect(wrapper.find('[data-testid="dashboard-quick-actions-child"]').exists()).toBe(true)
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardModels).toHaveBeenCalledTimes(1)
    expect(usageAPI.getByDateRange).toHaveBeenCalledTimes(1)
  })

  it('hides the channel status shortcut when channel monitoring is disabled', async () => {
    appStore.cachedPublicSettings = {
      channel_monitor_enabled: false
    }

    const wrapper = mountView()
    await flushPromises()

    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).not.toContain('/app/channel-status')
  })

  it('shows a retryable error state instead of a blank page when stats fail', async () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    usageAPI.getDashboardStats
      .mockRejectedValueOnce(new Error('stats unavailable'))
      .mockResolvedValueOnce(dashboardStatsFixture())

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="dashboard-load-error"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-stats-child"]').exists()).toBe(false)

    await wrapper.get('[data-testid="dashboard-retry"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="dashboard-load-error"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-stats-child"]').exists()).toBe(true)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(2)

    errorSpy.mockRestore()
  })
})
