import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AppUsageView from '../AppUsageView.vue'
import UsageView from '../UsageView.vue'
import UsageBalanceSummary from '@/components/user/UsageBalanceSummary.vue'
import UsageTable from '@/components/admin/usage/UsageTable.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import UserErrorRequestsTable from '@/components/user/UserErrorRequestsTable.vue'

const { api, auth, app, groups } = vi.hoisted(() => ({
  api: { query: vi.fn(), getStats: vi.fn(), getStatsByDateRange: vi.fn(), getDashboardModels: vi.fn(), getDashboardSnapshotV2: vi.fn(), listMyErrorRequests: vi.fn() },
  auth: { user: { balance: 8.53 }, refreshUser: vi.fn() },
  app: { cachedPublicSettings: { allow_user_view_error_requests: true }, showSuccess: vi.fn(), showError: vi.fn(), showWarning: vi.fn(), showInfo: vi.fn() },
  groups: vi.fn(),
}))
vi.mock('@/api', () => ({ usageAPI: api, keysAPI: { list: vi.fn(async () => ({ items: [], pages: 0 })) }, userGroupsAPI: { getAvailable: groups } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))
const row = {
  id: 101, request_id: 'req-usage-1', api_key: { id: 7, name: 'enterprise-key' },
  group_id: 23, group: { id: 23, name: 'Enterprise 70%' }, model: 'gpt-6',
  inbound_endpoint: '/v1/responses', ip_address: '203.0.113.10',
  input_tokens: 60, output_tokens: 24, cache_creation_tokens: 0, cache_read_tokens: 0,
  total_cost: 0.00384, actual_cost: 0.00384, billing_type: 0, billing_mode: 'token',
  image_count: 0, duration_ms: 119999, first_token_ms: 460, created_at: '2026-09-21T08:00:00Z',
}
const response = { items: [row], total: 1, pages: 1 }
const stats = { total_requests: 1, total_tokens: 84, total_actual_cost: .00384, endpoints: [] }
const wrappers: ReturnType<typeof mount>[] = []
function mountPage() {
  const wrapper = mount(AppUsageView, { global: { stubs: {
    AppSectionShell: { template: '<main data-testid="ssxz-shell"><slot /></main>' },
    RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
    Icon: true, Select: true, DateRangePicker: true, Pagination: true,
    UsageStatsCards: true, UserErrorRequestsTable: true,
    ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true, TokenUsageTrend: true,
  } } })
  wrappers.push(wrapper)
  return { wrapper, page: wrapper.findComponent(UsageView), summary: wrapper.findComponent(UsageBalanceSummary) }
}
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((next) => { resolve = next })
  return { promise, resolve }
}
beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  api.query.mockResolvedValue(response)
  api.getStats.mockResolvedValue(stats)
  api.getStatsByDateRange.mockResolvedValue(stats)
  api.getDashboardModels.mockResolvedValue({ models: [] })
  api.getDashboardSnapshotV2.mockResolvedValue({ trend: [], groups: [] })
  api.listMyErrorRequests.mockResolvedValue({ items: [], total: 0 })
  auth.refreshUser.mockResolvedValue(auth.user)
  groups.mockResolvedValue([{ id: 23, name: 'Enterprise 70%' }])
  app.cachedPublicSettings.allow_user_view_error_requests = true
})
afterEach(async () => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  await new Promise((resolve) => setTimeout(resolve, 0))
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('actual /app/usage route integration', () => {
  it('routes the customer menu to one upstream-backed workspace with SSXZ shell and new controls', async () => {
    const source = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    expect(source).toContain("path: '/app/usage'")
    expect(source).toContain("import('@/views/user/AppUsageView.vue')")
    const { wrapper } = mountPage()
    await flushPromises()
    expect(wrapper.find('[data-testid="ssxz-shell"]').exists()).toBe(true)
    expect(wrapper.findAllComponents(UsageView)).toHaveLength(1)
    expect(wrapper.find('[data-testid="usage-column-settings"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('usage.tabs.errors')
  })
  it('keeps eight grouped columns with group, endpoint, IP, support code and copy control', async () => {
    const { wrapper } = mountPage()
    await flushPromises()
    const table = wrapper.findComponent(UsageTable)
    expect(table.props('columns').map((c: { key: string }) => c.key)).toEqual(['api_key', 'model', 'endpoint', 'group', 'tokens', 'cost', 'latency', 'created_at'])
    expect(table.props('serverSideSort')).toBe(true)
    expect(table.props('showAccountBilling')).toBe(false)
    expect(table.props('showUpstreamEndpoint')).toBe(false)
    for (const value of ['enterprise-key', 'gpt-6', '/v1/responses', '203.0.113.10', 'Enterprise 70%', 'req-usage-1']) expect(table.text()).toContain(value)
    expect(table.find('[data-testid="grouped-detail-copy"]').exists()).toBe(true)
    expect(table.get('thead .usage-col-group-context').text()).toContain('admin.usage.group')
  })
  it('preserves sub-cent fees and rounds duration at the next minute', async () => {
    const { wrapper } = mountPage()
    await flushPromises()
    const tr = wrapper.get('tbody tr[data-row-id="101"]')
    expect(tr.get('.usage-col-cost').text()).toContain('$0.003840')
    expect(tr.get('.usage-col-latency').text()).toContain('2m 0s')
  })
  it('exposes optional columns and persists settings while time remains mandatory', async () => {
    const { wrapper } = mountPage()
    await flushPromises()
    await wrapper.get('[data-testid="usage-column-settings"]').trigger('click')
    expect(wrapper.find('[data-testid="usage-column-toggle-created_at"]').exists()).toBe(false)
    await wrapper.get('[data-testid="usage-column-toggle-user_agent"]').trigger('click')
    expect(wrapper.findComponent(UsageTable).props('columns').some((c: { key: string }) => c.key === 'user_agent')).toBe(true)
    expect(JSON.parse(localStorage.getItem('user-usage-hidden-columns')!)).not.toContain('user_agent')
  })
  it('keeps account-wide monthly summary independent of selected detail dates', async () => {
    const { wrapper } = mountPage()
    await flushPromises()
    const initial = [...api.getStatsByDateRange.mock.calls]
    wrapper.findComponent(DateRangePicker).vm.$emit('change', { startDate: '2026-08-01', endDate: '2026-08-15', preset: null })
    await flushPromises()
    expect(api.query).toHaveBeenLastCalledWith(expect.objectContaining({ start_date: '2026-08-01', end_date: '2026-08-15' }), expect.anything())
    expect(api.getStatsByDateRange.mock.calls).toEqual(initial)
    expect(api.getStats).toHaveBeenLastCalledWith(expect.objectContaining({ start_date: '2026-08-01' }))
  })
  it('loads records independently and deduplicates overlapping balance refreshes', async () => {
    const balance = deferred<typeof auth.user>()
    auth.refreshUser.mockReturnValue(balance.promise)
    const { wrapper, summary } = mountPage()
    await flushPromises()
    expect(wrapper.find('tbody tr[data-row-id="101"]').exists()).toBe(true)
    await summary.get('button').trigger('click')
    expect(auth.refreshUser).toHaveBeenCalledTimes(1)
    balance.resolve(auth.user)
    await flushPromises()
    auth.refreshUser.mockRejectedValueOnce(new Error('offline'))
    await summary.get('button').trigger('click')
    await flushPromises()
    expect(summary.text()).toContain('usage.workbench.balanceRefreshError')
  })
  it('does not turn failed monthly statistics into a fake zero', async () => {
    api.getStatsByDateRange.mockRejectedValue(new Error('offline'))
    const { wrapper, summary } = mountPage()
    await flushPromises()
    expect(summary.text()).toContain('usage.workbench.unavailable')
    expect(wrapper.find('tbody tr[data-row-id="101"]').exists()).toBe(true)
  })
  it('cancels obsolete requests and preserves newest filter results', async () => {
    const old = deferred<typeof response>()
    const { page } = mountPage()
    await flushPromises()
    api.query.mockReturnValueOnce(old.promise)
    const vm = page.vm as any
    vm.filters.model = 'old-model'
    vm.applyFilters()
    const oldSignal = api.query.mock.calls.at(-1)![1].signal
    vm.filters.model = 'gpt-6'
    vm.applyFilters()
    await flushPromises()
    expect(oldSignal.aborted).toBe(true)
    old.resolve({ ...response, items: [{ ...row, model: 'old-model' }] })
    await flushPromises()
    expect(page.findComponent(UsageTable).props('data')[0].model).toBe('gpt-6')
  })
  it('paginates and sorts on the server, not just the loaded page', async () => {
    const { wrapper, page } = mountPage()
    await flushPromises()
    ;(page.vm as any).handlePageChange(2)
    await flushPromises()
    expect(api.query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }), expect.anything())
    wrapper.findComponent(UsageTable).vm.$emit('sort', 'model', 'asc')
    await flushPromises()
    expect(api.query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, sort_by: 'model', sort_order: 'asc' }), expect.anything())
  })
  it('shows a failure state instead of stale records after an unsuccessful refresh', async () => {
    const { page } = mountPage()
    await flushPromises()
    api.query.mockRejectedValueOnce(new Error('offline'))
    ;(page.vm as any).refreshData()
    await flushPromises()
    expect(page.findComponent(UsageTable).exists()).toBe(false)
    expect(page.find('[role="alert"]').text()).toContain('usage.failedToLoad')
  })
  it('discards stale error responses and respects server-controlled visibility', async () => {
    const { wrapper, page } = mountPage()
    await flushPromises()
    const old = deferred<{ items: unknown[]; total: number }>()
    api.listMyErrorRequests.mockReturnValueOnce(old.promise)
    ;(page.vm as any).switchToErrors()
    ;(page.vm as any).errorFilter.model = 'gpt-6'
    ;(page.vm as any).applyErrorFilters()
    await flushPromises()
    old.resolve({ items: [{ model: 'old' }], total: 1 })
    await flushPromises()
    expect(wrapper.findComponent(UserErrorRequestsTable).props('rows')).toEqual([])
    wrapper.unmount()
    app.cachedPublicSettings.allow_user_view_error_requests = false
    const hidden = mountPage()
    await flushPromises()
    const count = api.listMyErrorRequests.mock.calls.length
    ;(hidden.page.vm as any).switchToErrors()
    expect(api.listMyErrorRequests).toHaveBeenCalledTimes(count)
    expect(hidden.wrapper.text()).not.toContain('usage.tabs.errors')
  })
  it('aborts active requests on unmount', async () => {
    const { wrapper } = mountPage()
    const signal = api.query.mock.calls.at(-1)![1].signal
    wrapper.unmount()
    expect(signal.aborted).toBe(true)
  })
  it('keeps overflow inside the table and delegates mobile cards to DataTable', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/UsageView.vue'), 'utf8')
    expect(source).toContain('min-width: 1320px')
    expect(source).toContain(':sticky-first-column="false"')
    const table = readFileSync(resolve(process.cwd(), 'src/components/admin/usage/UsageTable.vue'), 'utf8')
    expect(table).toContain('class="overflow-auto"')
    expect(table).toContain('<DataTable')
  })
})

describe('customer export safeguards', () => {
  function downloadMocks() {
    const create = vi.fn((_blob: Blob) => 'blob:usage-export')
    const revoke = vi.fn()
    Object.defineProperty(window.URL, 'createObjectURL', { configurable: true, value: create })
    Object.defineProperty(window.URL, 'revokeObjectURL', { configurable: true, value: revoke })
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    return { create, revoke }
  }
  it('deduplicates exports and freezes filters, sort, dates and total pages', async () => {
    downloadMocks()
    api.query.mockResolvedValue({ ...response, total: 101, pages: 2 })
    const { page } = mountPage()
    await flushPromises()
    const first = deferred<typeof response>()
    api.query.mockReturnValueOnce(first.promise)
    const vm = page.vm as any
    const task = vm.exportToCSV()
    const snapshot = { ...api.query.mock.calls.at(-1)![0] }
    const count = api.query.mock.calls.length
    await vm.exportToCSV()
    expect(api.query).toHaveBeenCalledTimes(count)
    vm.filters.model = 'changed-during-export'
    vm.pagination.total = 1
    vm.sortState.sort_by = 'model'
    first.resolve(response)
    await task
    expect(api.query).toHaveBeenLastCalledWith({ ...snapshot, page: 2 })
    expect(app.showSuccess).toHaveBeenCalled()
  })
  it.each(['click', 'remove', 'createElement'] as const)('releases download URLs when %s fails', async (operation) => {
    const { revoke } = downloadMocks()
    const { page } = mountPage()
    await flushPromises()
    if (operation === 'createElement') {
      const realCreate = document.createElement.bind(document)
      vi.spyOn(document, 'createElement').mockImplementation((tag, options) => {
        if (tag === 'a') throw new Error('cannot create anchor')
        return realCreate(tag, options)
      })
    } else {
      vi.spyOn(HTMLAnchorElement.prototype, operation).mockImplementation(() => { throw new Error('download failed') })
    }
    await (page.vm as any).exportToCSV()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(revoke).toHaveBeenCalledWith('blob:usage-export')
    expect(app.showSuccess).not.toHaveBeenCalled()
    expect(app.showError).toHaveBeenCalledWith('usage.exportFailed')
  })
  it('escapes spreadsheet formulas and exports groups and support codes without admin costs', async () => {
    const { create } = downloadMocks()
    api.query.mockResolvedValue({ ...response, items: [{ ...row, model: '=1+1', group: { ...row.group, name: '@group' } }] })
    const { page } = mountPage()
    await flushPromises()
    await (page.vm as any).exportToCSV()
    const csv = await new Promise<string>((resolve) => {
      const reader = new FileReader()
      reader.onload = () => resolve(String(reader.result))
      reader.readAsText(create.mock.calls[0][0])
    })
    expect(csv).toContain("'=1+1")
    expect(csv).toContain("'@group")
    expect(csv).toContain('Support Code')
    expect(csv).toContain('req-usage-1')
    expect(csv).not.toContain('account_cost')
  })
})
