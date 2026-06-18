<template>
  <AppSectionShell
    title="用量信息"
    subtitle="查看账户余额、本月消耗和最近使用明细。没有真实用量数据时只显示空状态，不跳转到旧版用量页。"
    eyebrow="账户计量"
    icon="chartBar"
  >
    <section class="usage-workbench" aria-label="用量信息">
      <div class="usage-summary-grid">
        <article class="usage-summary-card">
          <div class="summary-icon">
            <Icon name="creditCard" size="sm" />
          </div>
          <div>
            <span>账户余额</span>
            <strong>{{ balanceText }}</strong>
            <p>可用于站内聊天、图片生成和 API Key / 第三方接入调用。</p>
          </div>
          <RouterLink to="/purchase" class="summary-action">充值</RouterLink>
        </article>

        <article class="usage-summary-card">
          <div class="summary-icon">
            <Icon name="chartBar" size="sm" />
          </div>
          <div>
            <span>本月消耗</span>
            <strong>{{ monthlyCostText }}</strong>
            <p>{{ monthlyUsageNote }}</p>
          </div>
        </article>
      </div>

      <section class="usage-panel">
        <header class="panel-heading">
          <div>
            <h3>每月用量</h3>
            <p>按现有用量趋势接口汇总展示，没有数据时不会补假柱子。</p>
          </div>
          <span v-if="hasMonthlyUsage" class="panel-badge">真实数据</span>
        </header>

        <div v-if="hasMonthlyUsage" class="usage-chart" aria-label="每月用量图表">
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
          <strong>暂无月度用量数据</strong>
          <span>后续产生聊天、图片生成或第三方客户端调用后，这里会展示趋势。</span>
        </div>
      </section>

      <section class="usage-panel">
        <header class="panel-heading">
          <div>
            <h3>用量明细</h3>
            <p>展示最近的真实调用记录，方便核对模型、类型、用量和扣费。</p>
          </div>
          <button type="button" class="refresh-button" :disabled="loading" @click="loadUsageOverview">
            <Icon name="refresh" size="xs" />
            刷新
          </button>
        </header>

        <div v-if="loading" class="usage-empty compact">
          <Icon name="sync" size="md" />
          <strong>正在加载用量</strong>
        </div>

        <div v-else-if="loadError" class="usage-empty compact">
          <Icon name="exclamationTriangle" size="md" />
          <strong>{{ loadError }}</strong>
          <span>可以稍后刷新，不会自动跳到旧版页面。</span>
        </div>

        <div v-else-if="usageRows.length === 0" class="usage-empty compact">
          <Icon name="inbox" size="md" />
          <strong>暂无用量明细</strong>
          <span>真实调用产生后会在这里显示。</span>
        </div>

        <div v-else class="usage-table-wrap">
          <table class="usage-table">
            <thead>
              <tr>
                <th>创建时间</th>
                <th>类型</th>
                <th>模型</th>
                <th>用量</th>
                <th>扣费</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in usageRows" :key="row.id || row.request_id">
                <td>{{ formatDateTime(row.created_at) }}</td>
                <td>{{ formatUsageKind(row) }}</td>
                <td class="model-cell">{{ row.model || '-' }}</td>
                <td>{{ formatUsageAmount(row) }}</td>
                <td>{{ formatCost(row.actual_cost) }}</td>
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

const authStore = useAuthStore()
const usageRows = ref<UsageLog[]>([])
const usageStats = ref<UsageStatsResponse | null>(null)
const monthlySeries = ref<MonthlyUsage[]>([])
const loading = ref(false)
const loadError = ref('')

const today = new Date()
const monthStart = new Date(today.getFullYear(), today.getMonth(), 1)
const trendStart = new Date(today.getFullYear(), today.getMonth() - 5, 1)

const todayKey = toDateKey(today)
const monthStartKey = toDateKey(monthStart)
const trendStartKey = toDateKey(trendStart)

const balanceText = computed(() => formatCurrency(authStore.user?.balance || 0, 2))
const monthlyCostText = computed(() => formatCurrency(usageStats.value?.total_actual_cost || 0, 4))
const monthlyUsageNote = computed(() => {
  const requests = usageStats.value?.total_requests || 0
  const tokens = usageStats.value?.total_tokens || 0
  if (!requests && !tokens) return '本月暂未产生真实用量记录。'
  return `本月 ${formatNumber(requests)} 次请求，${formatNumber(tokens)} tokens。`
})
const hasMonthlyUsage = computed(() => monthlySeries.value.some((item) => item.requests > 0 || item.tokens > 0 || item.cost > 0))
const chartMax = computed(() => Math.max(1, ...monthlySeries.value.map((item) => chartMetric(item))))

onMounted(() => {
  void loadUsageOverview()
})

async function loadUsageOverview() {
  loading.value = true
  loadError.value = ''

  const [statsResult, logsResult, trendResult] = await Promise.allSettled([
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
    })
  ])

  if (statsResult.status === 'fulfilled') {
    usageStats.value = statsResult.value
  }

  if (logsResult.status === 'fulfilled') {
    usageRows.value = Array.isArray(logsResult.value.items) ? logsResult.value.items : []
  } else {
    usageRows.value = []
  }

  if (trendResult.status === 'fulfilled') {
    monthlySeries.value = buildMonthlySeries(trendResult.value.trend || [])
  } else {
    monthlySeries.value = []
  }

  if (logsResult.status === 'rejected') {
    loadError.value = '用量明细暂时无法加载'
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
      label: `${date.getMonth() + 1}月`,
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
  if (item.tokens > 0) return `${formatNumber(item.tokens)} tokens`
  return `${formatNumber(item.requests)} 次`
}

function formatUsageKind(row: UsageLog) {
  if (row.image_count > 0 || row.inbound_endpoint?.includes('/images/')) return '图片生成'
  if (row.inbound_endpoint?.includes('/chat/')) return '对话'
  if (row.api_key_id) return '第三方接入'
  return '网页端'
}

function formatUsageAmount(row: UsageLog) {
  if (row.image_count > 0) {
    const size = row.image_size ? ` · ${row.image_size}` : ''
    return `${formatNumber(row.image_count)} 张${size}`
  }
  const tokens = Number(row.input_tokens || 0) + Number(row.output_tokens || 0) + Number(row.cache_creation_tokens || 0) + Number(row.cache_read_tokens || 0)
  return `${formatNumber(tokens)} tokens`
}

function formatCost(value: number | null | undefined) {
  const cost = Number(value || 0)
  return formatCurrency(cost, cost > 0 && cost < 0.01 ? 6 : 4)
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
.usage-panel {
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

@media (max-width: 860px) {
  .usage-summary-grid {
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
