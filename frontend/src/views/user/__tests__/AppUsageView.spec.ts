import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { usageAPI, authStore } = vi.hoisted(() => ({
  usageAPI: {
    getStatsByDateRange: vi.fn(),
    query: vi.fn(),
    getDashboardTrend: vi.fn()
  },
  authStore: {
    user: {
      balance: 8.53
    },
    refreshUser: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'usage.workbench.title': 'Usage information',
  'usage.workbench.subtitle': 'Review account balance and recent usage.',
  'usage.workbench.eyebrow': 'Account metering',
  'usage.workbench.balanceTitle': 'Account balance',
  'usage.workbench.balanceDescription': 'Available for in-app chat, image generation, and third-party client calls.',
  'usage.workbench.recharge': 'Recharge',
  'usage.workbench.monthlyCostTitle': 'Current-month spend',
  'usage.workbench.unavailable': 'Unavailable',
  'usage.workbench.noRealUsageNote': 'No real usage records this month.',
  'usage.workbench.monthlyUsageSummary': 'This month: {requests} requests and {tokens} tokens.',
  'usage.workbench.statsLoadError': 'Monthly usage stats are temporarily unavailable. Refresh to retry.',
  'usage.workbench.billingExplanationTitle': 'Billing explanation',
  'usage.workbench.billingExplanationDescription': 'Backend-recorded real usage is the source of truth. The frontend does not decide prices.',
  'usage.workbench.billingExplanationItems.successCharged': 'Successful calls show the actual charge in usage details.',
  'usage.workbench.billingExplanationItems.failureNoCharge': 'Failed requests show as no charge or do not create a charge record.',
  'usage.workbench.billingExplanationItems.zeroCost': 'A $0.0000 fee means this record was not actually charged.',
  'usage.workbench.monthlyUsageTitle': 'Monthly usage',
  'usage.workbench.monthlyUsageDescription': 'Empty data is not filled with fake bars.',
  'usage.workbench.realDataBadge': 'Real data',
  'usage.workbench.monthlyChartLabel': 'Monthly usage chart',
  'usage.workbench.noMonthlyUsageTitle': 'No monthly usage data yet',
  'usage.workbench.noMonthlyUsageDescription': 'Real usage trends appear here.',
  'usage.workbench.trendLoadError': 'Monthly trend is temporarily unavailable',
  'usage.workbench.trendLoadErrorHint': 'API failures are not presented as an empty trend.',
  'usage.workbench.usageDetailsTitle': 'Usage details',
  'usage.workbench.usageDetailsDescription': 'Verify model, type, usage, and charge.',
  'usage.workbench.refresh': 'Refresh',
  'usage.workbench.loading': 'Loading usage',
  'usage.workbench.detailsLoadError': 'Usage details are temporarily unavailable',
  'usage.workbench.detailsLoadErrorHint': 'This page will not jump to the legacy usage page.',
  'usage.workbench.noDetailsTitle': 'No usage details yet',
  'usage.workbench.noDetailsDescription': 'Real calls will appear here.',
  'usage.workbench.createdAt': 'Created at',
  'usage.workbench.kind': 'Type',
  'usage.workbench.model': 'Model',
  'usage.workbench.amount': 'Usage',
  'usage.workbench.fee': 'Fee',
  'usage.workbench.noCharge': 'No charge',
  'usage.workbench.usageKindImage': 'Image generation',
  'usage.workbench.usageKindChat': 'Chat',
  'usage.workbench.usageKindThirdParty': 'Third-party access',
  'usage.workbench.usageKindWeb': 'Web app',
  'usage.workbench.imageAmount': '{count} images',
  'usage.workbench.imageAmountWithSize': '{count} images / {size}',
  'usage.workbench.tokenAmount': '{count} tokens',
  'usage.workbench.requestCount': '{count} requests',
  'usage.workbench.monthLabel': 'Month {month}'
}

function translate(key: string, params?: Record<string, unknown>) {
  let value = messages[key] ?? key
  for (const [paramKey, paramValue] of Object.entries(params || {})) {
    value = value.replaceAll(`{${paramKey}}`, String(paramValue))
  }
  return value
}

vi.mock('@/api', () => ({
  usageAPI
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: translate
    })
  }
})

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: `
      <main data-testid="app-section-shell">
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
        <slot />
      </main>
    `
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import AppUsageView from '../AppUsageView.vue'

function mountView() {
  return mount(AppUsageView, {
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

function mockZeroStats() {
  usageAPI.getStatsByDateRange.mockResolvedValue({
    total_requests: 0,
    total_input_tokens: 0,
    total_output_tokens: 0,
    total_cache_tokens: 0,
    total_tokens: 0,
    total_cost: 0,
    total_actual_cost: 0,
    average_duration_ms: 0
  })
}

describe('AppUsageView', () => {
  beforeEach(() => {
    usageAPI.getStatsByDateRange.mockReset()
    usageAPI.query.mockReset()
    usageAPI.getDashboardTrend.mockReset()
    authStore.refreshUser.mockReset()
    authStore.refreshUser.mockResolvedValue(authStore.user)
    authStore.user.balance = 8.53
  })

  it('renders usage data inside the new workbench page instead of the old usage UI', async () => {
    usageAPI.getStatsByDateRange.mockResolvedValue({
      total_requests: 3,
      total_input_tokens: 57,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 57,
      total_cost: 1.2345,
      total_actual_cost: 1.2345,
      average_duration_ms: 1200
    })
    usageAPI.query.mockResolvedValue({
      items: [
        {
          id: 7,
          request_id: 'req-image',
          model: 'gpt-image-2',
          inbound_endpoint: '/v1/images/generations',
          input_tokens: 0,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          image_count: 2,
          image_size: '1024x1024',
          actual_cost: 0.88,
          created_at: '2026-06-18T08:00:00Z'
        },
        {
          id: 8,
          request_id: 'req-no-charge',
          api_key_id: 1,
          model: 'deepseek-v4-flash',
          inbound_endpoint: '/v1/chat/completions',
          input_tokens: 12,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          image_count: 0,
          image_size: null,
          actual_cost: 0,
          created_at: '2026-06-18T08:01:00Z'
        },
        {
          id: 9,
          request_id: 'req-chat',
          model: 'gpt-5-mini',
          inbound_endpoint: '/v1/chat/completions',
          input_tokens: 45,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          image_count: 0,
          image_size: null,
          actual_cost: 0.3545,
          created_at: '2026-06-18T08:02:00Z'
        }
      ],
      total: 3,
      pages: 1
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [
        {
          date: '2026-06-18',
          requests: 3,
          input_tokens: 57,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          total_tokens: 57,
          cost: 1.2345,
          actual_cost: 1.2345
        }
      ],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('Usage information')
    expect(text).toContain('Account balance')
    expect(text).toContain('$8.53')
    expect(text).toContain('Current-month spend')
    expect(text).toContain('$1.2345')
    expect(text).toContain('This month: 3 requests and 57 tokens.')
    expect(text).toContain('Monthly usage')
    expect(text).toContain('Real data')
    expect(text).toContain('Billing explanation')
    expect(text).toContain('Backend-recorded real usage is the source of truth')
    expect(text).toContain('The frontend does not decide prices')
    expect(text).toContain('Failed requests show as no charge')
    expect(text).toContain('Usage details')
    const tableText = wrapper.get('table').text()
    expect(tableText).toContain('Image generation')
    expect(tableText).toContain('Third-party access')
    expect(tableText).toContain('Chat')
    expect(text).toContain('gpt-image-2')
    expect(text).toContain('2 images / 1024x1024')
    expect(text).toContain('deepseek-v4-flash')
    expect(text).toContain('$0.0000')
    expect(text).toContain('No charge')
    expect(usageAPI.query).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 8
    }))
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
  })

  it('refreshes the authenticated user balance when the usage overview is refreshed', async () => {
    mockZeroStats()
    usageAPI.query.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)

    await wrapper.get('button.refresh-button').trigger('click')
    await flushPromises()

    expect(authStore.refreshUser).toHaveBeenCalledTimes(2)
  })

  it('shows empty states without inventing usage rows when the existing APIs have no data', async () => {
    mockZeroStats()
    usageAPI.query.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('$0.0000')
    expect(text).toContain('No real usage records this month.')
    expect(text).toContain('No monthly usage data yet')
    expect(text).toContain('No usage details yet')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  it('keeps the workbench page in place when usage details cannot load', async () => {
    mockZeroStats()
    usageAPI.query.mockRejectedValue(new Error('unavailable'))
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Usage details are temporarily unavailable')
    expect(wrapper.text()).toContain('This page will not jump to the legacy usage page')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  it('does not present monthly stats failures as zero usage', async () => {
    usageAPI.getStatsByDateRange.mockRejectedValue(new Error('stats unavailable'))
    usageAPI.query.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    const summaryCards = wrapper.findAll('.usage-summary-card')
    expect(text).toContain('Unavailable')
    expect(text).toContain('Monthly usage stats are temporarily unavailable')
    expect(summaryCards[1].find('strong').text()).toBe('Unavailable')
    expect(summaryCards[1].text()).not.toContain('$0.0000')
    expect(summaryCards[1].text()).not.toContain('No real usage records this month.')
  })

  it('does not present monthly trend failures as empty trend data', async () => {
    mockZeroStats()
    usageAPI.query.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0
    })
    usageAPI.getDashboardTrend.mockRejectedValue(new Error('trend unavailable'))

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('Monthly trend is temporarily unavailable')
    expect(text).toContain('API failures are not presented as an empty trend')
    expect(text).not.toContain('No monthly usage data yet')
  })
})
