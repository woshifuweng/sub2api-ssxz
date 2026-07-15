<template>
  <section
    :class="[
      'token-usage-trend',
      variant === 'foundation' ? 'token-usage-trend--foundation' : 'card p-4'
    ]"
  >
    <header :class="{ 'token-usage-trend__header': variant === 'foundation' }">
      <div>
        <p v-if="variant === 'foundation'">Token 结构</p>
        <h3>{{ t('admin.dashboard.tokenUsageTrend') }}</h3>
      </div>
      <span v-if="variant === 'foundation'">{{ trendData.length }} 个数据点</span>
    </header>

    <div v-if="loading" class="token-usage-trend__state">
      <LoadingSpinner />
    </div>
    <div v-else-if="trendData.length > 0 && chartData" class="token-usage-trend__chart">
      <Line :data="chartData" :options="lineOptions" />
    </div>
    <div v-else class="token-usage-trend__state token-usage-trend__state--empty">
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip,
  type ChartData,
  type ChartOptions
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { TrendDataPoint } from '@/types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const { t } = useI18n()

const props = withDefaults(defineProps<{
  trendData: TrendDataPoint[]
  loading?: boolean
  theme?: 'light' | 'dark'
  variant?: 'default' | 'foundation'
}>(), {
  loading: false,
  theme: undefined,
  variant: 'default'
})

const isDarkMode = computed(() => {
  if (props.theme) return props.theme === 'dark'
  return typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
})

const chartColors = computed(() => {
  if (props.variant === 'foundation') {
    return isDarkMode.value
      ? {
          text: '#a8b0bd',
          grid: 'rgba(148, 163, 184, 0.16)',
          input: '#e2e8f0',
          output: '#94a3b8',
          cacheCreation: '#64748b',
          cacheRead: '#475569'
        }
      : {
          text: '#667085',
          grid: 'rgba(100, 116, 139, 0.14)',
          input: '#1f2937',
          output: '#475569',
          cacheCreation: '#718096',
          cacheRead: '#a0aec0'
        }
  }

  return {
    text: isDarkMode.value ? '#e5e7eb' : '#374151',
    grid: isDarkMode.value ? '#374151' : '#e5e7eb',
    input: '#f2b84b',
    output: '#d99b3d',
    cacheCreation: '#f59e0b',
    cacheRead: '#78716c'
  }
})

const chartData = computed<ChartData<'line'> | null>(() => {
  if (!props.trendData?.length) return null

  return {
    labels: props.trendData.map((point) => point.date),
    datasets: [
      createDataset('Input', props.trendData.map((point) => point.input_tokens), chartColors.value.input),
      createDataset('Output', props.trendData.map((point) => point.output_tokens), chartColors.value.output),
      createDataset(
        'Cache Creation',
        props.trendData.map((point) => point.cache_creation_tokens),
        chartColors.value.cacheCreation
      ),
      createDataset(
        'Cache Read',
        props.trendData.map((point) => point.cache_read_tokens),
        chartColors.value.cacheRead
      )
    ]
  }
})

const lineOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: {
    intersect: false,
    mode: 'index'
  },
  plugins: {
    legend: {
      position: 'top',
      align: 'start',
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        boxWidth: 7,
        boxHeight: 7,
        font: { size: 10 }
      }
    },
    tooltip: {
      callbacks: {
        label: (context) => `${context.dataset.label}: ${formatTokens(Number(context.raw))}`,
        footer: (tooltipItems) => {
          const dataIndex = tooltipItems[0]?.dataIndex
          if (dataIndex === undefined || !props.trendData[dataIndex]) return ''
          const point = props.trendData[dataIndex]
          return `Actual: $${formatCost(point.actual_cost)} | Standard: $${formatCost(point.cost)}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: { color: chartColors.value.grid },
      ticks: { color: chartColors.value.text, font: { size: 10 }, maxRotation: 0 }
    },
    y: {
      beginAtZero: true,
      grid: { color: chartColors.value.grid },
      ticks: {
        color: chartColors.value.text,
        font: { size: 10 },
        callback: (value) => formatTokens(Number(value))
      }
    }
  }
}))

function createDataset(label: string, data: number[], color: string) {
  return {
    label,
    data,
    borderColor: color,
    backgroundColor: `${color}14`,
    borderWidth: 1.75,
    pointRadius: 0,
    pointHoverRadius: 3,
    fill: true,
    tension: 0.3
  }
}

function formatTokens(value: number): string {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}

function formatCost(value: number): string {
  if (value >= 1000) return `${(value / 1000).toFixed(2)}K`
  if (value >= 1) return value.toFixed(2)
  if (value >= 0.01) return value.toFixed(3)
  return value.toFixed(4)
}
</script>

<style scoped>
.token-usage-trend > header h3 {
  margin: 0 0 1rem;
  font-size: 0.875rem;
  font-weight: 650;
  line-height: 1.25rem;
}

.token-usage-trend__chart,
.token-usage-trend__state {
  height: 12rem;
}

.token-usage-trend__state {
  display: flex;
  align-items: center;
  justify-content: center;
}

.token-usage-trend__state--empty {
  color: #6b7280;
  font-size: 0.75rem;
}

.token-usage-trend--foundation {
  min-width: 0;
  min-height: 20rem;
  padding: 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  box-shadow: 0 1px 2px hsl(var(--shadow));
}

.token-usage-trend--foundation .token-usage-trend__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid hsl(var(--border));
}

.token-usage-trend--foundation .token-usage-trend__header p,
.token-usage-trend--foundation .token-usage-trend__header h3 {
  margin: 0;
}

.token-usage-trend--foundation .token-usage-trend__header p,
.token-usage-trend--foundation .token-usage-trend__header > span {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1rem;
}

.token-usage-trend--foundation .token-usage-trend__header h3 {
  margin-top: 0.1rem;
}

.token-usage-trend--foundation .token-usage-trend__chart,
.token-usage-trend--foundation .token-usage-trend__state {
  height: 15rem;
  padding-top: 0.75rem;
}
</style>
