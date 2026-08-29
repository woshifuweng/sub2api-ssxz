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
    name: 'SelectStub',
    props: ['modelValue', 'options', 'disabled'],
    emits: ['update:modelValue'],
    template: '<select :disabled="disabled" />'
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

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((next, fail) => {
    resolve = next
    reject = fail
  })
  return { promise, resolve, reject }
}

function startProgrammaticLoad(wrapper: ReturnType<typeof mountView>, model: string) {
  const setupState = (wrapper.vm as unknown as {
    $: {
      setupState: {
        filters: { model?: string }
        loadUsageOverview: () => Promise<void>
      }
    }
  }).$.setupState
  setupState.filters.model = model
  return setupState.loadUsageOverview()
}

function mockObjectURLs() {
  const createObjectURLDescriptor = Object.getOwnPropertyDescriptor(window.URL, 'createObjectURL')
  const revokeObjectURLDescriptor = Object.getOwnPropertyDescriptor(window.URL, 'revokeObjectURL')
  const createObjectURL = vi.fn(() => 'blob:usage-export')
  const revokeObjectURL = vi.fn()

  Object.defineProperty(window.URL, 'createObjectURL', {
    configurable: true,
    value: createObjectURL
  })
  Object.defineProperty(window.URL, 'revokeObjectURL', {
    configurable: true,
    value: revokeObjectURL
  })

  return {
    createObjectURL,
    revokeObjectURL,
    restore() {
      if (createObjectURLDescriptor) {
        Object.defineProperty(window.URL, 'createObjectURL', createObjectURLDescriptor)
      } else {
        Reflect.deleteProperty(window.URL, 'createObjectURL')
      }
      if (revokeObjectURLDescriptor) {
        Object.defineProperty(window.URL, 'revokeObjectURL', revokeObjectURLDescriptor)
      } else {
        Reflect.deleteProperty(window.URL, 'revokeObjectURL')
      }
    }
  }
}

function usageRow(id: number, model: string) {
  return {
    id,
    request_id: `req-${id}`,
    api_key_id: 7,
    model,
    inbound_endpoint: '/v1/responses',
    input_tokens: 1,
    output_tokens: 0,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_cost: 0.01,
    actual_cost: 0.01,
    billing_type: 0,
    image_count: 0,
    image_size: null,
    duration_ms: 1000,
    first_token_ms: 500,
    created_at: '2026-08-29T08:03:00Z'
  }
}

describe('AppUsageView compact usage details', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    keysAPI.list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    authStore.refreshUser.mockResolvedValue(authStore.user)
    copyToClipboard.mockResolvedValue(true)
    usageAPI.getStatsByDateRange.mockResolvedValue({
      total_requests: 2,
      total_tokens: 94,
      total_actual_cost: 0.43
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
        }
      ],
      total: 2,
      pages: 1
    })
  })

  it('keeps scan-critical fields in the main row and reveals diagnostics on demand', async () => {
    const wrapper = mountView()
    await flushPromises()

    const headers = wrapper.findAll('thead th').map((header) => header.text())
    expect(headers.slice(0, 5)).toEqual(['Created at', 'Model', 'Usage', 'Duration', 'Fee'])
    expect(wrapper.findAll('tr.usage-row')).toHaveLength(2)
    expect(wrapper.findAll('tr.usage-detail-row')).toHaveLength(0)
    expect(wrapper.get('table').text()).not.toContain('/v1/responses')
    expect(wrapper.get('table').text()).not.toContain('req-usage-1')
    expect(wrapper.get('table').text()).not.toContain('Balance charge')

    const rows = wrapper.findAll('tr.usage-row')
    expect(rows[0].findAll('td')[4].text()).toBe('$0.28')
    expect(rows[1].findAll('td')[4].text()).toBe('$0.00')

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

  it('preserves sub-cent fees and rounds durations at the next minute', async () => {
    usageAPI.query.mockResolvedValue({
      items: [
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
      total: 1,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const cells = wrapper.get('tr.usage-row').findAll('td')
    expect.soft(cells[3].text()).toBe('2m')
    expect.soft(cells[4].text()).toBe('$0.00384')
  })

  it('keeps summary statistics on the current month when detail dates change', async () => {
    const wrapper = mountView()
    await flushPromises()
    const initialStatsCall = usageAPI.getStatsByDateRange.mock.calls[0]

    await wrapper.findAll('input[type="date"]')[0].setValue('2026-08-15')
    await flushPromises()

    expect(usageAPI.query).toHaveBeenLastCalledWith(expect.objectContaining({ start_date: '2026-08-15' }))
    expect(usageAPI.getStatsByDateRange).toHaveBeenLastCalledWith(...initialStatsCall)
  })

  it('loads usage independently and deduplicates overlapping balance refreshes', async () => {
    const balanceRefresh = deferred<typeof authStore.user>()
    authStore.refreshUser.mockReturnValue(balanceRefresh.promise)

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.findAll('tr.usage-row')).toHaveLength(2)
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)

    await wrapper.get('.refresh-button').trigger('click')
    await flushPromises()
    expect(usageAPI.query).toHaveBeenCalledTimes(2)
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)

    const modelInput = wrapper.get('input[type="search"]')
    await modelInput.setValue('filtered-model')
    await modelInput.trigger('keyup.enter')
    await flushPromises()
    expect(usageAPI.query).toHaveBeenCalledTimes(3)
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)

    balanceRefresh.resolve(authStore.user)
    await flushPromises()
    authStore.refreshUser.mockRejectedValueOnce(new Error('balance refresh failed'))
    await wrapper.get('.refresh-button').trigger('click')
    await flushPromises()
    expect(authStore.refreshUser).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.usage-summary-card p.is-warning').exists()).toBe(true)

    await modelInput.setValue('second-filter')
    await modelInput.trigger('keyup.enter')
    await flushPromises()
    expect(authStore.refreshUser).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.usage-summary-card p.is-warning').exists()).toBe(true)
  })

  it('keeps loading active when an older load finishes before the newer load', async () => {
    const olderQuery = deferred<{ items: ReturnType<typeof usageRow>[], total: number, pages: number }>()
    const newerQuery = deferred<{ items: ReturnType<typeof usageRow>[], total: number, pages: number }>()
    usageAPI.query
      .mockImplementationOnce(() => olderQuery.promise)
      .mockImplementationOnce(() => newerQuery.promise)

    const wrapper = mountView()
    await flushPromises()

    void startProgrammaticLoad(wrapper, 'newer-model')

    olderQuery.resolve({ items: [usageRow(201, 'older-model')], total: 1, pages: 1 })
    await flushPromises()
    expect(wrapper.get('.refresh-button').attributes('disabled')).toBeDefined()

    newerQuery.resolve({ items: [usageRow(202, 'newer-model')], total: 1, pages: 1 })
    await flushPromises()
    expect(wrapper.get('.refresh-button').attributes('disabled')).toBeUndefined()
    expect(wrapper.get('tr.usage-row .model-cell').text()).toBe('newer-model')
  })

  it('keeps newer expanded details when an older load rejects last', async () => {
    const olderQuery = deferred<{ items: ReturnType<typeof usageRow>[], total: number, pages: number }>()
    const newerQuery = deferred<{ items: ReturnType<typeof usageRow>[], total: number, pages: number }>()
    usageAPI.query
      .mockImplementationOnce(() => olderQuery.promise)
      .mockImplementationOnce(() => newerQuery.promise)

    const wrapper = mountView()
    await flushPromises()

    void startProgrammaticLoad(wrapper, 'newer-model')

    expect(usageAPI.query).toHaveBeenLastCalledWith(expect.objectContaining({ model: 'newer-model' }))

    newerQuery.resolve({ items: [usageRow(202, 'newer-model')], total: 1, pages: 1 })
    await flushPromises()
    expect(wrapper.get('tr.usage-row .model-cell').text()).toBe('newer-model')

    await wrapper.get('tr.usage-row').trigger('click')
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('req-202')

    olderQuery.reject(new Error('stale request failed'))
    await flushPromises()
    expect(wrapper.get('tr.usage-row .model-cell').text()).toBe('newer-model')
    expect(wrapper.get('tr.usage-detail-row').text()).toContain('req-202')
    expect(wrapper.find('.usage-empty.compact').exists()).toBe(false)
  })

  it('disables conflicting controls and freezes export filters and page count', async () => {
    const firstExportPage = deferred<{ items: ReturnType<typeof usageRow>[], total: number, pages: number }>()
    usageAPI.query
      .mockResolvedValueOnce({ items: [usageRow(301, 'visible-model')], total: 101, pages: 6 })
      .mockImplementationOnce(() => firstExportPage.promise)
      .mockResolvedValueOnce({ items: [usageRow(302, 'export-page-2')], total: 101, pages: 2 })

    const wrapper = mountView()
    await flushPromises()

    const modelInput = wrapper.get('input[type="search"]')
    const dateInputs = wrapper.findAll('input[type="date"]')
    await modelInput.setValue('original-model')
    const originalStartDate = (dateInputs[0].element as HTMLInputElement).value
    const originalEndDate = (dateInputs[1].element as HTMLInputElement).value
    const objectURLs = mockObjectURLs()
    const appendChildSpy = vi.spyOn(document.body, 'appendChild')
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    vi.useFakeTimers({ toFake: ['setTimeout'] })

    try {
      const filterButtons = wrapper.findAll('.filter-actions button')
      await filterButtons[1].trigger('click')
      await wrapper.vm.$nextTick()

      const apiKeySelect = wrapper.getComponent({ name: 'SelectStub' })
      const paginationButtons = wrapper.findAll('.usage-pagination button')
      expect.soft(apiKeySelect.attributes('disabled')).toBeDefined()
      expect.soft(modelInput.attributes('disabled')).toBeDefined()
      expect.soft(dateInputs.every((input) => input.attributes('disabled') !== undefined)).toBe(true)
      expect.soft(filterButtons[0].attributes('disabled')).toBeDefined()
      expect.soft(filterButtons[1].attributes('disabled')).toBeDefined()
      expect.soft(wrapper.get('.refresh-button').attributes('disabled')).toBeDefined()
      expect.soft(paginationButtons).toHaveLength(2)
      expect.soft(paginationButtons.every((button) => button.attributes('disabled') !== undefined)).toBe(true)

      await modelInput.setValue('changed-model')
      await modelInput.trigger('keyup.enter')
      await dateInputs[0].setValue('2026-01-01')
      await dateInputs[1].setValue('2026-01-31')
      apiKeySelect.vm.$emit('update:modelValue', 7)
      await filterButtons[0].trigger('click')
      await wrapper.get('.refresh-button').trigger('click')
      await paginationButtons[1].trigger('click')
      await wrapper.vm.$nextTick()

      expect.soft(usageAPI.query).toHaveBeenCalledTimes(2)
      expect.soft(authStore.refreshUser).toHaveBeenCalledTimes(1)

      firstExportPage.resolve({ items: [usageRow(303, 'export-page-1')], total: 101, pages: 2 })
      await flushPromises()

      expect(usageAPI.query.mock.calls.slice(1).map(([query]) => query)).toEqual([
        {
          page: 1,
          page_size: 100,
          api_key_id: undefined,
          model: 'original-model',
          start_date: originalStartDate,
          end_date: originalEndDate
        },
        {
          page: 2,
          page_size: 100,
          api_key_id: undefined,
          model: 'original-model',
          start_date: originalStartDate,
          end_date: originalEndDate
        }
      ])
      const anchor = appendChildSpy.mock.calls
        .map(([node]) => node)
        .find((node): node is HTMLAnchorElement => node instanceof HTMLAnchorElement)
      expect.soft(anchor?.download).toBe(`usage_${originalStartDate}_to_${originalEndDate}.csv`)
      expect.soft(document.body.contains(anchor || null)).toBe(false)
      expect.soft(clickSpy).toHaveBeenCalledTimes(1)
      expect.soft(appStore.showSuccess).toHaveBeenCalledTimes(1)
      await vi.runOnlyPendingTimersAsync()
      expect(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:usage-export')
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
      clickSpy.mockRestore()
      appendChildSpy.mockRestore()
      objectURLs.restore()
    }
  })

  it('appends and removes the download anchor and still schedules cleanup when click throws', async () => {
    const wrapper = mountView()
    await flushPromises()

    const objectURLs = mockObjectURLs()
    const appendChildSpy = vi.spyOn(document.body, 'appendChild')
    const removeSpy = vi.spyOn(Element.prototype, 'remove')
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {
      throw new Error('download click failed')
    })
    vi.useFakeTimers({ toFake: ['setTimeout'] })

    try {
      await wrapper.findAll('.filter-actions button')[1].trigger('click')
      await flushPromises()

      const anchor = appendChildSpy.mock.calls
        .map(([node]) => node)
        .find((node): node is HTMLAnchorElement => node instanceof HTMLAnchorElement)
      expect(anchor).toBeDefined()
      expect.soft(anchor?.href).toBe('blob:usage-export')
      expect.soft(anchor?.download).toMatch(/^usage_\d{4}-\d{2}-\d{2}_to_\d{4}-\d{2}-\d{2}\.csv$/)
      expect.soft(clickSpy).toHaveBeenCalledTimes(1)
      expect.soft(removeSpy.mock.instances).toContain(anchor)
      expect.soft(document.body.contains(anchor || null)).toBe(false)
      expect.soft(objectURLs.revokeObjectURL).not.toHaveBeenCalled()
      expect.soft(appStore.showError).toHaveBeenCalledTimes(1)

      await vi.runOnlyPendingTimersAsync()

      expect(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:usage-export')
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
      clickSpy.mockRestore()
      removeSpy.mockRestore()
      appendChildSpy.mockRestore()
      objectURLs.restore()
    }
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
