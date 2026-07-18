<template>
  <section class="dashboard-metrics" aria-label="账户与用量指标">
    <ProgressMetricCard
      v-if="!isSimple"
      test-id="metric-card-balance"
      :label="t('dashboard.balance')"
      :value="formatCurrency(balance)"
      :value-title="formatCurrencyTitle(balance)"
      :icon="Wallet"
      period-label="实时"
    >
      <span>有效 Key {{ stats.active_api_keys || 0 }} / {{ stats.total_api_keys || 0 }}</span>
      <span>{{ t('common.available') }}</span>
      <template #support>
        <div class="dashboard-balance-support">
          <div class="dashboard-balance-support__heading">
            <span>Key 可用率</span>
            <strong>{{ activeKeyRate }}%</strong>
          </div>
          <div class="dashboard-balance-support__track" aria-hidden="true">
            <span :style="{ width: `${activeKeyRate}%` }" />
          </div>
        </div>
      </template>
    </ProgressMetricCard>

    <ProgressMetricCard
      test-id="metric-card-total-requests"
      label="累计请求"
      :value="formatNumber(stats.total_requests || 0)"
      :icon="ChartSpline"
      period-label="累计"
      :series="requestSeries"
      :series-formatter="formatCompact"
      trend-label="所选周期请求趋势"
    >
      <span>全时累计</span>
    </ProgressMetricCard>

    <ProgressMetricCard
      test-id="metric-card-today-requests"
      :label="t('dashboard.todayRequests')"
      :value="formatNumber(stats.today_requests || 0)"
      :icon="Activity"
      period-label="今日"
      :series="todayRequestSeries"
      :series-formatter="formatCompact"
      trend-label="今日按小时请求趋势"
    >
      <span>仅统计今日</span>
    </ProgressMetricCard>

    <ProgressMetricCard
      test-id="metric-card-today-cost"
      :label="`${t('dashboard.todayCost')} · ${t('dashboard.actual')}`"
      :value="formatCurrency(stats.today_actual_cost || 0)"
      :value-title="formatCurrencyTitle(stats.today_actual_cost || 0)"
      :icon="CircleDollarSign"
      period-label="今日"
      :series="todayCostSeries"
      :series-formatter="formatCostSeries"
      trend-label="今日按小时实际消费"
    >
      <span :title="formatCurrencyTitle(stats.today_cost || 0)">{{ t('dashboard.todayCost') }} · {{ t('dashboard.standard') }} {{ formatCurrency(stats.today_cost || 0) }}</span>
      <span :title="`${formatCurrencyTitle(stats.total_actual_cost || 0)}；标准 ${formatCurrencyExact(stats.total_cost || 0)}`">{{ t('common.total') }} · {{ t('dashboard.actual') }} {{ formatCurrency(stats.total_actual_cost || 0) }} / {{ t('common.total') }} · {{ t('dashboard.standard') }} {{ formatCurrency(stats.total_cost || 0) }}</span>
    </ProgressMetricCard>

    <ProgressMetricCard
      test-id="metric-card-today-tokens"
      :label="t('dashboard.todayTokens')"
      :value="formatTokens(stats.today_tokens || 0)"
      :icon="Coins"
      period-label="今日"
      :series="todayTokenSeries"
      :series-formatter="formatCompact"
      trend-label="今日按小时 Token 趋势"
    >
      <span>{{ t('dashboard.input') }} {{ formatTokens(stats.today_input_tokens || 0) }}</span>
      <span>{{ t('dashboard.output') }} {{ formatTokens(stats.today_output_tokens || 0) }}</span>
    </ProgressMetricCard>

    <ProgressMetricCard
      test-id="metric-card-total-tokens"
      :label="t('dashboard.totalTokens')"
      :value="formatTokens(stats.total_tokens || 0)"
      :icon="DatabaseZap"
      period-label="全部"
      :series="tokenSeries"
      :series-formatter="formatCompact"
      trend-label="所选周期 Token 趋势"
    >
      <span>{{ t('dashboard.input') }} {{ formatTokens(stats.total_input_tokens || 0) }}</span>
      <span>{{ t('dashboard.output') }} {{ formatTokens(stats.total_output_tokens || 0) }}</span>
    </ProgressMetricCard>

    <ProgressMetricCard
      test-id="metric-card-performance"
      :label="t('dashboard.performance')"
      :value="`${formatTokens(stats.rpm || 0)} RPM`"
      :icon="Gauge"
      period-label="近 5 分钟"
    >
      <span>{{ formatTokens(stats.tpm || 0) }} TPM</span>
      <span>分钟平均</span>
    </ProgressMetricCard>

    <ProgressMetricCard
      test-id="metric-card-average-duration"
      :label="t('dashboard.avgResponse')"
      :value="formatDuration(stats.average_duration_ms || 0)"
      :icon="Timer"
      period-label="全部"
    >
      <span>{{ t('dashboard.averageTime') }}</span>
      <span>服务端处理耗时</span>
    </ProgressMetricCard>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Activity,
  ChartSpline,
  CircleDollarSign,
  Coins,
  DatabaseZap,
  Gauge,
  Timer,
  Wallet
} from '@lucide/vue'
import ProgressMetricCard from './ProgressMetricCard.vue'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'
import type { TrendDataPoint } from '@/types'
import { formatCurrency, formatCurrencyExact, formatCurrencyTitle } from '@/utils/format'

const props = withDefaults(defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  trend?: TrendDataPoint[]
  todayTrend?: TrendDataPoint[]
}>(), {
  trend: () => [],
  todayTrend: () => []
})

const { t } = useI18n()

const requestSeries = computed(() => props.trend.map((point) => point.requests || 0))
const tokenSeries = computed(() => props.trend.map((point) => point.total_tokens || 0))
const todayRequestSeries = computed(() => props.todayTrend.map((point) => point.requests || 0))
const todayCostSeries = computed(() => props.todayTrend.map((point) => point.actual_cost || 0))
const todayTokenSeries = computed(() => props.todayTrend.map((point) => point.total_tokens || 0))
const activeKeyRate = computed(() => {
  const total = props.stats.total_api_keys || 0
  if (total <= 0) return 0
  return Math.min(100, Math.round(((props.stats.active_api_keys || 0) / total) * 100))
})

const formatNumber = (value: number) => value.toLocaleString()
const formatCostSeries = (value: number) => formatCurrency(value)
const formatCompact = (value: number) =>
  new Intl.NumberFormat('en-US', {
    notation: 'compact',
    maximumFractionDigits: 1
  }).format(value)

const formatTokens = (value: number) => {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return value.toString()
}

const formatDuration = (value: number) =>
  value >= 1000 ? `${(value / 1000).toFixed(2)}s` : `${value.toFixed(0)}ms`
</script>

<style scoped>
.dashboard-metrics {
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  align-items: stretch;
  gap: 0.75rem;
}

.dashboard-metrics > * {
  height: 100%;
}

.dashboard-balance-support {
  display: grid;
  gap: 0.45rem;
}

.dashboard-balance-support__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.625rem;
  line-height: 0.9rem;
}

.dashboard-balance-support__heading strong {
  color: hsl(var(--foreground));
  font-size: 0.6875rem;
}

.dashboard-balance-support__track {
  height: 0.35rem;
  overflow: hidden;
  border-radius: 999px;
  background: hsl(var(--muted));
}

.dashboard-balance-support__track span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: hsl(var(--foreground));
}

@media (max-width: 1600px) {
  .dashboard-metrics {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .dashboard-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

</style>
