import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore, authStore, usageAPI } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {
      channel_monitor_enabled: true,
      payment_enabled: true,
      affiliate_enabled: false
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
    getByDateRange: vi.fn(),
    list: vi.fn()
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

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main data-testid="app-section-shell"><slot /></main>'
  }
}))

vi.mock('@/components/user/BalanceWarningBanner.vue', () => ({
  default: {
    name: 'BalanceWarningBanner',
    template: '<div data-testid="balance-warning-banner" />'
  }
}))

vi.mock('@/components/common/LoadingSpinner.vue', () => ({
  default: {
    name: 'LoadingSpinner',
    template: '<div data-testid="loading-spinner" />'
  }
}))

vi.mock('@/components/foundation', () => ({
  FoundationProvider: {
    name: 'FoundationProvider',
    props: ['theme'],
    template: '<div data-testid="dashboard-foundation" :data-theme="theme"><slot /></div>'
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
    props: ['stats', 'balance', 'isSimple', 'trend', 'todayTrend'],
    template: '<section data-testid="dashboard-stats-child">{{ stats.total_requests }} {{ balance }} {{ String(isSimple) }} {{ trend.length }} {{ todayTrend.length }}</section>'
  }
}))

vi.mock('@/components/user/dashboard/UserDashboardCharts.vue', () => ({
  default: {
    name: 'UserDashboardCharts',
    props: ['trend', 'models', 'loading', 'lastCallAt'],
    emits: ['extendRange'],
    template: '<section data-testid="dashboard-charts-child">{{ trend.length }} {{ models.length }} {{ String(loading) }} {{ lastCallAt }}</section>'
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
      channel_monitor_enabled: true,
      payment_enabled: true,
      affiliate_enabled: false
    }
    authStore.refreshUser.mockResolvedValue(authStore.user)
    usageAPI.getDashboardStats.mockResolvedValue(dashboardStatsFixture())
    usageAPI.getDashboardTrend.mockImplementation((params?: { granularity?: string }) => Promise.resolve({
      trend: params?.granularity === 'hour'
        ? [
            {
              date: '2026-07-02T08:00:00',
              requests: 7,
              input_tokens: 30,
              output_tokens: 20,
              cache_creation_tokens: 0,
              cache_read_tokens: 0,
              total_tokens: 50,
              cost: 0.4,
              actual_cost: 0.3
            },
            {
              date: '2026-07-02T09:00:00',
              requests: 20,
              input_tokens: 90,
              output_tokens: 60,
              cache_creation_tokens: 0,
              cache_read_tokens: 0,
              total_tokens: 150,
              cost: 1.1,
              actual_cost: 0.95
            }
          ]
        : [
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
      granularity: params?.granularity || 'day'
    }))
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
    usageAPI.list.mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  it('keeps real dashboard data while removing duplicated workbench and service directory layers', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-foundation"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('账户工作台')
    expect(wrapper.text()).not.toContain('管理常用功能')
    expect(wrapper.text()).toContain('42.35')
    expect(wrapper.text()).not.toContain('模型测试入口')
    expect(wrapper.find('[data-testid="dashboard-onboarding"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-stats-child"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-stats-child"]').text()).toContain('12345 42.35 false 1 2')
    expect(wrapper.find('[data-testid="dashboard-charts-child"]').text()).toContain('1 1')
    expect(wrapper.find('[data-testid="dashboard-recent-usage-child"]').text()).toContain('1')
    expect(wrapper.find('[data-testid="dashboard-quick-actions-child"]').exists()).toBe(true)
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledTimes(2)
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledWith(expect.objectContaining({ granularity: 'hour' }))
    expect(usageAPI.getDashboardModels).toHaveBeenCalledTimes(1)
    expect(usageAPI.getByDateRange).toHaveBeenCalledTimes(1)
  })

  it('shows the onboarding guide only when the account has no API keys', async () => {
    usageAPI.getDashboardStats.mockResolvedValueOnce({
      ...dashboardStatsFixture(),
      total_api_keys: 0,
      active_api_keys: 0
    })

    const wrapper = mountView()
    await flushPromises()

    const onboarding = wrapper.get('[data-testid="dashboard-onboarding"]')
    const hrefs = onboarding.findAll('a').map((link) => link.attributes('href'))
    expect(onboarding.element.tagName).toBe('DETAILS')
    expect(onboarding.attributes('open')).toBeUndefined()
    expect(onboarding.text()).toContain('首次接入指南')
    expect(hrefs).toContain('/app/keys')
    expect(hrefs).toContain('/app/docs')
    expect(hrefs).toContain('/app/purchase')
  })

  it('keeps recharge and order language consistent when online payment is disabled', async () => {
    appStore.cachedPublicSettings = {
      channel_monitor_enabled: true,
      payment_enabled: false,
      affiliate_enabled: false
    }
    usageAPI.getDashboardStats.mockResolvedValueOnce({
      ...dashboardStatsFixture(),
      total_api_keys: 0,
      active_api_keys: 0
    })

    const wrapper = mountView()
    await flushPromises()

    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).not.toContain('/app/purchase')
    expect(wrapper.text()).not.toContain('购买套餐')
    expect(wrapper.text()).not.toContain('去充值')
    expect(wrapper.text()).not.toContain('充值和订单')
    expect(wrapper.text()).toContain('去兑换')
    expect(hrefs).toContain('/app/redeem')
    expect(wrapper.text()).toContain('兑换码')
  })

  it('resolves the latest call date and switches to a 30-day range when the selected range is empty', async () => {
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-07-20',
      end_date: '2026-07-26',
      granularity: 'day'
    })
    usageAPI.getDashboardModels.mockResolvedValue({
      models: [],
      start_date: '2026-07-20',
      end_date: '2026-07-26'
    })
    usageAPI.list.mockResolvedValue({
      items: [
        {
          id: 5,
          request_id: 'req-5',
          model: 'gpt-4o-mini',
          input_tokens: 10,
          output_tokens: 5,
          total_cost: 0.1,
          actual_cost: 0.08,
          created_at: '2026-05-02T10:00:00Z'
        }
      ],
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    expect(usageAPI.list).toHaveBeenCalledWith(1, 1)
    const charts = wrapper.findComponent({ name: 'UserDashboardCharts' })
    expect(charts.props('lastCallAt')).toBe('2026-05-02T10:00:00Z')

    usageAPI.getDashboardTrend.mockClear()
    usageAPI.getDashboardModels.mockClear()
    charts.vm.$emit('extendRange')
    await flushPromises()

    const formatLD = (date: Date) => [
      date.getFullYear(),
      String(date.getMonth() + 1).padStart(2, '0'),
      String(date.getDate()).padStart(2, '0')
    ].join('-')
    const expectedStartDate = new Date()
    expectedStartDate.setDate(expectedStartDate.getDate() - 29)
    const expectedStart = formatLD(expectedStartDate)
    const expectedEnd = formatLD(new Date())
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledWith(expect.objectContaining({
      start_date: expectedStart,
      end_date: expectedEnd,
      granularity: 'day'
    }))
    expect(usageAPI.getDashboardModels).toHaveBeenCalledWith({
      start_date: expectedStart,
      end_date: expectedEnd
    })
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
