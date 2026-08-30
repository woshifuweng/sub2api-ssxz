import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { usageAPI, keysAPI, authStore, appStore } = vi.hoisted(() => ({
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
  }
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
  'usage.workbench.group': 'Group',
  'usage.workbench.noGroup': 'No group recorded',
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

function mockObjectURLs() {
  const createObjectURLDescriptor = Object.getOwnPropertyDescriptor(window.URL, 'createObjectURL')
  const revokeObjectURLDescriptor = Object.getOwnPropertyDescriptor(window.URL, 'revokeObjectURL')
  const createObjectURL = vi.fn((blob: Blob) => {
    void blob
    return 'blob:usage-export'
  })
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

function waitForZeroDelayCleanup() {
  return new Promise<void>((resolve) => window.setTimeout(resolve, 0))
}

function readBlobText(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.addEventListener('load', () => resolve(String(reader.result)))
    reader.addEventListener('error', () => reject(reader.error))
    reader.readAsText(blob)
  })
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
          api_key: { id: 7, name: 'enterprise-key' },
          group_id: 23,
          group: { id: 23, name: 'Enterprise 70%' },
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

  it('uses eight grouped columns without hiding any usage details', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/AppUsageView.vue'), 'utf8')
    const columnsSource = source.match(/const usageTableColumns = computed<Column\[\]>\(\(\) => \[([\s\S]*?)\n\]\)/)?.[1]

    expect(source).toContain("import UsageTable from '@/components/admin/usage/UsageTable.vue'")
    expect(source).toContain('<UsageTable')
    expect(source).toContain(':grouped-details="true"')
    expect(columnsSource).toBeDefined()
    expect([...columnsSource!.matchAll(/key:\s*'([^']+)'/g)].map((match) => match[1])).toEqual([
      'api_key',
      'model',
      'endpoint',
      'group',
      'tokens',
      'cost',
      'latency',
      'created_at'
    ])

    for (const key of ['reasoning_effort', 'ip_address', 'stream', 'billing_mode', 'request_id']) {
      expect(columnsSource).not.toContain(`key: '${key}'`)
    }
  })

  it('keeps complete scan-critical fields in the main desktop row', async () => {
    const wrapper = mountView()
    await flushPromises()

    const headers = wrapper.findAll('thead th').map((header) => header.text())
    expect(headers).toHaveLength(8)
    expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(2)

    const tableText = wrapper.get('[data-testid="usage-native-table"]').text()
    expect(tableText).toContain('enterprise-key')
    expect(tableText).toContain('/v1/responses')
    expect(tableText).toContain('Enterprise 70%')
    expect(tableText).toContain('req-usage-1')
    expect(tableText).toContain('$0.280000')
    expect(tableText).toContain('60')
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

    const row = wrapper.get('tbody tr[data-row-id="103"]')
    expect.soft(row.get('.usage-col-latency').text()).toContain('2m 0s')
    expect.soft(row.get('.usage-col-cost').text()).toContain('$0.003840')
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

    expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(2)
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

  it('blocks filter, reset, refresh and page actions while a filter load is pending', async () => {
    const pendingQuery = deferred<{ items: ReturnType<typeof usageRow>[], total: number, pages: number }>()
    usageAPI.query
      .mockResolvedValueOnce({ items: [usageRow(201, 'initial-model')], total: 41, pages: 3 })
      .mockImplementationOnce(() => pendingQuery.promise)

    const wrapper = mountView()
    await flushPromises()

    const modelInput = wrapper.get('input[type="search"]')
    const dateInputs = wrapper.findAll('input[type="date"]')
    const apiKeySelect = wrapper.getComponent({ name: 'SelectStub' })
    const filterButtons = wrapper.findAll('.filter-actions button')
    const paginationButtons = wrapper.findAll('.usage-pagination button')
    await modelInput.setValue('held-model')
    await modelInput.trigger('keyup.enter')
    await wrapper.vm.$nextTick()

    expect.soft(usageAPI.query).toHaveBeenLastCalledWith(expect.objectContaining({ model: 'held-model' }))
    expect.soft(modelInput.attributes('disabled')).toBeDefined()
    expect.soft(dateInputs.every((input) => input.attributes('disabled') !== undefined)).toBe(true)
    expect.soft(apiKeySelect.attributes('disabled')).toBeDefined()
    expect.soft(filterButtons[0].attributes('disabled')).toBeDefined()
    expect.soft(wrapper.get('.refresh-button').attributes('disabled')).toBeDefined()
    expect.soft(wrapper.find('.usage-pagination').exists()).toBe(false)

    await modelInput.setValue('blocked-model')
    await modelInput.trigger('keyup.enter')
    await dateInputs[0].setValue('2026-01-01')
    apiKeySelect.vm.$emit('update:modelValue', 7)
    await filterButtons[0].trigger('click')
    await wrapper.get('.refresh-button').trigger('click')
    await paginationButtons[1].trigger('click')
    await wrapper.vm.$nextTick()

    expect.soft(usageAPI.query).toHaveBeenCalledTimes(2)
    expect.soft(authStore.refreshUser).toHaveBeenCalledTimes(1)

    pendingQuery.resolve({ items: [usageRow(202, 'held-model-result')], total: 41, pages: 3 })
    await flushPromises()

    expect.soft(wrapper.get('.refresh-button').attributes('disabled')).toBeUndefined()
    expect.soft(wrapper.get('tbody tr[data-row-id] .usage-col-model-context').text()).toContain('held-model-result')
    expect(wrapper.get('.usage-pagination').text()).toContain('1 / 3')
  })

  it('blocks repeated page, filter and refresh actions while pagination is pending', async () => {
    const pendingPage = deferred<{ items: ReturnType<typeof usageRow>[], total: number, pages: number }>()
    usageAPI.query
      .mockResolvedValueOnce({ items: [usageRow(203, 'page-one')], total: 41, pages: 3 })
      .mockImplementationOnce(() => pendingPage.promise)

    const wrapper = mountView()
    await flushPromises()

    const paginationButtons = wrapper.findAll('.usage-pagination button')
    await paginationButtons[1].trigger('click')
    await wrapper.vm.$nextTick()

    expect.soft(usageAPI.query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }))
    expect.soft(wrapper.get('.refresh-button').attributes('disabled')).toBeDefined()
    expect.soft(wrapper.find('.usage-pagination').exists()).toBe(false)

    const modelInput = wrapper.get('input[type="search"]')
    await modelInput.setValue('blocked-model')
    await modelInput.trigger('keyup.enter')
    await wrapper.get('.refresh-button').trigger('click')
    await paginationButtons[0].trigger('click')
    await paginationButtons[1].trigger('click')
    await wrapper.vm.$nextTick()

    expect.soft(usageAPI.query).toHaveBeenCalledTimes(2)
    expect.soft(authStore.refreshUser).toHaveBeenCalledTimes(1)

    pendingPage.resolve({ items: [usageRow(204, 'page-two-result')], total: 41, pages: 3 })
    await flushPromises()

    expect.soft(wrapper.get('tbody tr[data-row-id] .usage-col-model-context').text()).toContain('page-two-result')
    expect(wrapper.get('.usage-pagination').text()).toContain('2 / 3')
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
      await waitForZeroDelayCleanup()
      expect(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:usage-export')
    } finally {
      wrapper.unmount()
      clickSpy.mockRestore()
      appendChildSpy.mockRestore()
      objectURLs.restore()
    }
  })

  it('still schedules object URL cleanup when clicking and removing the anchor throw', async () => {
    const wrapper = mountView()
    await flushPromises()

    const objectURLs = mockObjectURLs()
    const appendChildSpy = vi.spyOn(document.body, 'appendChild')
    const removeSpy = vi.spyOn(Element.prototype, 'remove').mockImplementation(() => {
      throw new Error('anchor removal failed')
    })
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {
      throw new Error('download click failed')
    })
    let anchor: HTMLAnchorElement | undefined

    try {
      await wrapper.findAll('.filter-actions button')[1].trigger('click')
      await flushPromises()

      anchor = appendChildSpy.mock.calls
        .map(([node]) => node)
        .find((node): node is HTMLAnchorElement => node instanceof HTMLAnchorElement)
      expect(anchor).toBeDefined()
      expect.soft(anchor?.href).toBe('blob:usage-export')
      expect.soft(anchor?.download).toMatch(/^usage_\d{4}-\d{2}-\d{2}_to_\d{4}-\d{2}-\d{2}\.csv$/)
      expect.soft(clickSpy).toHaveBeenCalledTimes(1)
      expect.soft(removeSpy.mock.instances).toContain(anchor)
      expect.soft(appStore.showError).toHaveBeenCalledTimes(1)

      await waitForZeroDelayCleanup()

      expect(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:usage-export')
    } finally {
      clickSpy.mockRestore()
      removeSpy.mockRestore()
      appendChildSpy.mockRestore()
      anchor?.remove()
      wrapper.unmount()
      objectURLs.restore()
    }
  })

  it('still schedules object URL cleanup when creating the download anchor throws', async () => {
    const wrapper = mountView()
    await flushPromises()

    const objectURLs = mockObjectURLs()
    const createElementSpy = vi.spyOn(document, 'createElement').mockImplementation(() => {
      throw new Error('anchor creation failed')
    })

    try {
      await wrapper.findAll('.filter-actions button')[1].trigger('click')
      await flushPromises()
      await waitForZeroDelayCleanup()

      expect.soft(objectURLs.createObjectURL).toHaveBeenCalledTimes(1)
      expect.soft(createElementSpy).toHaveBeenCalledWith('a')
      expect.soft(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:usage-export')
      expect(appStore.showError).toHaveBeenCalledTimes(1)
    } finally {
      createElementSpy.mockRestore()
      wrapper.unmount()
      objectURLs.restore()
    }
  })

  it('keeps formula-like values escaped in the generated CSV Blob', async () => {
    usageAPI.query
      .mockResolvedValueOnce({ items: [usageRow(305, 'visible-model')], total: 1, pages: 1 })
      .mockResolvedValueOnce({ items: [usageRow(306, '=2+2')], total: 1, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    const objectURLs = mockObjectURLs()
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)

    try {
      await wrapper.findAll('.filter-actions button')[1].trigger('click')
      await flushPromises()

      const [blob] = objectURLs.createObjectURL.mock.calls[0]
      const csv = await readBlobText(blob)
      expect(csv).toContain(",'=2+2,")

      await waitForZeroDelayCleanup()
      expect(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:usage-export')
    } finally {
      clickSpy.mockRestore()
      wrapper.unmount()
      objectURLs.restore()
    }
  })

  it('keeps the support code visible with a copy control in the grouped activity column', async () => {
    const wrapper = mountView()
    await flushPromises()

    const supportCell = wrapper.get('tbody tr[data-row-id="101"] .usage-col-activity [data-testid="grouped-detail-created-at"]')
    expect(supportCell.text()).toContain('req-usage-1')
    expect(supportCell.find('button').exists()).toBe(true)
  })

  it('keeps wide data inside the table and delegates mobile rendering to the native DataTable cards', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/AppUsageView.vue'), 'utf8')
    const nativeTableSource = readFileSync(resolve(process.cwd(), 'src/components/admin/usage/UsageTable.vue'), 'utf8')
    const dataTableSource = readFileSync(resolve(process.cwd(), 'src/components/common/DataTable.vue'), 'utf8')

    expect(source).toMatch(/\.usage-native-table\s+:deep\(\.table-wrapper\)[\s\S]*overflow-x:\s*auto/)
    expect(source).toMatch(/\.usage-native-table\s+:deep\(table\)[\s\S]*min-width:\s*86\.5rem/)
    expect(source).toContain('.usage-native-table :deep(.usage-col-api-key) { width: 10rem; }')
    expect(source).toContain('.usage-native-table :deep(.usage-col-model-context) { width: 8.5rem; }')
    expect(source).toContain('.usage-native-table :deep(.usage-col-route) { width: 16rem; }')
    expect(source).toContain('.usage-native-table :deep(.usage-col-group-context) { width: 14rem; }')
    expect(source).toContain('.usage-native-table :deep(.usage-col-tokens) { width: 10.5rem; }')
    expect(source).toContain('.usage-native-table :deep(.usage-col-cost) { width: 7.5rem; }')
    expect(source).toContain('.usage-native-table :deep(.usage-col-latency) { width: 8.5rem; }')
    expect(source).toContain('.usage-native-table :deep(.usage-col-activity) { width: 11.5rem; }')
    expect(source).toMatch(/\.usage-native-table\s+:deep\(td\)[\s\S]*overflow:\s*hidden/)
    expect(nativeTableSource).toContain('<DataTable')
    expect(nativeTableSource).toContain('block max-w-full truncate')
    expect(nativeTableSource).toContain(':sticky-first-column="stickyFirstColumn"')
    expect(source).toContain(':sticky-first-column="false"')
    expect(source).toContain(':sticky-actions-column="false"')
    expect(dataTableSource).toContain('v-if="!isDesktopViewport"')
    expect(dataTableSource).toContain(':data-field="column.key"')
  })
})
