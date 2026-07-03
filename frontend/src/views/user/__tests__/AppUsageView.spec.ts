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
      balance: 8.53,
      username: 'real-user',
      email: 'real@example.com'
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
  'usage.workbench.balanceRefreshError': 'Balance could not be refreshed. The value shown may be stale; please retry shortly.',
  'usage.workbench.recharge': 'Recharge',
  'usage.workbench.monthlyCostTitle': 'Current-month spend',
  'usage.workbench.unavailable': 'Unavailable',
  'usage.workbench.noRealUsageNote': 'No usage records this month.',
  'usage.workbench.monthlyUsageSummary': 'This month: {requests} requests and {tokens} usage units.',
  'usage.workbench.statsLoadError': 'Monthly usage stats are temporarily unavailable. Refresh to retry.',
  'usage.workbench.billingExplanationTitle': 'Billing explanation',
  'usage.workbench.billingExplanationDescription': 'Usage records show the final charge result. Completed calls display the charged amount; failed or unfinished requests are not shown as charged.',
  'usage.workbench.billingExplanationItems.successCharged': 'Completed calls show the final charge in usage details.',
  'usage.workbench.billingExplanationItems.failureNoCharge': 'Failed or unfinished requests show as no charge, or do not create a charge record.',
  'usage.workbench.billingExplanationItems.zeroCost': 'A $0.0000 fee means no balance was deducted for this row.',
  'usage.workbench.monthlyUsageTitle': 'Monthly usage',
  'usage.workbench.monthlyUsageDescription': 'Empty data is not filled with fake bars.',
  'usage.workbench.realDataBadge': 'Real data',
  'usage.workbench.demoDataBadge': 'Demo data',
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
  'usage.workbench.endpoint': 'Endpoint',
  'usage.workbench.model': 'Model',
  'usage.workbench.source': 'Source',
  'usage.workbench.amount': 'Usage',
  'usage.workbench.billingBasis': 'Billing',
  'usage.workbench.billingBalance': 'Balance charge',
  'usage.workbench.billingSubscription': 'Subscription quota',
  'usage.workbench.billingNoCharge': 'No balance charged',
  'usage.workbench.standardVsActual': 'Actual charge {actual}',
  'usage.workbench.actualChargeBasis': 'Actual charge {amount}',
  'usage.workbench.noChargeBasis': 'No balance was deducted for this row.',
  'usage.workbench.performance': 'Processing time',
  'usage.workbench.performanceHealthy': 'Normal',
  'usage.workbench.performanceUnknown': 'No record',
  'usage.workbench.performanceSlowFirstToken': 'Slightly slower start',
  'usage.workbench.performanceSlowTotal': 'Longer processing',
  'usage.workbench.performanceNoRecord': 'No displayable timing yet',
  'usage.workbench.performanceSummary': 'Started in {firstToken}; total time {duration}',
  'usage.workbench.performanceSlowFirstTokenHint': 'This task was slightly slower to start, usually because of model tier, task complexity, or extra processing steps. For faster replies, use a lighter model or lower reasoning effort.',
  'usage.workbench.performanceSlowTotalHint': 'This task took longer, usually because it did more checking or used more steps. For faster replies, use a lighter model, lower reasoning effort, or shorten one-shot input.',
  'usage.workbench.fee': 'Fee',
  'usage.workbench.noCharge': 'No charge',
  'usage.workbench.zeroTokenCharged': 'Image / fixed-fee item',
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
    authStore.user.username = 'real-user'
    authStore.user.email = 'real@example.com'
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
          total_cost: 0.9,
          image_count: 2,
          image_size: '1024x1024',
          actual_cost: 0.88,
          billing_type: 0,
          duration_ms: 207544,
          first_token_ms: 507,
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
          total_cost: 0,
          image_count: 0,
          image_size: null,
          actual_cost: 0,
          billing_type: 0,
          duration_ms: 6400,
          first_token_ms: 6200,
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
          total_cost: 0.3545,
          image_count: 0,
          image_size: null,
          actual_cost: 0.3545,
          billing_type: 1,
          duration_ms: 987,
          first_token_ms: 123,
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
    expect(text).toContain('This month: 3 requests and 57 usage units.')
    expect(text).toContain('Monthly usage')
    expect(text).toContain('Real data')
    expect(text).toContain('Billing explanation')
    expect(text).toContain('Usage records show the final charge result')
    expect(text).toContain('Completed calls display the charged amount')
    expect(text).toContain('Failed or unfinished requests show as no charge')
    expect(text).toContain('Usage details')
    const tableText = wrapper.get('table').text()
    expect(tableText).toContain('Image generation')
    expect(tableText).toContain('Third-party access')
    expect(tableText).toContain('Chat')
    expect(tableText).toContain('Endpoint')
    expect(tableText).toContain('Source')
    expect(tableText).toContain('/v1/images/generations')
    expect(tableText).toContain('/v1/chat/completions')
    expect(text).toContain('gpt-image-2')
    expect(text).toContain('2 images / 1024x1024')
    expect(tableText).toContain('Image / fixed-fee item')
    expect(text).toContain('deepseek-v4-flash')
    expect(text).toContain('$0.0000')
    expect(text).toContain('No charge')
    expect(tableText).toContain('Billing')
    expect(tableText).toContain('Balance charge')
    expect(tableText).toContain('Subscription quota')
    expect(tableText).toContain('No balance charged')
    expect(tableText).toContain('Actual charge $0.8800')
    expect(tableText).toContain('Actual charge $0.3545')
    expect(tableText).toContain('No balance was deducted for this row.')
    expect(tableText).toContain('Processing time')
    expect(tableText).toContain('Longer processing')
    expect(tableText).toContain('Started in 507 ms; total time 3m 28s')
    expect(tableText).toContain('did more checking or used more steps')
    expect(tableText).toContain('lighter model, lower reasoning effort')
    expect(tableText).toContain('Slightly slower start')
    expect(tableText).toContain('Started in 6.2 s; total time 6.4 s')
    expect(tableText).toContain('task complexity, or extra processing steps')
    expect(tableText).toContain('Normal')
    expect(tableText).toContain('Started in 123 ms; total time 987 ms')
    expect(tableText).not.toContain('upstream account')
    expect(tableText).not.toContain('web search')
    expect(tableText).not.toContain('tool calls')
    expect(usageAPI.query).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 8
    }))
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
  })

  it('marks local mock usage as demo data instead of real data', async () => {
    authStore.user.username = 'demo-user'
    authStore.user.email = 'user@example.local'
    usageAPI.getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0.25,
      total_actual_cost: 0.25,
      average_duration_ms: 0
    })
    usageAPI.query.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [
        {
          date: '2026-06-18',
          requests: 1,
          input_tokens: 0,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          total_tokens: 0,
          cost: 0.25,
          actual_cost: 0.25
        }
      ],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Demo data')
    expect(wrapper.text()).not.toContain('Real data')
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

  it('marks the balance as stale when authenticated user refresh fails', async () => {
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
    authStore.refreshUser
      .mockRejectedValueOnce(new Error('session refresh failed'))
      .mockResolvedValueOnce(authStore.user)

    const wrapper = mountView()
    await flushPromises()

    let balanceCard = wrapper.findAll('.usage-summary-card')[0]
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
    expect(balanceCard.text()).toContain('$8.53')
    expect(balanceCard.text()).toContain('Balance could not be refreshed')
    expect(balanceCard.text()).not.toContain('Available for in-app chat')
    expect(balanceCard.find('p.is-warning').exists()).toBe(true)

    await wrapper.get('button.refresh-button').trigger('click')
    await flushPromises()

    balanceCard = wrapper.findAll('.usage-summary-card')[0]
    expect(authStore.refreshUser).toHaveBeenCalledTimes(2)
    expect(balanceCard.text()).toContain('Available for in-app chat')
    expect(balanceCard.text()).not.toContain('Balance could not be refreshed')
    expect(balanceCard.find('p.is-warning').exists()).toBe(false)
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
    expect(text).toContain('No usage records this month.')
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
    expect(summaryCards[1].text()).not.toContain('No usage records this month.')
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
