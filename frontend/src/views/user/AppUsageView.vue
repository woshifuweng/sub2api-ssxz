<template>
  <AppSectionShell
    :title="t('usage.workbench.title')"
    :subtitle="t('usage.workbench.subtitle')"
    :eyebrow="t('usage.workbench.eyebrow')"
    icon="chartBar"
  >
    <section class="usage-workbench" :aria-label="t('usage.workbench.title')">
      <div class="usage-summary-grid">
        <article class="usage-summary-card">
          <div class="summary-icon">
            <Icon name="creditCard" size="sm" />
          </div>
          <div>
            <span>{{ t('usage.workbench.balanceTitle') }}</span>
            <strong :title="balanceTitle">{{ balanceText }}</strong>
            <p :class="{ 'is-warning': balanceRefreshError }">{{ balanceDescriptionText }}</p>
          </div>
          <RouterLink to="/app/purchase" class="btn btn-primary summary-action">
            {{ t('usage.workbench.recharge') }}
          </RouterLink>
        </article>

        <article class="usage-summary-card">
          <div class="summary-icon">
            <Icon name="chartBar" size="sm" />
          </div>
          <div>
            <span>{{ t('usage.workbench.monthlyCostTitle') }}</span>
            <strong :title="monthlyCostTitle">{{ monthlyCostText }}</strong>
            <p>{{ monthlyUsageNote }}</p>
          </div>
        </article>
      </div>

      <section class="usage-explainer" :aria-label="t('usage.workbench.billingExplanationTitle')">
        <div>
          <strong>{{ t('usage.workbench.billingExplanationTitle') }}</strong>
          <p>{{ t('usage.workbench.billingExplanationDescription') }}</p>
        </div>
        <ul>
          <li v-for="item in billingExplanationItems" :key="item">{{ item }}</li>
        </ul>
      </section>

      <section class="usage-trends-section">
        <header class="panel-heading">
          <div>
            <h3>{{ t('usage.workbench.monthlyUsageTitle') }}</h3>
            <p>{{ t('usage.workbench.monthlyUsageDescription') }}</p>
          </div>
          <span v-if="hasDailyUsage" class="panel-badge">{{ usageDataBadge }}</span>
        </header>

        <div v-if="trendLoadError" class="usage-empty">
          <div class="usage-empty__icon is-warning"><Icon name="exclamationTriangle" size="lg" /></div>
          <strong>{{ t('usage.workbench.trendLoadError') }}</strong>
          <span>{{ t('usage.workbench.trendLoadErrorHint') }}</span>
        </div>

        <div v-else-if="hasDailyUsage" class="usage-trend-grid" :aria-label="t('usage.workbench.monthlyChartLabel')">
          <ProgressMetricCard
            test-id="usage-trend-cost"
            :label="t('usage.workbench.dailyCostTrend')"
            :icon="CircleDollarSign"
            detailed
            :series="dailyCostSeries"
            :series-labels="dailyLabels"
            :series-formatter="formatTrendCost"
            :period-options="metricPeriodOptions"
            default-period="30d"
            :period-select-label="t('usage.workbench.periodSelectLabel')"
            :view-toggle-label="t('usage.workbench.viewToggleLabel')"
            :curve-view-label="t('usage.workbench.curveView')"
            :bar-view-label="t('usage.workbench.barView')"
            :low-label="t('usage.workbench.low')"
            :average-label="t('usage.workbench.average')"
            :peak-label="t('usage.workbench.peak')"
            :trend-label="t('usage.workbench.dailyCostTrend')"
          >
            <span>{{ t('usage.workbench.actualChargeTrendHint') }}</span>
          </ProgressMetricCard>

          <ProgressMetricCard
            test-id="usage-trend-requests"
            :label="t('usage.workbench.dailyRequestTrend')"
            :icon="Activity"
            detailed
            :series="dailyRequestSeries"
            :series-labels="dailyLabels"
            :series-formatter="formatTrendNumber"
            :period-options="metricPeriodOptions"
            default-period="30d"
            :period-select-label="t('usage.workbench.periodSelectLabel')"
            :view-toggle-label="t('usage.workbench.viewToggleLabel')"
            :curve-view-label="t('usage.workbench.curveView')"
            :bar-view-label="t('usage.workbench.barView')"
            :low-label="t('usage.workbench.low')"
            :average-label="t('usage.workbench.average')"
            :peak-label="t('usage.workbench.peak')"
            :trend-label="t('usage.workbench.dailyRequestTrend')"
          >
            <span>{{ t('usage.workbench.requestTrendHint') }}</span>
          </ProgressMetricCard>

          <ProgressMetricCard
            test-id="usage-trend-tokens"
            :label="t('usage.workbench.dailyTokenTrend')"
            :icon="DatabaseZap"
            detailed
            :series="dailyTokenSeries"
            :series-labels="dailyLabels"
            :series-formatter="formatTrendNumber"
            :period-options="metricPeriodOptions"
            default-period="30d"
            :period-select-label="t('usage.workbench.periodSelectLabel')"
            :view-toggle-label="t('usage.workbench.viewToggleLabel')"
            :curve-view-label="t('usage.workbench.curveView')"
            :bar-view-label="t('usage.workbench.barView')"
            :low-label="t('usage.workbench.low')"
            :average-label="t('usage.workbench.average')"
            :peak-label="t('usage.workbench.peak')"
            :trend-label="t('usage.workbench.dailyTokenTrend')"
          >
            <span>{{ t('usage.workbench.tokenTrendHint') }}</span>
          </ProgressMetricCard>
        </div>

        <div v-else class="usage-empty">
          <div class="usage-empty__icon"><Icon name="chartBar" size="lg" /></div>
          <strong>{{ t('usage.workbench.noMonthlyUsageTitle') }}</strong>
          <span>{{ t('usage.workbench.noMonthlyUsageDescription') }}</span>
          <RouterLink to="/app/keys" class="btn btn-primary btn-sm">
            {{ t('usage.workbench.manageKeys', '管理 API Key') }}
          </RouterLink>
        </div>
      </section>

      <section class="usage-panel">
        <header class="panel-heading">
          <div>
            <h3>{{ t('usage.workbench.usageDetailsTitle') }}</h3>
            <p>{{ t('usage.workbench.usageDetailsDescription') }}</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm refresh-button" :disabled="loading" @click="loadUsageOverview">
            <Icon name="refresh" size="xs" />
            {{ t('usage.workbench.refresh') }}
          </button>
        </header>

        <div class="usage-filters" data-testid="usage-filters">
          <label class="filter-field">
            <span>{{ t('usage.apiKeyFilter') }}</span>
            <Select
              data-testid="usage-api-key-filter"
              :model-value="filters.api_key_id"
              :options="apiKeyOptions"
              @update:model-value="updateApiKeyFilter"
            />
          </label>
          <label class="filter-field">
            <span>{{ t('usage.model') }}</span>
            <input
              v-model.trim="filters.model"
              class="f0-input-control"
              type="search"
              :placeholder="t('usage.workbench.modelFilterPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </label>
          <label class="filter-field">
            <span>{{ t('usage.workbench.startDate') }}</span>
            <input v-model="filters.start_date" class="f0-input-control" type="date" @change="applyFilters" />
          </label>
          <label class="filter-field">
            <span>{{ t('usage.workbench.endDate') }}</span>
            <input v-model="filters.end_date" class="f0-input-control" type="date" @change="applyFilters" />
          </label>
          <div class="filter-actions">
            <button type="button" class="btn btn-secondary" @click="resetFilters">
              {{ t('common.reset') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="exporting || totalRows === 0" @click="exportToCSV">
              <Icon name="download" size="xs" />
              {{ exporting ? t('usage.exporting') : t('usage.exportCsv') }}
            </button>
          </div>
        </div>

        <div v-if="loading" class="usage-empty compact">
          <div class="usage-empty__icon"><Icon name="sync" size="md" /></div>
          <strong>{{ t('usage.workbench.loading') }}</strong>
        </div>

        <div v-else-if="detailsLoadError" class="usage-empty compact">
          <div class="usage-empty__icon is-warning"><Icon name="exclamationTriangle" size="md" /></div>
          <strong>{{ t('usage.workbench.detailsLoadError') }}</strong>
          <span>{{ t('usage.workbench.detailsLoadErrorHint') }}</span>
        </div>

        <div v-else-if="usageRows.length === 0" class="usage-empty compact">
          <div class="usage-empty__icon"><Icon name="inbox" size="md" /></div>
          <strong>{{ t('usage.workbench.noDetailsTitle') }}</strong>
          <span>{{ t('usage.workbench.noDetailsDescription') }}</span>
          <RouterLink to="/app/keys" class="btn btn-primary btn-sm">
            {{ t('usage.workbench.manageKeys', '管理 API Key') }}
          </RouterLink>
        </div>

        <div v-else class="usage-table-wrap">
          <table class="usage-table f0-table">
            <thead>
              <tr>
                <th>{{ t('usage.workbench.createdAt') }}</th>
                <th>{{ t('usage.workbench.kind') }}</th>
                <th>{{ t('usage.workbench.endpoint') }}</th>
                <th>{{ t('usage.workbench.model') }}</th>
                <th>{{ t('usage.workbench.source') }}</th>
                <th>{{ t('usage.workbench.amount') }}</th>
                <th>{{ t('usage.workbench.billingBasis') }}</th>
                <th>{{ t('usage.workbench.performance') }}</th>
                <th>{{ t('usage.workbench.supportCode') }}</th>
                <th>{{ t('usage.workbench.fee') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in usageRows" :key="row.id || row.request_id">
                <td>{{ formatDateTime(row.created_at) }}</td>
                <td>{{ formatUsageKind(row) }}</td>
                <td><code>{{ resolveEndpoint(row) }}</code></td>
                <td class="model-cell">{{ row.model || '-' }}</td>
                <td>{{ formatSource(row) }}</td>
                <td>{{ formatUsageAmount(row) }}</td>
                <td class="billing-cell">
                  <span>{{ formatBillingType(row) }}</span>
                  <small>{{ formatBillingBasis(row) }}</small>
                </td>
                <td class="performance-cell">
                  <span class="performance-badge" :class="performanceBadgeClass(row)">
                    {{ formatPerformanceStatus(row) }}
                  </span>
                  <small>{{ formatPerformanceSummary(row) }}</small>
                  <small v-if="formatPerformanceHint(row)" class="performance-hint">
                    {{ formatPerformanceHint(row) }}
                  </small>
                </td>
                <td class="support-cell">
                  <template v-if="resolveSupportCode(row)">
                    <code>{{ resolveSupportCode(row) }}</code>
                    <button
                      type="button"
                      class="support-code-button"
                      :aria-label="t('usage.workbench.copySupportCode')"
                      :title="t('usage.workbench.copySupportCode')"
                      @click="copySupportCode(row)"
                    >
                      {{ copiedSupportCode === resolveSupportCode(row) ? t('usage.workbench.copied') : t('usage.workbench.copy') }}
                    </button>
                  </template>
                  <span v-else>-</span>
                </td>
                <td>
                  <span :title="formatCostTitle(row.actual_cost)">{{ formatCost(row.actual_cost) }}</span>
                  <small v-if="isNoCharge(row)" class="usage-cost-note">
                    {{ t('usage.workbench.noCharge') }}
                  </small>
                  <small v-else-if="isZeroTokenCharged(row)" class="usage-cost-note">
                    {{ t('usage.workbench.zeroTokenCharged') }}
                  </small>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="!loading && totalRows > 0" class="usage-pagination">
          <span>{{ t('usage.workbench.paginationSummary', { total: totalRows }) }}</span>
          <div>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="changePage(page - 1)">
              {{ t('pagination.previous') }}
            </button>
            <strong>{{ page }} / {{ totalPages }}</strong>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="page >= totalPages" @click="changePage(page + 1)">
              {{ t('pagination.next') }}
            </button>
          </div>
        </div>
      </section>
    </section>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Activity, CircleDollarSign, DatabaseZap } from '@lucide/vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import ProgressMetricCard from '@/components/user/dashboard/ProgressMetricCard.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import { keysAPI, usageAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import type { ApiKey, TrendDataPoint, UsageLog, UsageQueryParams, UsageStatsResponse } from '@/types'
import {
  formatCurrency as formatMoney,
  formatCurrencyTitle as formatMoneyTitle
} from '@/utils/format'

interface DailyUsage {
  key: string
  label: string
  requests: number
  tokens: number
  cost: number
}

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const usageRows = ref<UsageLog[]>([])
const usageStats = ref<UsageStatsResponse | null>(null)
const dailySeries = ref<DailyUsage[]>([])
const loading = ref(false)
const detailsLoadError = ref(false)
const statsLoadError = ref(false)
const trendLoadError = ref(false)
const balanceRefreshError = ref(false)
const copiedSupportCode = ref<string | null>(null)
const apiKeys = ref<ApiKey[]>([])
const exporting = ref(false)
const page = ref(1)
const pageSize = 8
const totalRows = ref(0)
const totalPages = ref(1)

const today = new Date()
const monthStart = new Date(today.getFullYear(), today.getMonth(), 1)
const trendStart = new Date(today.getFullYear(), today.getMonth() - 5, 1)

const todayKey = toDateKey(today)
const monthStartKey = toDateKey(monthStart)
const trendStartKey = toDateKey(trendStart)
const filters = ref<UsageQueryParams>({
  api_key_id: undefined,
  model: '',
  start_date: monthStartKey,
  end_date: todayKey
})

const balanceText = computed(() => formatMoney(authStore.user?.balance || 0))
const balanceTitle = computed(() => formatMoneyTitle(authStore.user?.balance || 0))
const balanceDescriptionText = computed(() => {
  if (balanceRefreshError.value) return t('usage.workbench.balanceRefreshError')
  return t('usage.workbench.balanceDescription')
})
const billingExplanationItems = computed(() => [
  t('usage.workbench.billingExplanationItems.successCharged'),
  t('usage.workbench.billingExplanationItems.failureNoCharge'),
  t('usage.workbench.billingExplanationItems.zeroCost')
])
const monthlyCostText = computed(() => {
  if (statsLoadError.value) return t('usage.workbench.unavailable')
  return formatMoney(usageStats.value?.total_actual_cost || 0)
})
const monthlyCostTitle = computed(() => (
  statsLoadError.value ? undefined : formatMoneyTitle(usageStats.value?.total_actual_cost || 0)
))
const monthlyUsageNote = computed(() => {
  if (statsLoadError.value) return t('usage.workbench.statsLoadError')

  const requests = usageStats.value?.total_requests || 0
  const tokens = usageStats.value?.total_tokens || 0
  if (!requests && !tokens) return t('usage.workbench.noRealUsageNote')
  return t('usage.workbench.monthlyUsageSummary', {
    requests: formatNumber(requests),
    tokens: formatNumber(tokens)
  })
})
const hasDailyUsage = computed(() => dailySeries.value.some((item) => item.requests > 0 || item.tokens > 0 || item.cost > 0))
const isDemoUser = computed(() => {
  const username = authStore.user?.username?.toLowerCase?.() || ''
  const email = authStore.user?.email?.toLowerCase?.() || ''
  return username.includes('demo') || email.endsWith('@example.local')
})
const usageDataBadge = computed(() =>
  isDemoUser.value ? t('usage.workbench.demoDataBadge') : t('usage.workbench.realDataBadge')
)
const apiKeyOptions = computed(() => [
  { value: null, label: t('usage.allApiKeys') },
  ...apiKeys.value.map((key) => ({ value: key.id, label: key.name }))
])
const dailyLabels = computed(() => dailySeries.value.map((item) => item.label))
const dailyCostSeries = computed(() => dailySeries.value.map((item) => item.cost))
const dailyRequestSeries = computed(() => dailySeries.value.map((item) => item.requests))
const dailyTokenSeries = computed(() => dailySeries.value.map((item) => item.tokens))
const metricPeriodOptions = computed(() => [
  { value: '7d', label: t('usage.workbench.period7Days'), points: 7 },
  { value: '30d', label: t('usage.workbench.period30Days'), points: 30 },
  { value: '90d', label: t('usage.workbench.period90Days'), points: 90 },
  { value: 'all', label: t('usage.workbench.periodAll') }
])

onMounted(() => {
  void loadApiKeys()
  void loadUsageOverview()
})

async function loadUsageOverview() {
  loading.value = true
  detailsLoadError.value = false
  statsLoadError.value = false
  trendLoadError.value = false
  balanceRefreshError.value = false

  const [statsResult, logsResult, trendResult, userResult] = await Promise.allSettled([
    usageAPI.getStatsByDateRange(
      monthStartKey,
      todayKey,
      filters.value.api_key_id ? Number(filters.value.api_key_id) : undefined
    ),
    usageAPI.query({
      page: page.value,
      page_size: pageSize,
      api_key_id: filters.value.api_key_id ? Number(filters.value.api_key_id) : undefined,
      model: filters.value.model?.trim() || undefined,
      start_date: filters.value.start_date || monthStartKey,
      end_date: filters.value.end_date || todayKey
    }),
    usageAPI.getDashboardTrend({
      start_date: trendStartKey,
      end_date: todayKey,
      granularity: 'day'
    }),
    authStore.refreshUser()
  ])

  if (statsResult.status === 'fulfilled') {
    usageStats.value = statsResult.value
  } else {
    usageStats.value = null
    statsLoadError.value = true
  }

  if (logsResult.status === 'fulfilled') {
    usageRows.value = Array.isArray(logsResult.value.items) ? logsResult.value.items : []
    totalRows.value = Number(logsResult.value.total || 0)
    totalPages.value = Math.max(1, Number(logsResult.value.pages || 1))
  } else {
    usageRows.value = []
    totalRows.value = 0
    totalPages.value = 1
    detailsLoadError.value = true
  }

  if (trendResult.status === 'fulfilled') {
    dailySeries.value = buildDailySeries(trendResult.value.trend || [])
  } else {
    dailySeries.value = []
    trendLoadError.value = true
  }

  if (userResult?.status === 'rejected') {
    balanceRefreshError.value = true
  }

  loading.value = false
}

async function loadApiKeys() {
  try {
    const response = await keysAPI.list(1, 100)
    apiKeys.value = Array.isArray(response.items) ? response.items : []
  } catch {
    apiKeys.value = []
  }
}

function applyFilters() {
  page.value = 1
  void loadUsageOverview()
}

function updateApiKeyFilter(value: string | number | boolean | null) {
  filters.value.api_key_id = typeof value === 'number' ? value : undefined
  applyFilters()
}

function resetFilters() {
  filters.value = {
    api_key_id: undefined,
    model: '',
    start_date: monthStartKey,
    end_date: todayKey
  }
  applyFilters()
}

function changePage(nextPage: number) {
  page.value = Math.min(Math.max(1, nextPage), totalPages.value)
  void loadUsageOverview()
}

function escapeCSVValue(value: unknown) {
  if (value == null) return ''
  const raw = String(value)
  const safe = /^[=+\-@\t\r]/.test(raw) ? `'${raw}` : raw
  return /[,"\n\r]/.test(safe) ? `"${safe.replace(/"/g, '""')}"` : safe
}

async function exportToCSV() {
  if (totalRows.value === 0) return
  exporting.value = true
  try {
    const rows: UsageLog[] = []
    const exportPageSize = 100
    const pages = Math.ceil(totalRows.value / exportPageSize)
    for (let exportPage = 1; exportPage <= pages; exportPage += 1) {
      const response = await usageAPI.query({
        page: exportPage,
        page_size: exportPageSize,
        api_key_id: filters.value.api_key_id ? Number(filters.value.api_key_id) : undefined,
        model: filters.value.model?.trim() || undefined,
        start_date: filters.value.start_date,
        end_date: filters.value.end_date
      })
      rows.push(...response.items)
    }
    const header = ['Time', 'Model', 'Endpoint', 'Input Tokens', 'Output Tokens', 'Standard Cost', 'Actual Cost', 'Support Code']
    const body = rows.map((row) => [
      row.created_at,
      row.model,
      resolveEndpoint(row),
      row.input_tokens,
      row.output_tokens,
      row.total_cost,
      row.actual_cost,
      resolveSupportCode(row)
    ].map(escapeCSVValue).join(','))
    const blob = new Blob([[header.join(','), ...body].join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `usage_${filters.value.start_date}_to_${filters.value.end_date}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    appStore.showSuccess(t('usage.exportSuccess'))
  } catch {
    appStore.showError(t('usage.exportFailed'))
  } finally {
    exporting.value = false
  }
}

function buildDailySeries(points: TrendDataPoint[]): DailyUsage[] {
  const buckets = new Map<string, DailyUsage>()

  for (const point of points) {
    const key = String(point.date || '').slice(0, 10)
    if (!/^\d{4}-\d{2}-\d{2}$/.test(key)) continue
    const existing = buckets.get(key) || {
      key,
      label: formatDailyLabel(key),
      requests: 0,
      tokens: 0,
      cost: 0
    }
    existing.requests += Number(point.requests || 0)
    existing.tokens += Number(point.total_tokens || 0)
    existing.cost += Number(point.actual_cost || 0)
    buckets.set(key, existing)
  }

  const result: DailyUsage[] = []
  const cursor = new Date(trendStart.getFullYear(), trendStart.getMonth(), trendStart.getDate())
  const lastDay = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  while (cursor <= lastDay) {
    const key = toDateKey(cursor)
    result.push(buckets.get(key) || {
      key,
      label: formatDailyLabel(key),
      requests: 0,
      tokens: 0,
      cost: 0
    })
    cursor.setDate(cursor.getDate() + 1)
  }
  return result
}

function formatDailyLabel(key: string) {
  const [, month = '', day = ''] = key.split('-')
  return `${month}/${day}`
}

function formatTrendCost(value: number) {
  return formatMoney(value)
}

function formatTrendNumber(value: number) {
  return new Intl.NumberFormat(undefined, {
    notation: value >= 1000 ? 'compact' : 'standard',
    maximumFractionDigits: 1
  }).format(value)
}

function formatUsageKind(row: UsageLog) {
  const endpoint = resolveEndpoint(row)
  if (row.image_count > 0 || endpoint.includes('/images/')) return t('usage.workbench.usageKindImage')
  if (endpoint.includes('/chat/')) return t('usage.workbench.usageKindChat')
  if (row.api_key_id) return t('usage.workbench.usageKindThirdParty')
  return t('usage.workbench.usageKindWeb')
}

function resolveEndpoint(row: UsageLog) {
  return row.inbound_endpoint || '-'
}

function resolveSupportCode(row: UsageLog) {
  return row.request_id || ''
}

async function copySupportCode(row: UsageLog) {
  const supportCode = resolveSupportCode(row)
  if (!supportCode) return
  const copied = await copyToClipboard(supportCode, t('usage.workbench.supportCodeCopied'))
  if (!copied) return
  copiedSupportCode.value = supportCode
  window.setTimeout(() => {
    if (copiedSupportCode.value === supportCode) copiedSupportCode.value = null
  }, 2000)
}

function formatSource(row: UsageLog) {
  return row.api_key_id ? t('usage.workbench.usageKindThirdParty') : t('usage.workbench.usageKindWeb')
}

function formatUsageAmount(row: UsageLog) {
  if (row.image_count > 0) {
    const count = formatNumber(Number(row.image_count || 0))
    if (row.image_size) return t('usage.workbench.imageAmountWithSize', { count, size: row.image_size })
    return t('usage.workbench.imageAmount', { count })
  }
  const tokens = Number(row.input_tokens || 0) + Number(row.output_tokens || 0) + Number(row.cache_creation_tokens || 0) + Number(row.cache_read_tokens || 0)
  return t('usage.workbench.tokenAmount', { count: formatNumber(tokens) })
}

function formatBillingType(row: UsageLog) {
  if (isNoCharge(row)) return t('usage.workbench.billingNoCharge')
  if (Number(row.billing_type) === 1) return t('usage.workbench.billingSubscription')
  return t('usage.workbench.billingBalance')
}

function formatBillingBasis(row: UsageLog) {
  const standardCost = Number(row.total_cost || 0)
  const actualCost = Number(row.actual_cost || 0)
  if (isNoCharge(row)) return t('usage.workbench.noChargeBasis')
  if (standardCost > 0 && Math.abs(standardCost - actualCost) > 0.000001) {
    return t('usage.workbench.standardVsActual', {
      standard: formatCost(standardCost),
      actual: formatCost(actualCost)
    })
  }
  return t('usage.workbench.actualChargeBasis', { amount: formatCost(actualCost) })
}

function formatPerformanceStatus(row: UsageLog) {
  if (!hasLatencyRecord(row)) return t('usage.workbench.performanceUnknown')
  if (isSlowFirstToken(row)) return t('usage.workbench.performanceSlowFirstToken')
  if (isSlowTotalDuration(row)) return t('usage.workbench.performanceSlowTotal')
  return t('usage.workbench.performanceHealthy')
}

function formatPerformanceSummary(row: UsageLog) {
  if (!hasLatencyRecord(row)) return t('usage.workbench.performanceNoRecord')
  return t('usage.workbench.performanceSummary', {
    firstToken: formatLatency(row.first_token_ms),
    duration: formatLatency(row.duration_ms)
  })
}

function formatPerformanceHint(row: UsageLog) {
  if (isSlowFirstToken(row)) return t('usage.workbench.performanceSlowFirstTokenHint')
  if (isSlowTotalDuration(row)) return t('usage.workbench.performanceSlowTotalHint')
  return ''
}

function performanceBadgeClass(row: UsageLog) {
  if (!hasLatencyRecord(row)) return 'is-muted'
  if (isSlowFirstToken(row) || isSlowTotalDuration(row)) return 'is-warning'
  return 'is-normal'
}

function hasLatencyRecord(row: UsageLog) {
  return isFiniteLatency(row.duration_ms) || isFiniteLatency(row.first_token_ms)
}

function isSlowFirstToken(row: UsageLog) {
  return isFiniteLatency(row.first_token_ms) && Number(row.first_token_ms) > 5000
}

function isSlowTotalDuration(row: UsageLog) {
  return isFiniteLatency(row.duration_ms) && Number(row.duration_ms) >= 60000
}

function isFiniteLatency(value: number | null | undefined) {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function formatLatency(value: number | null | undefined) {
  if (!isFiniteLatency(value)) return '-'
  const ms = Number(value)
  if (ms < 1000) return `${Math.round(ms)} ms`
  if (ms < 60000) {
    const digits = ms < 10000 ? 1 : 0
    return `${(ms / 1000).toFixed(digits)} s`
  }
  const minutes = Math.floor(ms / 60000)
  const seconds = Math.round((ms % 60000) / 1000)
  if (seconds <= 0) return `${minutes}m`
  return `${minutes}m ${seconds}s`
}

function formatCost(value: number | null | undefined) {
  return formatMoney(Number(value || 0))
}

function formatCostTitle(value: number | null | undefined) {
  return formatMoneyTitle(Number(value || 0))
}

function isNoCharge(row: UsageLog) {
  return Number(row.actual_cost || 0) <= 0
}

function isZeroTokenCharged(row: UsageLog) {
  const tokenTotal = Number(row.input_tokens || 0)
    + Number(row.output_tokens || 0)
    + Number(row.cache_creation_tokens || 0)
    + Number(row.cache_read_tokens || 0)
  return tokenTotal <= 0 && Number(row.actual_cost || 0) > 0
}

function formatNumber(value: number) {
  return Number(value || 0).toLocaleString()
}

function formatDateTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

function toDateKey(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
</script>

<style scoped>
.usage-workbench {
  display: grid;
  gap: 1.5rem;
}

.usage-summary-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.usage-summary-card,
.usage-panel,
.usage-explainer {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.usage-filters {
  display: grid;
  grid-template-columns: minmax(10rem, 1fr) minmax(10rem, 1fr) minmax(9rem, 0.8fr) minmax(9rem, 0.8fr) auto;
  gap: 0.75rem;
  align-items: end;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 1rem;
}

.filter-field {
  display: grid;
  gap: 0.4rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.78rem;
  font-weight: 700;
}

.filter-actions,
.usage-pagination,
.usage-pagination > div {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.usage-pagination {
  justify-content: space-between;
  border-top: 1px solid var(--ssxz-border);
  color: var(--ssxz-text-muted);
  padding: 0.85rem 1rem;
  font-size: 0.8rem;
}

.usage-pagination strong {
  min-width: 4rem;
  color: var(--ssxz-text-primary);
  text-align: center;
}

.usage-summary-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.85rem;
  align-items: center;
  border-radius: var(--ssxz-radius-card);
  padding: 1.25rem;
}

.summary-icon {
  display: grid;
  width: 2.45rem;
  height: 2.45rem;
  place-items: center;
  border-radius: var(--ssxz-radius-button);
  background: color-mix(in srgb, var(--ssxz-action-soft) 78%, transparent);
  color: var(--ssxz-action);
}

.usage-summary-card span,
.panel-heading p,
.usage-empty span,
.usage-table th {
  color: var(--ssxz-text-muted);
}

.usage-summary-card span {
  font-size: 0.82rem;
  font-weight: 800;
}

.usage-summary-card strong {
  display: block;
  margin-top: 0.15rem;
  color: var(--ssxz-text-primary);
  font-size: clamp(1.45rem, 3vw, 2rem);
  letter-spacing: 0;
}

.usage-summary-card p {
  margin: 0.28rem 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  line-height: 1.55;
}

.usage-summary-card p.is-warning {
  color: var(--ssxz-warning, #b45309);
  font-weight: 750;
}

.panel-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  font-size: 0.82rem;
  font-weight: 850;
}

.summary-action {
  justify-self: end;
  text-decoration: none;
}

.usage-panel {
  overflow: hidden;
  border-radius: var(--ssxz-radius-card);
}

.usage-trends-section {
  display: grid;
  gap: 1rem;
}

.usage-trends-section .panel-heading {
  padding: 0;
  border-bottom: 0;
}

.usage-explainer {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(18rem, 1.4fr);
  gap: 1rem;
  align-items: start;
  border-radius: var(--ssxz-radius-card);
  padding: 1.25rem;
}

.usage-explainer strong {
  display: block;
  color: var(--ssxz-text-primary);
  font-size: 0.96rem;
  font-weight: 850;
}

.usage-explainer p,
.usage-explainer li {
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  line-height: 1.55;
}

.usage-explainer p {
  margin: 0.28rem 0 0;
}

.usage-explainer ul {
  display: grid;
  gap: 0.28rem;
  margin: 0;
  padding-left: 1.05rem;
}

.panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 1rem;
}

.panel-heading h3 {
  margin: 0;
  color: var(--ssxz-text-primary);
  font-size: 1.05rem;
  font-weight: 850;
}

.panel-heading p {
  margin: 0.25rem 0 0;
  font-size: 0.82rem;
  line-height: 1.55;
}

.panel-badge {
  border: 1px solid color-mix(in srgb, var(--ssxz-action) 35%, var(--ssxz-border));
  background: color-mix(in srgb, var(--ssxz-action-soft) 72%, transparent);
  color: var(--ssxz-action);
  padding: 0.3rem 0.62rem;
}

.refresh-button {
  flex: 0 0 auto;
}

.refresh-button:disabled {
  cursor: not-allowed;
  opacity: 0.62;
}

.usage-trend-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
}

.usage-empty {
  display: grid;
  min-height: 14rem;
  place-items: center;
  align-content: center;
  gap: 0.45rem;
  color: var(--ssxz-text-muted);
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface);
  padding: 1.4rem;
  text-align: center;
}

.usage-empty.compact {
  min-height: 10rem;
}

.usage-empty__icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
}

.usage-empty__icon.is-warning {
  background: color-mix(in srgb, var(--ssxz-warning) 12%, var(--ssxz-surface));
  color: var(--ssxz-warning);
}

.usage-empty strong {
  color: var(--ssxz-text-primary);
  font-size: 1rem;
}

.usage-empty span {
  max-width: 28rem;
  line-height: 1.6;
}

.usage-table-wrap {
  overflow-x: auto;
  padding: 0.85rem;
}

.usage-table {
  width: 100%;
  min-width: 58rem;
  border-collapse: collapse;
  color: var(--ssxz-text-secondary);
  font-size: 0.86rem;
}

.usage-table th,
.usage-table td {
  border-bottom: 1px solid var(--ssxz-border);
  padding: 0.8rem 0.75rem;
  text-align: left;
  vertical-align: middle;
}

.usage-table th {
  background: color-mix(in srgb, var(--ssxz-surface-muted) 70%, transparent);
  font-size: 0.76rem;
  font-weight: 850;
}

.usage-table tbody tr:hover {
  background: color-mix(in srgb, var(--ssxz-action-soft) 38%, transparent);
}

.model-cell {
  color: var(--ssxz-text-primary);
  font-weight: 800;
}

.billing-cell,
.performance-cell,
.support-cell {
  display: grid;
  min-width: 9rem;
  gap: 0.18rem;
}

.billing-cell span {
  color: var(--ssxz-text-primary);
  font-weight: 800;
}

.billing-cell small {
  color: var(--ssxz-text-muted);
  font-size: 0.74rem;
  line-height: 1.35;
}

.performance-badge {
  width: fit-content;
  border-radius: 999px;
  padding: 0.14rem 0.48rem;
  font-size: 0.72rem;
  font-weight: 850;
}

.performance-badge.is-normal {
  background: color-mix(in srgb, var(--ssxz-success, #16a34a) 14%, transparent);
  color: var(--ssxz-success, #15803d);
}

.performance-badge.is-warning {
  background: color-mix(in srgb, var(--ssxz-warning, #d97706) 18%, transparent);
  color: var(--ssxz-warning, #b45309);
}

.performance-badge.is-muted {
  background: color-mix(in srgb, var(--ssxz-surface-muted) 84%, transparent);
  color: var(--ssxz-text-muted);
}

.performance-cell small {
  color: var(--ssxz-text-muted);
  font-size: 0.74rem;
  line-height: 1.35;
}

.performance-hint {
  max-width: 15rem;
}

.support-cell code {
  width: fit-content;
  max-width: 14rem;
  overflow-wrap: anywhere;
  border-radius: 0.45rem;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 80%, transparent);
  color: var(--ssxz-text-primary);
  padding: 0.16rem 0.34rem;
  font-size: 0.74rem;
}

.support-code-button {
  width: fit-content;
  border: 0;
  background: transparent;
  color: var(--ssxz-action);
  cursor: pointer;
  font-size: 0.72rem;
  font-weight: 800;
  padding: 0;
  text-align: left;
}

.support-code-button:hover {
  text-decoration: underline;
}

.usage-cost-note {
  display: block;
  margin-top: 0.16rem;
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
  font-weight: 750;
}

@media (max-width: 860px) {
  .usage-summary-grid {
    grid-template-columns: 1fr;
  }

  .usage-explainer {
    grid-template-columns: 1fr;
  }

  .usage-summary-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .summary-action {
    grid-column: 1 / -1;
    width: 100%;
  }

  .panel-heading {
    flex-direction: column;
  }

  .usage-filters {
    grid-template-columns: 1fr;
  }

  .usage-trend-grid {
    grid-template-columns: 1fr;
  }

  .filter-actions,
  .filter-actions .btn {
    width: 100%;
  }

  .usage-pagination {
    align-items: stretch;
    flex-direction: column;
  }

  .usage-pagination > div {
    justify-content: space-between;
  }
}
</style>
