<template>
  <section class="dashboard-list-panel">
    <header class="dashboard-list-panel__header">
      <div>
        <p>调用记录</p>
        <h2>{{ t('dashboard.recentUsage') }}</h2>
      </div>
      <span>{{ periodLabel || t('dashboard.last7Days') }}</span>
    </header>

    <div class="dashboard-list-panel__content">
      <div v-if="loading" class="dashboard-list-panel__loading">
        <LoadingSpinner size="lg" />
      </div>
      <div v-else-if="data.length === 0" class="dashboard-list-panel__empty">
        <EmptyState
          :title="hasAnyUsage ? t('dashboard.noUsageRecordsInRange') : t('dashboard.noUsageRecords')"
          :description="hasAnyUsage ? t('dashboard.tryWiderRange') : t('dashboard.startUsingApi')"
        />
      </div>
      <div v-else class="dashboard-usage-list">
        <article v-for="log in data" :key="log.id" class="dashboard-usage-row">
          <span class="dashboard-usage-row__icon" aria-hidden="true">
            <ModelIcon :model="log.model" size="18px" />
          </span>
          <div class="dashboard-usage-row__model">
            <strong>{{ log.model }}</strong>
            <span>{{ formatDateTime(log.created_at) }}</span>
          </div>
          <div class="dashboard-usage-row__tokens">
            <span>Token</span>
            <strong>{{ (log.input_tokens + log.output_tokens).toLocaleString() }}</strong>
          </div>
          <div class="dashboard-usage-row__cost">
            <span>{{ t('dashboard.actual') }} / {{ t('dashboard.standard') }}</span>
            <strong :title="`${formatCurrencyTitle(log.actual_cost)}；标准 ${formatCurrencyExact(log.total_cost)}`">{{ formatCurrency(log.actual_cost) }} <small>/ {{ formatCurrency(log.total_cost) }}</small></strong>
          </div>
        </article>

        <router-link to="/app/usage" class="dashboard-list-panel__link">
          {{ t('dashboard.viewAllUsage') }}
          <ArrowRight />
        </router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowRight } from '@lucide/vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { formatCurrency, formatCurrencyExact, formatCurrencyTitle, formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

withDefaults(defineProps<{
  data: UsageLog[]
  loading: boolean
  periodLabel?: string
  hasAnyUsage?: boolean
}>(), {
  periodLabel: '',
  hasAnyUsage: false
})

const { t } = useI18n()
</script>

<style scoped>
.dashboard-list-panel {
  min-width: 0;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  box-shadow: 0 1px 2px hsl(var(--shadow));
}

.dashboard-list-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  min-height: 4.25rem;
  padding: 0.9rem 1rem;
  border-bottom: 1px solid hsl(var(--border));
}

.dashboard-list-panel__header p,
.dashboard-list-panel__header h2 {
  margin: 0;
}

.dashboard-list-panel__header p,
.dashboard-list-panel__header > span,
.dashboard-usage-row span {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.dashboard-list-panel__header h2 {
  margin-top: 0.1rem;
  font-size: 0.875rem;
  font-weight: 650;
  line-height: 1.25rem;
}

.dashboard-list-panel__header > span {
  border: 1px solid hsl(var(--border));
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  background: hsl(var(--muted));
  font-weight: 600;
}

.dashboard-list-panel__content {
  padding: 0 1rem 0.75rem;
}

.dashboard-list-panel__loading,
.dashboard-list-panel__empty {
  display: flex;
  min-height: 18rem;
  align-items: center;
  justify-content: center;
}

.dashboard-usage-list {
  display: grid;
}

.dashboard-usage-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1.2fr) minmax(5rem, 0.55fr) minmax(8rem, 0.7fr);
  align-items: center;
  gap: 0.75rem;
  min-height: 4.5rem;
  border-bottom: 1px solid hsl(var(--border) / 0.68);
}

.dashboard-usage-row:hover {
  background: hsl(var(--muted) / 0.45);
}

.dashboard-usage-row__icon {
  display: inline-flex;
  width: 2.25rem;
  height: 2.25rem;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted));
}

.dashboard-usage-row__model,
.dashboard-usage-row__tokens,
.dashboard-usage-row__cost {
  display: grid;
  min-width: 0;
  gap: 0.15rem;
}

.dashboard-usage-row__model strong {
  overflow: hidden;
  font-size: 0.75rem;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-usage-row__tokens,
.dashboard-usage-row__cost {
  text-align: right;
}

.dashboard-usage-row__tokens strong,
.dashboard-usage-row__cost strong {
  font-size: 0.75rem;
  font-weight: 650;
}

.dashboard-usage-row__cost small {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 500;
}

.dashboard-list-panel__link {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  color: hsl(var(--foreground));
  font-size: 0.75rem;
  font-weight: 600;
}

.dashboard-list-panel__link svg {
  width: 0.9rem;
  height: 0.9rem;
  stroke-width: 1.8;
}

@media (max-width: 640px) {
  .dashboard-usage-row {
    grid-template-columns: auto minmax(0, 1fr) auto;
    padding: 0.75rem 0;
  }

  .dashboard-usage-row__tokens {
    display: none;
  }

  .dashboard-usage-row__cost {
    min-width: 7rem;
  }
}
</style>
