<template>
  <section class="dashboard-analytics" aria-label="用量分析">
    <div class="dashboard-filter-bar">
      <div>
        <p>分析范围</p>
        <strong>按真实用量记录聚合</strong>
      </div>
      <div class="dashboard-filter-bar__controls">
        <label>
          <span>{{ t('dashboard.timeRange') }}</span>
          <DateRangePicker
            :start-date="startDate"
            :end-date="endDate"
            @update:startDate="$emit('update:startDate', $event)"
            @update:endDate="$emit('update:endDate', $event)"
            @change="$emit('dateRangeChange', $event)"
          />
        </label>
        <label class="dashboard-filter-bar__granularity">
          <span>{{ t('dashboard.granularity') }}</span>
          <Select
            :model-value="granularity"
            :options="[
              { value: 'day', label: t('dashboard.day') },
              { value: 'hour', label: t('dashboard.hour') }
            ]"
            @update:model-value="$emit('update:granularity', $event)"
            @change="$emit('granularityChange')"
          />
        </label>
      </div>
    </div>

    <div class="dashboard-chart-grid">
      <div
        class="dashboard-chart-panel dashboard-model-panel"
        :class="{ 'dashboard-chart-panel--empty': !loading && models.length === 0 }"
      >
        <div v-if="loading" class="dashboard-chart-loading">
          <LoadingSpinner size="md" />
        </div>
        <div class="dashboard-chart-panel__heading">
          <div>
            <p>模型结构</p>
            <h3>{{ t('dashboard.modelDistribution') }}</h3>
          </div>
          <span>{{ models.length }} 个模型</span>
        </div>
        <div v-if="models.length > 0" class="dashboard-model-panel__body">
          <div class="dashboard-doughnut">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
          </div>
          <div class="dashboard-model-table-wrap">
            <table class="dashboard-model-table">
              <thead>
                <tr>
                  <th>{{ t('dashboard.model') }}</th>
                  <th>{{ t('dashboard.requests') }}</th>
                  <th>{{ t('dashboard.tokens') }}</th>
                  <th>{{ t('dashboard.actual') }}</th>
                  <th>{{ t('dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="model in models" :key="model.model">
                  <td :title="model.model">
                    <span class="dashboard-model-name">
                      <ModelIcon :model="model.model" size="16px" />
                      {{ model.model }}
                    </span>
                  </td>
                  <td>{{ formatNumber(model.requests) }}</td>
                  <td>{{ formatTokens(model.total_tokens) }}</td>
                  <td :title="formatCurrencyTitle(model.actual_cost)">${{ formatCost(model.actual_cost) }}</td>
                  <td :title="formatCurrencyTitle(model.cost)">${{ formatCost(model.cost) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div v-else class="dashboard-chart-empty">
          <strong>暂无模型用量</strong>
          <span>完成一次模型调用后，这里会显示用量构成。</span>
        </div>
      </div>

      <TokenUsageTrend
        :trend-data="trend"
        :loading="loading"
        :theme="theme"
        variant="foundation"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArcElement,
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
import { Doughnut } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import {
  formatCostFixed as formatCost,
  formatCurrencyTitle,
  formatNumberLocaleString as formatNumber,
  formatTokensK as formatTokens
} from '@/utils/format'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

const props = withDefaults(defineProps<{
  loading: boolean
  startDate: string
  endDate: string
  granularity: string
  trend: TrendDataPoint[]
  models: ModelStat[]
  theme?: 'light' | 'dark'
}>(), {
  theme: 'light'
})

defineEmits([
  'update:startDate',
  'update:endDate',
  'update:granularity',
  'dateRangeChange',
  'granularityChange'
])

const { t } = useI18n()

const chartPalette = computed(() =>
  props.theme === 'dark'
    ? ['#e2e8f0', '#94a3b8', '#64748b', '#475569', '#cbd5e1', '#7c8a9b', '#a8b3c1', '#526071']
    : ['#1f2937', '#475569', '#64748b', '#94a3b8', '#334155', '#718096', '#a0aec0', '#cbd5e1']
)

const modelData = computed<ChartData<'doughnut'> | null>(() => {
  if (!props.models?.length) return null
  return {
    labels: props.models.map((model) => model.model),
    datasets: [
      {
        data: props.models.map((model) => model.total_tokens),
        backgroundColor: props.models.map((_, index) => chartPalette.value[index % chartPalette.value.length]),
        borderWidth: 0,
        spacing: 2
      }
    ]
  }
})

const doughnutOptions = computed<ChartOptions<'doughnut'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '70%',
  animation: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context) => `${context.label}: ${formatTokens(Number(context.parsed))} tokens`
      }
    }
  }
}))
</script>

<style scoped>
.dashboard-analytics {
  display: grid;
  gap: 0.75rem;
}

.dashboard-filter-bar,
.dashboard-chart-panel {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  box-shadow: 0 1px 2px hsl(var(--shadow));
}

.dashboard-filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.8rem 1rem;
}

.dashboard-filter-bar p,
.dashboard-filter-bar strong,
.dashboard-chart-panel__heading p,
.dashboard-chart-panel__heading h3 {
  margin: 0;
}

.dashboard-filter-bar p,
.dashboard-chart-panel__heading p,
.dashboard-filter-bar label > span {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1rem;
}

.dashboard-filter-bar strong {
  display: block;
  margin-top: 0.1rem;
  font-size: 0.8125rem;
  font-weight: 650;
  line-height: 1.2rem;
}

.dashboard-filter-bar__controls,
.dashboard-filter-bar label {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.dashboard-filter-bar__granularity {
  width: 10rem;
}

.dashboard-filter-bar__granularity :deep(> div),
.dashboard-filter-bar__granularity :deep(.relative) {
  flex: 1;
}

.dashboard-chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 0.75rem;
}

.dashboard-chart-panel {
  position: relative;
  min-width: 0;
  min-height: 20rem;
  overflow: hidden;
  padding: 1rem;
}

.dashboard-chart-panel--empty {
  min-height: 10rem;
}

.dashboard-chart-loading {
  position: absolute;
  z-index: 2;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: hsl(var(--card) / 0.9);
}

.dashboard-chart-panel__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid hsl(var(--border));
}

.dashboard-chart-panel__heading h3 {
  margin-top: 0.1rem;
  font-size: 0.875rem;
  font-weight: 650;
  line-height: 1.25rem;
}

.dashboard-chart-panel__heading > span {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.dashboard-model-panel__body {
  display: grid;
  grid-template-columns: 10rem minmax(0, 1fr);
  align-items: center;
  gap: 1rem;
  min-height: 15rem;
  padding-top: 0.75rem;
}

.dashboard-doughnut {
  height: 10rem;
}

.dashboard-chart-empty {
  display: grid;
  min-height: 5rem;
  align-items: center;
  justify-content: center;
  align-content: center;
  gap: 0.25rem;
  margin-top: 0.75rem;
  border: 1px dashed hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted) / 0.28);
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  text-align: center;
}

.dashboard-chart-empty strong {
  color: hsl(var(--foreground));
  font-size: 0.75rem;
  font-weight: 650;
}

.dashboard-chart-empty span {
  font-size: 0.6875rem;
}

.dashboard-model-table-wrap {
  max-height: 14rem;
  overflow: auto;
}

.dashboard-model-table {
  width: 100%;
  min-width: 30rem;
  border-collapse: collapse;
  font-size: 0.6875rem;
}

.dashboard-model-table th,
.dashboard-model-table td {
  padding: 0.55rem 0.45rem;
  border-bottom: 1px solid hsl(var(--border) / 0.68);
  text-align: right;
}

.dashboard-model-table th {
  color: hsl(var(--muted-foreground));
  font-size: 0.625rem;
  font-weight: 600;
}

.dashboard-model-table th:first-child,
.dashboard-model-table td:first-child {
  text-align: left;
}

.dashboard-model-table td {
  color: hsl(var(--muted-foreground));
}

.dashboard-model-table td:first-child,
.dashboard-model-table td:nth-last-child(2) {
  color: hsl(var(--foreground));
}

.dashboard-model-name {
  display: inline-flex;
  max-width: 11rem;
  align-items: center;
  gap: 0.4rem;
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1180px) {
  .dashboard-chart-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 720px) {
  .dashboard-filter-bar,
  .dashboard-filter-bar__controls,
  .dashboard-filter-bar label {
    align-items: stretch;
    flex-direction: column;
  }

  .dashboard-filter-bar__controls,
  .dashboard-filter-bar__granularity {
    width: 100%;
  }

  .dashboard-model-panel__body {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-doughnut {
    width: 11rem;
    margin: 0 auto;
  }
}
</style>
