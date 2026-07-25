<template>
  <component
    :is="to ? RouterLink : 'div'"
    v-bind="to ? { to } : {}"
    :class="[
      'progress-metric-card',
      {
        'progress-metric-card--interactive': to,
        'progress-metric-card--detailed': detailed
      }
    ]"
    :data-testid="testId"
    :aria-label="to ? `查看${label}` : undefined"
  >
    <FoundationCard class="progress-metric-card__surface">
      <div class="progress-metric-card__header">
        <div class="progress-metric-card__identity">
          <span class="progress-metric-card__icon" aria-hidden="true">
            <component :is="icon" />
          </span>
          <p v-if="detailed" class="progress-metric-card__label">{{ label }}</p>
        </div>

        <div v-if="detailed" class="progress-metric-card__controls">
          <label v-if="periodOptions.length" class="progress-metric-card__period-select">
            <span class="sr-only">{{ periodSelectLabel }}</span>
            <select v-model="selectedPeriod" data-testid="metric-period-select">
              <option v-for="option in periodOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </label>
          <div class="progress-metric-card__view-toggle" role="group" :aria-label="viewToggleLabel">
            <button
              type="button"
              :class="{ 'is-active': selectedView === 'curve' }"
              data-testid="metric-view-curve"
              :aria-label="curveViewLabel"
              :title="curveViewLabel"
              @click="selectedView = 'curve'"
            >
              <ChartSpline aria-hidden="true" />
            </button>
            <button
              type="button"
              :class="{ 'is-active': selectedView === 'bar' }"
              data-testid="metric-view-bar"
              :aria-label="barViewLabel"
              :title="barViewLabel"
              @click="selectedView = 'bar'"
            >
              <ChartColumn aria-hidden="true" />
            </button>
          </div>
        </div>
        <span v-else class="progress-metric-card__period">{{ periodLabel }}</span>
      </div>

      <div class="progress-metric-card__body">
        <p v-if="!detailed" class="progress-metric-card__label">{{ label }}</p>
        <p class="progress-metric-card__value" :title="valueTitle">{{ displayValue }}</p>
        <div class="progress-metric-card__details">
          <slot />
        </div>
      </div>

      <div
        v-if="hasSeries"
        class="progress-metric-card__trend"
        :aria-label="trendLabel || `${label}趋势`"
      >
        <div class="progress-metric-card__chart">
          <Line
            v-if="selectedView === 'curve'"
            :data="lineChartData"
            :options="lineChartOptions"
          />
          <Bar
            v-else
            :data="barChartData"
            :options="barChartOptions"
          />
        </div>
        <dl
          v-if="showStats"
          :class="[
            'progress-metric-card__summary',
            { 'progress-metric-card__summary--sr-only': !detailed }
          ]"
        >
          <div>
            <dt>{{ lowLabel }}</dt>
            <dd>{{ formatSeriesValue(summary.low) }}</dd>
          </div>
          <div>
            <dt>{{ averageLabel }}</dt>
            <dd>{{ formatSeriesValue(summary.average) }}</dd>
          </div>
          <div>
            <dt>{{ peakLabel }}</dt>
            <dd>{{ formatSeriesValue(summary.peak) }}</dd>
          </div>
        </dl>
      </div>
      <div v-else-if="$slots.support" class="progress-metric-card__support">
        <slot name="support" />
      </div>
    </FoundationCard>
  </component>
</template>

<script setup lang="ts">
import { computed, ref, type Component } from 'vue'
import { RouterLink } from 'vue-router'
import { ChartColumn, ChartSpline } from '@lucide/vue'
import {
  BarElement,
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
import { Bar, Line } from 'vue-chartjs'
import { FoundationCard } from '@/components/foundation'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, BarElement, Tooltip, Filler)

type MetricView = 'curve' | 'bar'

interface MetricPeriodOption {
  value: string
  label: string
  points?: number
}

const props = withDefaults(defineProps<{
  testId: string
  label: string
  value?: string
  icon: Component
  periodLabel?: string
  valueTitle?: string
  trendLabel?: string
  series?: number[]
  seriesLabels?: string[]
  seriesFormatter?: (value: number) => string
  detailed?: boolean
  periodOptions?: MetricPeriodOption[]
  defaultPeriod?: string
  defaultView?: MetricView
  showStats?: boolean
  periodSelectLabel?: string
  viewToggleLabel?: string
  curveViewLabel?: string
  barViewLabel?: string
  lowLabel?: string
  averageLabel?: string
  peakLabel?: string
  to?: string
}>(), {
  value: undefined,
  periodLabel: '当前',
  valueTitle: undefined,
  trendLabel: '',
  series: () => [],
  seriesLabels: () => [],
  seriesFormatter: undefined,
  detailed: false,
  periodOptions: () => [],
  defaultPeriod: '',
  defaultView: 'curve',
  showStats: true,
  periodSelectLabel: '选择统计周期',
  viewToggleLabel: '切换图表样式',
  curveViewLabel: '曲线图',
  barViewLabel: '柱状图',
  lowLabel: '低位',
  averageLabel: '均值',
  peakLabel: '峰值',
  to: undefined
})

const selectedPeriod = ref(
  props.defaultPeriod || props.periodOptions[0]?.value || ''
)
const selectedView = ref<MetricView>(props.defaultView)

const normalizedPoints = computed(() => props.series
  .map((value, index) => ({
    value,
    label: props.seriesLabels[index] || String(index + 1)
  }))
  .filter((point) => Number.isFinite(point.value)))

const selectedPeriodOption = computed(() => (
  props.periodOptions.find((option) => option.value === selectedPeriod.value)
    || props.periodOptions[props.periodOptions.length - 1]
))

const visiblePoints = computed(() => {
  const points = normalizedPoints.value
  const count = selectedPeriodOption.value?.points
  return count && count < points.length ? points.slice(-count) : points
})

const normalizedSeries = computed(() => visiblePoints.value.map((point) => point.value))
const visibleLabels = computed(() => visiblePoints.value.map((point) => point.label))
const hasSeries = computed(() => (
  props.detailed ? normalizedSeries.value.length > 0 : normalizedSeries.value.length > 1
))

const summary = computed(() => {
  if (!hasSeries.value) return { low: 0, average: 0, peak: 0 }
  const values = normalizedSeries.value
  return {
    low: Math.min(...values),
    average: values.reduce((sum, value) => sum + value, 0) / values.length,
    peak: Math.max(...values)
  }
})

const displayValue = computed(() => {
  if (props.value !== undefined) return props.value
  return formatSeriesValue(normalizedSeries.value.reduce((sum, value) => sum + value, 0))
})

const lineChartData = computed<ChartData<'line'>>(() => ({
  labels: visibleLabels.value,
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

const barChartData = computed<ChartData<'bar'>>(() => ({
  labels: visibleLabels.value,
  datasets: [
    {
      data: normalizedSeries.value,
      backgroundColor: 'rgba(143, 170, 196, 0.48)',
      borderColor: '#8faac4',
      borderWidth: 1,
      borderRadius: 4,
      maxBarThickness: 18
    }
  ]
}))

const compactChartOptions: ChartOptions<'line'> = {
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

const lineChartOptions = computed<ChartOptions<'line'>>(() => (
  props.detailed
    ? {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        interaction: { intersect: false, mode: 'index' },
        plugins: {
          legend: { display: false },
          tooltip: {
            enabled: true,
            displayColors: false,
            callbacks: {
              label: (context) => formatSeriesValue(Number(context.parsed.y || 0))
            }
          }
        },
        scales: {
          x: { display: false },
          y: { display: false, beginAtZero: true }
        },
        layout: { padding: 2 }
      }
    : compactChartOptions
))

const barChartOptions: ChartOptions<'bar'> = {
  responsive: true,
  maintainAspectRatio: false,
  animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: { display: false },
    tooltip: {
      enabled: true,
      displayColors: false,
      callbacks: {
        label: (context) => formatSeriesValue(Number(context.parsed.y || 0))
      }
    }
  },
  scales: {
    x: { display: false },
    y: { display: false, beginAtZero: true }
  },
  layout: { padding: 2 }
}

function formatSeriesValue(value: number): string {
  if (props.seriesFormatter) return props.seriesFormatter(value)
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 1 }).format(value)
}
</script>

<style scoped>
.progress-metric-card {
  display: block;
  min-width: 0;
  border-radius: var(--radius);
  color: inherit;
  text-decoration: none;
}

.progress-metric-card--interactive {
  cursor: pointer;
}

.progress-metric-card--interactive :deep(.f0-card) {
  transition: border-color 150ms ease, box-shadow 150ms ease, transform 120ms ease;
}

.progress-metric-card--interactive:hover :deep(.f0-card) {
  border-color: hsl(var(--foreground) / 0.28);
  box-shadow: var(--ssxz-shadow-card);
  transform: translateY(-1px);
}

.progress-metric-card--interactive:active :deep(.f0-card) {
  transform: translateY(0);
}

.progress-metric-card--interactive:focus-visible {
  outline: 2px solid hsl(var(--ring));
  outline-offset: 2px;
}

.progress-metric-card :deep(.f0-card-content) {
  display: flex;
  flex-direction: column;
  padding: 0.8rem;
}

.progress-metric-card__header,
.progress-metric-card__identity,
.progress-metric-card__controls,
.progress-metric-card__view-toggle,
.progress-metric-card__summary,
.progress-metric-card__details {
  display: flex;
  align-items: center;
}

.progress-metric-card__header {
  justify-content: space-between;
  gap: 0.75rem;
}

.progress-metric-card__identity {
  min-width: 0;
  gap: 0.65rem;
}

.progress-metric-card__icon {
  display: inline-flex;
  width: 1.75rem;
  height: 1.75rem;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
}

.progress-metric-card__icon :deep(svg) {
  width: 0.9rem;
  height: 0.9rem;
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

.progress-metric-card__controls {
  flex: 0 0 auto;
  gap: 0.45rem;
}

.progress-metric-card__period-select select,
.progress-metric-card__view-toggle {
  height: 2rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-button);
  background: transparent;
  color: var(--ssxz-text-secondary);
}

.progress-metric-card__period-select select {
  min-width: 7.5rem;
  cursor: pointer;
  padding: 0 1.8rem 0 0.65rem;
  font-size: 0.72rem;
  font-weight: 700;
}

.progress-metric-card__period-select select:focus-visible,
.progress-metric-card__view-toggle button:focus-visible {
  outline: 2px solid var(--ssxz-action);
  outline-offset: 2px;
}

.progress-metric-card__view-toggle {
  overflow: hidden;
}

.progress-metric-card__view-toggle button {
  display: grid;
  width: 2rem;
  height: 2rem;
  place-items: center;
  border: 0;
  background: transparent;
  color: var(--ssxz-text-muted);
  cursor: pointer;
}

.progress-metric-card__view-toggle button + button {
  border-left: 1px solid var(--ssxz-border);
}

.progress-metric-card__view-toggle button:hover,
.progress-metric-card__view-toggle button.is-active {
  background: var(--ssxz-action-soft);
  color: var(--ssxz-action);
}

.progress-metric-card__view-toggle svg {
  width: 0.9rem;
  height: 0.9rem;
  stroke-width: 1.8;
}

.progress-metric-card__body {
  margin-top: 0.65rem;
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
  margin-top: 0.15rem;
  color: hsl(var(--foreground));
  font-size: clamp(1.25rem, 1.55vw, 1.6rem);
  font-weight: 700;
  line-height: 1.8rem;
}

.progress-metric-card__details {
  display: grid;
  min-width: 0;
  gap: 0.15rem;
  margin-top: 0.3rem;
  font-size: 0.6875rem;
  line-height: 1rem;
}

.progress-metric-card__details :deep(> *) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-metric-card__trend {
  margin-top: auto;
  padding-top: 0.5rem;
}

.progress-metric-card__chart {
  position: relative;
  width: 100%;
  min-width: 0;
  height: 2.1rem;
  overflow: hidden;
}

.progress-metric-card__summary {
  justify-content: space-between;
  gap: 0.5rem;
  margin: 0.2rem 0 0;
}

.progress-metric-card__summary--sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  margin: -1px;
  padding: 0;
  border: 0;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
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
  padding-top: 0.5rem;
}

.progress-metric-card__summary dd {
  color: hsl(var(--foreground));
  font-weight: 600;
}

.progress-metric-card--detailed :deep(.f0-card-content) {
  min-height: 17rem;
  padding: 1.15rem;
}

.progress-metric-card--detailed .progress-metric-card__body {
  margin-top: 1rem;
}

.progress-metric-card--detailed .progress-metric-card__value {
  margin-top: 0;
  font-size: clamp(1.75rem, 2.5vw, 2.4rem);
  line-height: 1.08;
}

.progress-metric-card--detailed .progress-metric-card__details {
  min-height: 1rem;
  margin-top: 0.45rem;
}

.progress-metric-card--detailed .progress-metric-card__trend {
  position: relative;
  margin-top: 0.75rem;
  padding-top: 0.75rem;
}

.progress-metric-card--detailed .progress-metric-card__trend::before {
  position: absolute;
  inset: 0 0 2rem;
  border-radius: var(--ssxz-radius-button);
  background-image: radial-gradient(circle, color-mix(in srgb, var(--ssxz-text-muted) 24%, transparent) 0 1px, transparent 1px);
  background-size: 14px 14px;
  content: '';
  mask-image: linear-gradient(to right, transparent, #000 42%, #000);
  pointer-events: none;
}

.progress-metric-card--detailed .progress-metric-card__chart {
  height: 7.5rem;
}

.progress-metric-card--detailed .progress-metric-card__summary {
  position: relative;
  margin-top: 0.7rem;
  padding-top: 0.7rem;
  border-top: 1px solid var(--ssxz-border);
}

.progress-metric-card--detailed .progress-metric-card__summary div {
  display: grid;
  gap: 0.12rem;
}

.progress-metric-card--detailed .progress-metric-card__summary dt,
.progress-metric-card--detailed .progress-metric-card__summary dd {
  font-size: 0.7rem;
  line-height: 1rem;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  margin: -1px;
  padding: 0;
  border: 0;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}

@media (max-width: 560px) {
  .progress-metric-card--detailed .progress-metric-card__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .progress-metric-card--detailed .progress-metric-card__controls {
    width: 100%;
  }

  .progress-metric-card--detailed .progress-metric-card__period-select {
    flex: 1;
  }

  .progress-metric-card--detailed .progress-metric-card__period-select select {
    width: 100%;
  }
}

</style>
