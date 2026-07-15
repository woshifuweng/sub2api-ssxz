<template>
  <FoundationCard class="progress-metric-card" :data-testid="testId">
    <div class="progress-metric-card__header">
      <span class="progress-metric-card__icon" aria-hidden="true">
        <component :is="icon" />
      </span>
      <span class="progress-metric-card__period">{{ periodLabel }}</span>
    </div>

    <div class="progress-metric-card__body">
      <p class="progress-metric-card__label">{{ label }}</p>
      <p class="progress-metric-card__value" :title="valueTitle">{{ value }}</p>
      <div class="progress-metric-card__details">
        <slot />
      </div>
    </div>

    <div v-if="hasSeries" class="progress-metric-card__trend" aria-hidden="true">
      <p v-if="trendLabel" class="progress-metric-card__trend-label">{{ trendLabel }}</p>
      <div class="progress-metric-card__chart">
        <Line :data="chartData" :options="chartOptions" />
      </div>
      <dl class="progress-metric-card__summary">
        <div>
          <dt>低位</dt>
          <dd>{{ formatSeriesValue(summary.low) }}</dd>
        </div>
        <div>
          <dt>均值</dt>
          <dd>{{ formatSeriesValue(summary.average) }}</dd>
        </div>
        <div>
          <dt>峰值</dt>
          <dd>{{ formatSeriesValue(summary.peak) }}</dd>
        </div>
      </dl>
    </div>
    <div v-else-if="$slots.support" class="progress-metric-card__support">
      <slot name="support" />
    </div>
  </FoundationCard>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
  type ChartData,
  type ChartOptions
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { FoundationCard } from '@/components/foundation'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Filler)

const props = withDefaults(defineProps<{
  testId: string
  label: string
  value: string
  icon: Component
  periodLabel?: string
  valueTitle?: string
  trendLabel?: string
  series?: number[]
  seriesFormatter?: (value: number) => string
}>(), {
  periodLabel: '当前',
  valueTitle: undefined,
  trendLabel: '',
  series: () => [],
  seriesFormatter: undefined
})

const normalizedSeries = computed(() => props.series.filter((value) => Number.isFinite(value)))
const hasSeries = computed(() => normalizedSeries.value.length > 1)

const summary = computed(() => {
  if (!hasSeries.value) return { low: 0, average: 0, peak: 0 }
  const values = normalizedSeries.value
  return {
    low: Math.min(...values),
    average: values.reduce((sum, value) => sum + value, 0) / values.length,
    peak: Math.max(...values)
  }
})

const chartData = computed<ChartData<'line'>>(() => ({
  labels: normalizedSeries.value.map((_, index) => String(index + 1)),
  datasets: [
    {
      data: normalizedSeries.value,
      borderColor: '#718096',
      backgroundColor: 'rgba(113, 128, 150, 0.12)',
      borderWidth: 1.75,
      pointRadius: 0,
      pointHoverRadius: 0,
      fill: true,
      tension: 0.35
    }
  ]
}))

const chartOptions: ChartOptions<'line'> = {
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  events: [],
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false }
  },
  scales: {
    x: { display: false },
    y: { display: false, beginAtZero: true }
  },
  layout: {
    padding: 1
  }
}

function formatSeriesValue(value: number): string {
  if (props.seriesFormatter) return props.seriesFormatter(value)
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 1 }).format(value)
}
</script>

<style scoped>
.progress-metric-card {
  min-height: 13.25rem;
}

.progress-metric-card :deep(.f0-card-content) {
  display: flex;
  min-height: 13.25rem;
  flex-direction: column;
  padding: 1rem;
}

.progress-metric-card__header,
.progress-metric-card__summary,
.progress-metric-card__details {
  display: flex;
  align-items: center;
}

.progress-metric-card__header {
  justify-content: space-between;
  gap: 0.75rem;
}

.progress-metric-card__icon {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
}

.progress-metric-card__icon :deep(svg) {
  width: 1rem;
  height: 1rem;
  stroke-width: 1.8;
}

.progress-metric-card__period,
.progress-metric-card__label,
.progress-metric-card__details,
.progress-metric-card__summary dt {
  color: hsl(var(--muted-foreground));
}

.progress-metric-card__period {
  font-size: 0.6875rem;
  font-weight: 600;
}

.progress-metric-card__body {
  margin-top: 1rem;
}

.progress-metric-card__label,
.progress-metric-card__value {
  margin: 0;
}

.progress-metric-card__label {
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.125rem;
}

.progress-metric-card__value {
  margin-top: 0.25rem;
  color: hsl(var(--foreground));
  font-size: clamp(1.45rem, 2vw, 1.8rem);
  font-weight: 700;
  line-height: 2.1rem;
}

.progress-metric-card__details {
  min-height: 1.25rem;
  flex-wrap: wrap;
  gap: 0.25rem 0.65rem;
  margin-top: 0.3rem;
  font-size: 0.6875rem;
  line-height: 1rem;
}

.progress-metric-card__trend {
  margin-top: auto;
  padding-top: 0.8rem;
}

.progress-metric-card__trend-label {
  margin: 0 0 0.25rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.625rem;
  font-weight: 600;
  line-height: 0.9rem;
}

.progress-metric-card__chart {
  height: 2.75rem;
}

.progress-metric-card__summary {
  justify-content: space-between;
  gap: 0.5rem;
  margin: 0.35rem 0 0;
}

.progress-metric-card__summary div {
  min-width: 0;
}

.progress-metric-card__summary dt,
.progress-metric-card__summary dd {
  margin: 0;
  font-size: 0.625rem;
  line-height: 0.9rem;
}

.progress-metric-card__support {
  margin-top: auto;
  padding-top: 0.8rem;
}

.progress-metric-card__summary dd {
  color: hsl(var(--foreground));
  font-weight: 600;
}

@media (max-width: 640px) {
  .progress-metric-card,
  .progress-metric-card :deep(.f0-card-content) {
    min-height: 12rem;
  }
}
</style>
