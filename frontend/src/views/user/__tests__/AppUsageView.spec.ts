import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { usageAPI, keysAPI, authStore, appStore, copyToClipboard } = vi.hoisted(() => ({
  usageAPI: {
    getStatsByDateRange: vi.fn(),
    query: vi.fn(),
    getDashboardTrend: vi.fn()
  },
  keysAPI: {
    list: vi.fn()
  },
  authStore: {
    user: {
      balance: 8.53,
      username: 'real-user',
      email: 'real@example.com'
    },
    refreshUser: vi.fn()
  },
  appStore: {
    showSuccess: vi.fn(),
    showError: vi.fn()
  },
  copyToClipboard: vi.fn()
}))

const messages: Record<string, string> = {
  'usage.workbench.createdAt': 'Created at',
  'usage.workbench.model': 'Model',
  'usage.workbench.amount': 'Usage',
  'usage.workbench.duration': 'Duration',
  'usage.workbench.fee': 'Fee',
  'usage.workbench.feeTooltip': 'Final charged amount',
  'usage.workbench.expandRow': 'Show details',
  'usage.workbench.collapseRow': 'Hide details',
  'usage.workbench.kind': 'Type',
  'usage.workbench.endpoint': 'Endpoint',
  'usage.workbench.billingBasis': 'Billing',
  'usage.workbench.performance': 'Processing time',
  'usage.workbench.supportCode': 'Support code',
  'usage.workbench.copySupportCode': 'Copy support code',
  'usage.workbench.supportCodeCopied': 'Support code copied',
  'usage.workbench.copy': 'Copy',
  'usage.workbench.copied': 'Copied',
  'usage.workbench.detailNote': 'Note',
  'usage.workbench.tokenBreakdown': 'Input {input} · Output {output} · Cache write {cacheWrite} · Cache read {cacheRead}',
  'usage.tokenDetails': 'Token breakdown',
  'usage.workbench.billingBalance': 'Balance charge',
  'usage.workbench.billingSubscription': 'Subscription quota',
  'usage.workbench.billingNoCharge': 'No balance charged',
  'usage.workbench.standardVsActual': 'Actual charge {actual}',
  'usage.workbench.actualChargeBasis': 'Actual charge {amount}',
  'usage.workbench.noChargeBasis': 'No balance was deducted for this row.',
  'usage.workbench.performanceHealthy': 'Normal',
  'usage.workbench.performanceUnknown': 'No record',
  'usage.workbench.performanceSlowFirstToken': 'Slightly slower start',
  'usage.workbench.performanceSlowTotal': 'Longer processing',
  'usage.workbench.performanceNoRecord': 'No displayable timing yet',
  'usage.workbench.performanceSummary': 'Started in {firstToken}; total time {duration}',
  'usage.workbench.performanceSlowFirstTokenHint': 'This request started slowly.',
  'usage.workbench.performanceSlowTotalHint': 'This request took longer to finish.',
  'usage.workbench.noCharge': 'No charge',
  'usage.workbench.zeroTokenCharged': 'Image / fixed-fee item',
  'usage.workbench.usageKindImage': 'Image generation',
  'usage.workbench.usageKindChat': 'Chat',
  'usage.workbench.usageKindThirdParty': 'Third-party access',
  'usage.workbench.usageKindWeb': 'Web app',
  'usage.workbench.imageAmount': '{count} images',
  'usage.workbench.imageAmountWithSize': '{count} images / {size}',
  'usage.workbench.tokenAmount': '{count} tokens',
  'usage.workbench.monthlyUsageSummary': 'This month: {requests} requests and {tokens} usage units.',
  'usage.workbench.monthLabel': 'Month {month}'
}

function translate(key: string, params?: Record<string, unknown>) {
  let value = messages[key] ?? key
  for (const [paramKey, paramValue] of Object.entries(params || {})) {
    value = value.replaceAll(`{${paramKey}}`, String(paramValue))
  }
  return value
}

vi.mock('@/api', () => ({ usageAPI, keysAPI }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard })
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: translate }) }
})
vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main><slot /></main>'
  }
}))
vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))
vi.mock('@/components/common/Select.vue', () => ({
  default: {
    props: ['modelValue', 'options'],
    emits: ['update:modelValue'],
    template: '<select />'
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

describe('AppUsageView compact usage details', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    keysAPI.list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    authStore.refreshUser.mockResolvedValue(authStore.user)
    copyToClipboard.mockResolvedValue(true)
    usageAPI.getStatsByDateRange.mockResolvedValue({
      total_requests: 3,
      total_tokens: 98,
      total_actual_cost: 0.43384
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-08-01',
      end_date: '2026-08-29',
      granularity: 'day'
    })
    usageAPI.query.mockResolvedValue({
      items: [
        {
          id: 101,
          request_id: 'req-usage-1',
          api_key_id: 7,
          model: 'gpt-5.5',
          inbound_endpoint: '/v1/responses',
          input_tokens: 60,
          output_tokens: 24,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          total_cost: 0.28,
          actual_cost: 0.28,
          billing_type: 0,
          image_count: 0,
          image_size: null,
          duration_ms: 3200,
          first_token_ms: 460,
          created_at: '2026-08-29T08:00:00Z'
        },
        {
          id: 102,
          request_id: 'req-usage-2',
          api_key_id: 7,
          model: 'gpt-5.5',
          inbound_endpoint: '/v1/responses',
          input_tokens: 10,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          total_cost: 0,
          actual_cost: 0,
          billing_type: 0,
          image_count: 0,
          image_size: null,
          duration_ms: 76000,
          first_token_ms: 6200,
          created_at: '2026-08-29T08:01:00Z'
        },
        {
          id: 103,
          request_id: 'req-usage-3',
          api_key_id: 7,
          model: 'gpt-5.5',
          inbound_endpoint: '/v1/responses',
          input_tokens: 4,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          total_cost: 0.00384,
          actual_cost: 0.00384,
          billing_type: 0,
          image_count: 0,
          image_size: null,
          duration_ms: 119999,
          first_token_ms: 700,
          created_at: '2026-08-29T08:02:00Z'
        }
      ],
      total: 3,
      pages: 1
    })
  })

  it('keeps scan-critical fields in the main row and reveals diagnostics on demand', async () => {
    const wrapper = mountView()
    await flushPromises()

    const headers = wrapper.findAll('thead th').map((header) => header.text())
    expect(headers.slice(0, 5)).toEqual(['Created at', 'Model', 'Usage', 'Duration', 'Fee'])
    expect(wrapper.findAll('tr.usage-row')).toHaveLength(3)
    expect(wrapper.findAll('tr.usage-detail-row')).toHaveLength(0)
    expect(wrapper.get('table').text()).not.toContain('/v1/responses')
    expect(wrapper.get('table').text()).not.toContain('req-usage-1')
    expect(wrapper.get('table').text()).not.toContain('Balance charge')

    const rows = wrapper.findAll('tr.usage-row')
    expect(rows[0].text()).toContain('$0.28')
    expect(rows[1].text()).toContain('$0.00')
    expect.soft(rows[2].text()).toContain('$0.00384')
    expect.soft(rows[2].text()).toContain('2m')
    rows.forEach((row) => expect.soft(row.text()).not.toContain('1m 60s'))

    await rows[0].trigger('click')

    expect(wrapper.findAll('tr.usage-detail-row')).toHaveLength(1)
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('/v1/responses')
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('req-usage-1')
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('Balance charge')
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('Input 60')

    await rows[1].trigger('click')

    expect(wrapper.findAll('tr.usage-detail-row')).toHaveLength(1)
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('req-usage-2')
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('Slightly slower start')
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('No charge')

    await wrapper.findAll('tr.usage-row')[1].trigger('click')
    expect(wrapper.findAll('tr.usage-detail-row')).toHaveLength(0)
    expect(usageAPI.query).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20
    }))
  })

  it('copies the support code without collapsing the expanded row', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('tr.usage-row').trigger('click')
    await wrapper.get('.support-code-button').trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenCalledWith('req-usage-1', 'Support code copied')
    expect(wrapper.findAll('tr.usage-detail-row')).toHaveLength(1)
  })

  it('aligns numeric headers and values on the same right edge', async () => {
    const wrapper = mountView()
    await flushPromises()

    const headers = wrapper.findAll('thead th.num-cell')
    const values = wrapper.get('tr.usage-row').findAll('td.num-cell')
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/AppUsageView.vue'), 'utf8')

    expect(headers).toHaveLength(3)
    expect(values).toHaveLength(3)
    expect(source).toMatch(/\.usage-table th\.num-cell,\s*\.usage-table td\.num-cell\s*{\s*text-align:\s*right;/)
  })
})
