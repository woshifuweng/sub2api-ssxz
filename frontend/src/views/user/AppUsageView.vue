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
            <strong>{{ balanceText }}</strong>
            <p :class="{ 'is-warning': balanceRefreshError }">{{ balanceDescriptionText }}</p>
          </div>
          <RouterLink to="/app/purchase" class="summary-action">
            {{ t('usage.workbench.recharge') }}
          </RouterLink>
        </article>

        <article class="usage-summary-card">
          <div class="summary-icon">
            <Icon name="chartBar" size="sm" />
          </div>
          <div>
            <span>{{ t('usage.workbench.monthlyCostTitle') }}</span>
            <strong>{{ monthlyCostText }}</strong>
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
          <span v-if="hasMonthlyUsage" class="panel-badge">
            {{ t('usage.workbench.realDataBadge') }}
          </span>
        </header>

        <div v-if="trendLoadError" class="usage-empty">
          <Icon name="exclamationTriangle" size="lg" />
          <strong>{{ t('usage.workbench.trendLoadError') }}</strong>
          <span>{{ t('usage.workbench.trendLoadErrorHint') }}</span>
        </div>

        <div v-else-if="hasMonthlyUsage" class="usage-chart" :aria-label="t('usage.workbench.monthlyChartLabel')">
          <div
            v-for="item in monthlySeries"
            :key="item.key"
            class="chart-column"
          >
            <div class="chart-bar-track">
              <div class="chart-bar" :style="{ height: chartBarHeight(item) }" />
            </div>
            <strong>{{ item.label }}</strong>
            <span>{{ formatMonthlyValue(item) }}</span>
          </div>
        </div>

        <div v-else class="usage-empty">
          <Icon name="chartBar" size="lg" />
          <strong>{{ t('usage.workbench.noMonthlyUsageTitle') }}</strong>
          <span>{{ t('usage.workbench.noMonthlyUsageDescription') }}</span>
        </div>
      </section>

      <section class="usage-panel">
        <header class="panel-heading">
          <div>
            <h3>{{ t('usage.workbench.usageDetailsTitle') }}</h3>
            <p>{{ t('usage.workbench.usageDetailsDescription') }}</p>
          </div>
          <button type="button" class="refresh-button" :disabled="loading" @click="loadUsageOverview">
            <Icon name="refresh" size="xs" />
            {{ t('usage.workbench.refresh') }}
          </button>
        </header>

        <div v-if="loading" class="usage-empty compact">
          <Icon name="sync" size="md" />
          <strong>{{ t('usage.workbench.loading') }}</strong>
        </div>

        <div v-else-if="detailsLoadError" class="usage-empty compact">
          <Icon name="exclamationTriangle" size="md" />
          <strong>{{ t('usage.workbench.detailsLoadError') }}</strong>
          <span>{{ t('usage.workbench.detailsLoadErrorHint') }}</span>
        </div>

        <div v-else-if="usageRows.length === 0" class="usage-empty compact">
          <Icon name="inbox" size="md" />
          <strong>{{ t('usage.workbench.noDetailsTitle') }}</strong>
          <span>{{ t('usage.workbench.noDetailsDescription') }}</span>
        </div>

        <div v-else class="usage-table-wrap">
          <table class="usage-table">
            <thead>
              <tr>
                <th>{{ t('usage.workbench.createdAt') }}</th>
                <th>{{ t('usage.workbench.kind') }}</th>
                <th>{{ t('usage.workbench.model') }}</th>
                <th>{{ t('usage.workbench.amount') }}</th>
                <th>{{ t('usage.workbench.billingBasis') }}</th>
                <th>{{ t('usage.workbench.fee') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in usageRows" :key="row.id || row.request_id">
                <td>{{ formatDateTime(row.created_at) }}</td>
                <td>{{ formatUsageKind(row) }}</td>
                <td class="model-cell">{{ row.model || '-' }}</td>
                <td>{{ formatUsageAmount(row) }}</td>
                <td class="billing-cell">
                  <span>{{ formatBillingType(row) }}</span>
                  <small>{{ formatBillingBasis(row) }}</small>
                </td>
                <td>
                  <span>{{ formatCost(row.actual_cost) }}</span>
                  <small v-if="isNoCharge(row)" class="usage-cost-note">
                    {{ t('usage.workbench.noCharge') }}
                  </small>
                </td>
              </tr>
            </tbody>
          </table>
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
import { usageAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { TrendDataPoint, UsageLog, UsageStatsResponse } from '@/types'

interface MonthlyUsage {
  key: string
  label: string
  requests: number
  tokens: number
  cost: number
}

const { t } = useI18n()
const authStore = useAuthStore()
const usageRows = ref<UsageLog[]>([])
const usageStats = ref<UsageStatsResponse | null>(null)
const monthlySeries = ref<MonthlyUsage[]>([])
const loading = ref(false)
const detailsLoadError = ref(false)
const statsLoadError = ref(false)
const trendLoadError = ref(false)
const balanceRefreshError = ref(false)

const today = new Date()
const monthStart = new Date(today.getFullYear(), today.getMonth(), 1)
const trendStart = new Date(today.getFullYear(), today.getMonth() - 5, 1)

const todayKey = toDateKey(today)
const monthStartKey = toDateKey(monthStart)
const trendStartKey = toDateKey(trendStart)

const balanceText = computed(() => formatCurrency(authStore.user?.balance || 0, 2))
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
  return formatCurrency(usageStats.value?.total_actual_cost || 0, 4)
})
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
const chartMax = computed(() => Math.max(1, ...monthlySeries.value.map((item) => chartMetric(item))))

onMounted(() => {
  void loadUsageOverview()
})

async function loadUsageOverview() {
  loading.value = true
  detailsLoadError.value = false
  statsLoadError.value = false
  trendLoadError.value = false
  balanceRefreshError.value = false

  const [statsResult, logsResult, trendResult, userResult] = await Promise.allSettled([
    usageAPI.getStatsByDateRange(monthStartKey, todayKey),
    usageAPI.query({
      page: 1,
      page_size: 8,
      start_date: monthStartKey,
      end_date: todayKey
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
  } else {
    usageRows.value = []
    detailsLoadError.value = true
  }

  if (trendResult.status === 'fulfilled') {
    monthlySeries.value = buildMonthlySeries(trendResult.value.trend || [])
  } else {
    monthlySeries.value = []
    trendLoadError.value = true
  }

  if (userResult?.status === 'rejected') {
    balanceRefreshError.value = true
  }

  loading.value = false
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
  if (item.cost > 0) return formatCurrency(item.cost, item.cost < 0.01 ? 6 : 4)
  if (item.tokens > 0) return t('usage.workbench.tokenAmount', { count: formatNumber(item.tokens) })
  return t('usage.workbench.requestCount', { count: formatNumber(item.requests) })
}

function formatUsageKind(row: UsageLog) {
  if (row.api_key_id) return t('usage.workbench.usageKindThirdParty')
  if (row.image_count > 0 || row.inbound_endpoint?.includes('/images/')) return t('usage.workbench.usageKindImage')
  if (row.inbound_endpoint?.includes('/chat/')) return t('usage.workbench.usageKindChat')
  return t('usage.workbench.usageKindWeb')
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

function formatCost(value: number | null | undefined) {
  const cost = Number(value || 0)
  return formatCurrency(cost, cost > 0 && cost < 0.01 ? 6 : 4)
}

function isNoCharge(row: UsageLog) {
  return Number(row.actual_cost || 0) <= 0
}

function formatCurrency(value: number, digits: number) {
  return `$${Number(value || 0).toFixed(digits)}`
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
  gap: 1rem;
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
  background: var(--ssxz-surface);
  box-shadow: var(--ssxz-shadow);
}

.usage-summary-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.85rem;
  align-items: center;
  border-radius: 1.25rem;
  padding: 1rem;
}

.summary-icon {
  display: grid;
  width: 2.45rem;
  height: 2.45rem;
  place-items: center;
  border-radius: 0.85rem;
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

.summary-action,
.refresh-button,
.panel-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  font-size: 0.82rem;
  font-weight: 850;
}

.summary-action {
  min-height: 2.15rem;
  background: var(--ssxz-action);
  color: var(--ssxz-action-text);
  padding: 0 0.86rem;
}

.usage-panel {
  overflow: hidden;
  border-radius: 1.25rem;
}

.usage-explainer {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(18rem, 1.4fr);
  gap: 1rem;
  align-items: start;
  border-radius: 1.25rem;
  padding: 1rem;
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
  gap: 0.35rem;
  min-height: 2.1rem;
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
  padding: 0 0.72rem;
}

.refresh-button:hover:not(:disabled) {
  border-color: var(--ssxz-border-strong);
  color: var(--ssxz-text-primary);
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
  width: 100%;
  border-radius: 0.9rem 0.9rem 0 0;
  background: linear-gradient(180deg, var(--ssxz-accent), var(--ssxz-action));
  box-shadow: 0 -8px 24px color-mix(in srgb, var(--ssxz-action) 28%, transparent);
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
  padding: 1.4rem;
  text-align: center;
}

.usage-empty.compact {
  min-height: 10rem;
}

.usage-empty svg {
  color: var(--ssxz-action);
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
  min-width: 46rem;
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

.billing-cell {
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
}
</style>
