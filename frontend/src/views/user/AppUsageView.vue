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

      <section class="usage-panel">
        <header class="panel-heading">
          <div>
            <h3>{{ t('usage.workbench.monthlyUsageTitle') }}</h3>
            <p>{{ t('usage.workbench.monthlyUsageDescription') }}</p>
          </div>
          <span v-if="hasMonthlyUsage" class="panel-badge">{{ usageDataBadge }}</span>
        </header>

        <div v-if="trendLoadError" class="usage-empty">
          <div class="usage-empty__icon is-warning"><Icon name="exclamationTriangle" size="lg" /></div>
          <strong>{{ t('usage.workbench.trendLoadError') }}</strong>
          <span>{{ t('usage.workbench.trendLoadErrorHint') }}</span>
        </div>

        <div v-else-if="hasMonthlyUsage" class="usage-chart" :aria-label="t('usage.workbench.monthlyChartLabel')">
          <div
            v-for="item in monthlySeries"
            :key="item.key"
            class="chart-column"
            :title="item.cost > 0 ? formatCostTitle(item.cost) : undefined"
          >
            <div class="chart-bar-track">
              <div class="chart-bar" :style="{ height: chartBarHeight(item) }" />
            </div>
            <strong>{{ item.label }}</strong>
            <span>{{ formatMonthlyValue(item) }}</span>
          </div>
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
          <button type="button" class="btn btn-secondary btn-sm refresh-button" :disabled="controlsDisabled" @click="refreshUsageOverview">
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
              :disabled="controlsDisabled"
              @update:model-value="updateApiKeyFilter"
            />
          </label>
          <label class="filter-field">
            <span>{{ t('usage.model') }}</span>
            <input
              v-model.trim="filters.model"
              class="f0-input-control"
              type="search"
              :disabled="controlsDisabled"
              :placeholder="t('usage.workbench.modelFilterPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </label>
          <label class="filter-field">
            <span>{{ t('usage.workbench.startDate') }}</span>
            <input v-model="filters.start_date" class="f0-input-control" type="date" :disabled="controlsDisabled" @change="applyFilters" />
          </label>
          <label class="filter-field">
            <span>{{ t('usage.workbench.endDate') }}</span>
            <input v-model="filters.end_date" class="f0-input-control" type="date" :disabled="controlsDisabled" @change="applyFilters" />
          </label>
          <div class="filter-actions">
            <button type="button" class="btn btn-secondary" :disabled="controlsDisabled" @click="resetFilters">
              {{ t('common.reset') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="controlsDisabled || totalRows === 0" @click="exportToCSV">
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

        <div v-else class="usage-native-table" data-testid="usage-native-table">
          <UsageTable
            :data="usageRows"
            :columns="usageTableColumns"
            :show-account-billing="false"
            :show-upstream-endpoint="false"
            :sticky-first-column="false"
            :sticky-actions-column="false"
            flat
          />
        </div>
        <div v-if="!loading && totalRows > 0" class="usage-pagination">
          <span>{{ t('usage.workbench.paginationSummary', { total: totalRows }) }}</span>
          <div>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="controlsDisabled || page <= 1" @click="changePage(page - 1)">
              {{ t('pagination.previous') }}
            </button>
            <strong>{{ page }} / {{ totalPages }}</strong>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="controlsDisabled || page >= totalPages" @click="changePage(page + 1)">
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
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import UsageTable from '@/components/admin/usage/UsageTable.vue'
import { keysAPI, usageAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey, TrendDataPoint, UsageLog, UsageQueryParams, UsageStatsResponse } from '@/types'
import type { Column } from '@/components/common/types'
import {
  formatCurrency as formatMoney,
  formatCurrencyTitle as formatMoneyTitle
} from '@/utils/format'

interface MonthlyUsage {
  key: string
  label: string
  requests: number
  tokens: number
  cost: number
}

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const usageRows = ref<UsageLog[]>([])
const usageStats = ref<UsageStatsResponse | null>(null)
const monthlySeries = ref<MonthlyUsage[]>([])
const loading = ref(false)
const detailsLoadError = ref(false)
const statsLoadError = ref(false)
const trendLoadError = ref(false)
const balanceRefreshError = ref(false)
const apiKeys = ref<ApiKey[]>([])
const exporting = ref(false)
const page = ref(1)
const pageSize = 20
const totalRows = ref(0)
const totalPages = ref(1)
let loadRequestSequence = 0
let balanceRefreshPromise: Promise<void> | null = null

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
const hasMonthlyUsage = computed(() => monthlySeries.value.some((item) => item.requests > 0 || item.tokens > 0 || item.cost > 0))
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
const usageTableColumns = computed<Column[]>(() => [
  { key: 'api_key', label: t('usage.apiKeyFilter'), class: 'usage-col-api-key' },
  { key: 'model', label: t('usage.model'), sortable: true, class: 'usage-col-model' },
  { key: 'reasoning_effort', label: t('usage.reasoningEffort'), class: 'usage-col-reasoning' },
  { key: 'endpoint', label: t('usage.endpoint'), class: 'usage-col-endpoint' },
  { key: 'ip_address', label: t('admin.usage.ipAddress'), class: 'usage-col-ip' },
  { key: 'group', label: t('admin.usage.group'), class: 'usage-col-group' },
  { key: 'stream', label: t('usage.type'), class: 'usage-col-type' },
  { key: 'billing_mode', label: t('admin.usage.billingMode'), class: 'usage-col-billing' },
  { key: 'tokens', label: t('usage.tokens'), class: 'usage-col-tokens' },
  { key: 'cost', label: t('usage.cost'), class: 'usage-col-cost' },
  { key: 'latency', label: t('usage.duration'), class: 'usage-col-latency' },
  { key: 'created_at', label: t('usage.time'), sortable: true, class: 'usage-col-time' },
  { key: 'request_id', label: t('usage.workbench.supportCode'), class: 'usage-col-support' }
])
const controlsDisabled = computed(() => loading.value || exporting.value)
const chartMax = computed(() => Math.max(1, ...monthlySeries.value.map((item) => chartMetric(item))))

onMounted(() => {
  void loadApiKeys()
  void loadUsageOverview()
  void refreshBalance()
})

async function loadUsageOverview() {
  const requestSequence = ++loadRequestSequence
  const requestedPage = page.value
  const requestedApiKeyId = filters.value.api_key_id ? Number(filters.value.api_key_id) : undefined
  const requestedModel = filters.value.model?.trim() || undefined
  const requestedStartDate = filters.value.start_date || monthStartKey
  const requestedEndDate = filters.value.end_date || todayKey

  loading.value = true
  detailsLoadError.value = false
  statsLoadError.value = false
  trendLoadError.value = false

  const [statsResult, logsResult, trendResult] = await Promise.allSettled([
    usageAPI.getStatsByDateRange(
      monthStartKey,
      todayKey,
      requestedApiKeyId
    ),
    usageAPI.query({
      page: requestedPage,
      page_size: pageSize,
      api_key_id: requestedApiKeyId,
      model: requestedModel,
      start_date: requestedStartDate,
      end_date: requestedEndDate
    }),
    usageAPI.getDashboardTrend({
      start_date: trendStartKey,
      end_date: todayKey,
      granularity: 'day'
    })
  ])

  if (requestSequence !== loadRequestSequence) return

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
    monthlySeries.value = buildMonthlySeries(trendResult.value.trend || [])
  } else {
    monthlySeries.value = []
    trendLoadError.value = true
  }

  loading.value = false
}

function refreshBalance() {
  if (balanceRefreshPromise) return balanceRefreshPromise

  balanceRefreshError.value = false
  const refreshPromise = authStore.refreshUser()
    .then(() => undefined)
    .catch(() => {
      balanceRefreshError.value = true
    })
    .finally(() => {
      if (balanceRefreshPromise === refreshPromise) balanceRefreshPromise = null
    })
  balanceRefreshPromise = refreshPromise
  return refreshPromise
}

function refreshUsageOverview() {
  if (controlsDisabled.value) return
  void loadUsageOverview()
  void refreshBalance()
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
  if (controlsDisabled.value) return
  page.value = 1
  void loadUsageOverview()
}

function updateApiKeyFilter(value: string | number | boolean | null) {
  if (controlsDisabled.value) return
  filters.value.api_key_id = typeof value === 'number' ? value : undefined
  applyFilters()
}

function resetFilters() {
  if (controlsDisabled.value) return
  filters.value = {
    api_key_id: undefined,
    model: '',
    start_date: monthStartKey,
    end_date: todayKey
  }
  applyFilters()
}

function changePage(nextPage: number) {
  if (controlsDisabled.value) return
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
  if (controlsDisabled.value || totalRows.value === 0) return
  const exportFilters = { ...filters.value }
  const exportTotalRows = totalRows.value
  exporting.value = true
  try {
    const rows: UsageLog[] = []
    const exportPageSize = 100
    const pages = Math.ceil(exportTotalRows / exportPageSize)
    for (let exportPage = 1; exportPage <= pages; exportPage += 1) {
      const response = await usageAPI.query({
        page: exportPage,
        page_size: exportPageSize,
        api_key_id: exportFilters.api_key_id ? Number(exportFilters.api_key_id) : undefined,
        model: exportFilters.model?.trim() || undefined,
        start_date: exportFilters.start_date,
        end_date: exportFilters.end_date
      })
      rows.push(...response.items)
    }
    const header = ['Time', 'Model', 'Endpoint', 'Group', 'Input Tokens', 'Output Tokens', 'Standard Cost', 'Actual Cost', 'Support Code']
    const body = rows.map((row) => [
      row.created_at,
      row.model,
      resolveEndpoint(row),
      resolveGroupName(row),
      row.input_tokens,
      row.output_tokens,
      row.total_cost,
      row.actual_cost,
      resolveSupportCode(row)
    ].map(escapeCSVValue).join(','))
    const blob = new Blob([[header.join(','), ...body].join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = window.URL.createObjectURL(blob)
    let link: HTMLAnchorElement | null = null
    try {
      link = document.createElement('a')
      link.href = url
      link.download = `usage_${exportFilters.start_date}_to_${exportFilters.end_date}.csv`
      document.body.appendChild(link)
      link.click()
    } finally {
      try {
        link?.remove()
      } finally {
        window.setTimeout(() => window.URL.revokeObjectURL(url), 0)
      }
    }
    appStore.showSuccess(t('usage.exportSuccess'))
  } catch {
    appStore.showError(t('usage.exportFailed'))
  } finally {
    exporting.value = false
  }
}

function buildMonthlySeries(points: TrendDataPoint[]): MonthlyUsage[] {
  const buckets = new Map<string, MonthlyUsage>()

  for (const point of points) {
    const date = new Date(point.date)
    if (Number.isNaN(date.getTime())) continue
    const key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
    const existing = buckets.get(key) || {
      key,
      label: t('usage.workbench.monthLabel', { month: date.getMonth() + 1 }),
      requests: 0,
      tokens: 0,
      cost: 0
    }
    existing.requests += Number(point.requests || 0)
    existing.tokens += Number(point.total_tokens || 0)
    existing.cost += Number(point.actual_cost || 0)
    buckets.set(key, existing)
  }

  return Array.from(buckets.values()).sort((a, b) => a.key.localeCompare(b.key))
}

function chartMetric(item: MonthlyUsage) {
  if (item.cost > 0) return item.cost
  if (item.tokens > 0) return item.tokens
  return item.requests
}

function chartBarHeight(item: MonthlyUsage) {
  const percent = Math.max(8, Math.round((chartMetric(item) / chartMax.value) * 100))
  return `${percent}%`
}

function formatMonthlyValue(item: MonthlyUsage) {
  if (item.cost > 0) return formatMoney(item.cost)
  if (item.tokens > 0) return t('usage.workbench.tokenAmount', { count: formatNumber(item.tokens) })
  return t('usage.workbench.requestCount', { count: formatNumber(item.requests) })
}

function resolveEndpoint(row: UsageLog) {
  return row.inbound_endpoint || '-'
}

function resolveSupportCode(row: UsageLog) {
  return row.request_id || ''
}

function resolveGroupName(row: UsageLog) {
  const name = row.group?.name?.trim()
  if (name) return name
  if (row.group_id) return `#${row.group_id}`
  return t('usage.workbench.noGroup')
}

function formatCostTitle(value: number | null | undefined) {
  return formatMoneyTitle(Number(value || 0))
}

function formatNumber(value: number) {
  return Number(value || 0).toLocaleString()
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
.usage-empty span {
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

.usage-chart {
  display: grid;
  min-height: 18rem;
  grid-template-columns: repeat(auto-fit, minmax(5.2rem, 1fr));
  gap: 0.8rem;
  align-items: end;
  padding: 1.2rem;
}

.chart-column {
  display: grid;
  min-height: 14rem;
  align-items: end;
  gap: 0.4rem;
  text-align: center;
}

.chart-bar-track {
  position: relative;
  display: flex;
  height: 10rem;
  align-items: flex-end;
  overflow: hidden;
  border-radius: 0.9rem;
  background:
    repeating-linear-gradient(
      to top,
      color-mix(in srgb, var(--ssxz-border) 45%, transparent) 0,
      color-mix(in srgb, var(--ssxz-border) 45%, transparent) 1px,
      transparent 1px,
      transparent 2rem
    ),
    color-mix(in srgb, var(--ssxz-surface-muted) 80%, transparent);
}

.chart-bar {
  width: min(100%, 4.25rem);
  margin-inline: auto;
  border-radius: 0.9rem 0.9rem 0 0;
  background: color-mix(in srgb, var(--ssxz-text-muted) 72%, var(--ssxz-surface));
  box-shadow: 0 -6px 18px rgb(0 0 0 / 12%);
}

.chart-column strong {
  color: var(--ssxz-text-primary);
  font-size: 0.82rem;
}

.chart-column span {
  color: var(--ssxz-text-muted);
  font-size: 0.74rem;
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

.usage-native-table {
  overflow: hidden;
  padding: 0.75rem;
}

.usage-native-table :deep(.overflow-auto),
.usage-native-table :deep(.table-wrapper) {
  max-width: 100%;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
}

.usage-native-table :deep(table) {
  width: 100%;
  min-width: 105.5rem;
  table-layout: fixed;
}

.usage-native-table :deep(.table-header),
.usage-native-table :deep(.table-body) {
  background: var(--ssxz-surface-raised);
}

.usage-native-table :deep(th) {
  background: color-mix(in srgb, var(--ssxz-surface-muted) 72%, transparent);
  color: var(--ssxz-subtle);
  font-size: 0.72rem;
  letter-spacing: 0;
  white-space: nowrap;
}

.usage-native-table :deep(td) {
  overflow: hidden;
  border-color: color-mix(in srgb, var(--ssxz-border) 70%, transparent);
  color: var(--ssxz-body);
  vertical-align: middle;
}

.usage-native-table :deep(tbody tr:hover) {
  background: color-mix(in srgb, var(--ssxz-action-soft) 34%, transparent);
}

.usage-native-table :deep(.usage-col-api-key) { width: 10rem; }
.usage-native-table :deep(.usage-col-model) { width: 7.5rem; }
.usage-native-table :deep(.usage-col-reasoning) { width: 5rem; }
.usage-native-table :deep(.usage-col-endpoint) { width: 8rem; }
.usage-native-table :deep(.usage-col-ip) { width: 8.5rem; }
.usage-native-table :deep(.usage-col-group) { width: 11rem; }
.usage-native-table :deep(.usage-col-type) { width: 4.5rem; }
.usage-native-table :deep(.usage-col-billing) { width: 5.5rem; }
.usage-native-table :deep(.usage-col-tokens) { width: 9.5rem; }
.usage-native-table :deep(.usage-col-cost) { width: 7.5rem; }
.usage-native-table :deep(.usage-col-latency) { width: 7.5rem; }
.usage-native-table :deep(.usage-col-time) { width: 10rem; }
.usage-native-table :deep(.usage-col-support) { width: 10.5rem; }

.usage-native-table :deep(.usage-col-group > span),
.usage-native-table :deep(.usage-col-support > div) {
  min-width: 0;
  max-width: 100%;
}

.usage-native-table :deep(.usage-col-group > span) {
  overflow: hidden;
  background: var(--ssxz-action-soft);
  color: var(--ssxz-action);
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 767px) {
  .usage-native-table {
    padding: 0.75rem;
  }

  .usage-native-table :deep(.space-y-3 > div) {
    border-color: var(--ssxz-border);
    background: var(--ssxz-surface-raised);
  }
}

@media (max-width: 1100px) {
  .usage-filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .filter-actions {
    grid-column: 1 / -1;
    justify-content: flex-end;
  }

  .filter-actions .btn {
    min-width: 6rem;
  }
}

@media (max-width: 860px) {
  .usage-summary-grid {
    grid-template-columns: 1fr;
  }

  .usage-explainer {
    grid-template-columns: 1fr;
  }

  .usage-detail-grid {
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
