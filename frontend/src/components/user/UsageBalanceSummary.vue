<template>
  <section class="usage-summary-grid" :aria-label="t('usage.workbench.billingExplanationTitle')">
    <article class="usage-summary-card">
      <div>
        <span>{{ t('usage.workbench.balanceTitle') }}</span>
        <strong :title="formatCurrencyTitle(authStore.user?.balance || 0)">{{ formatCurrency(authStore.user?.balance || 0) }}</strong>
        <p :role="balanceRefreshError ? 'alert' : undefined">{{ t(balanceRefreshError ? 'usage.workbench.balanceRefreshError' : 'usage.workbench.balanceDescription') }}</p>
      </div>
      <RouterLink to="/app/purchase" class="btn btn-primary">{{ t('usage.workbench.recharge') }}</RouterLink>
    </article>
    <article class="usage-summary-card">
      <div>
        <span>{{ t('usage.workbench.monthlyCostTitle') }}</span>
        <strong :title="statsLoadError ? undefined : formatCurrencyTitle(monthlyCost)">{{ statsLoadError ? t('usage.workbench.unavailable') : formatCurrency(monthlyCost) }}</strong>
        <p :role="statsLoadError ? 'alert' : undefined">{{ statsLoadError ? t('usage.workbench.statsLoadError') : t('usage.workbench.monthlyUsageSummary', { requests, tokens }) }}</p>
      </div>
      <button class="btn btn-secondary" :disabled="refreshing" @click="refresh">{{ t('common.refresh') }}</button>
    </article>
    <details class="usage-billing-help">
      <summary>{{ t('usage.workbench.billingExplanationTitle') }}</summary>
      <p>{{ t('usage.workbench.billingExplanationDescription') }}</p>
      <ul>
        <li v-for="key in ['successCharged', 'failureNoCharge', 'zeroCost']" :key="key">{{ t(`usage.workbench.billingExplanationItems.${key}`) }}</li>
      </ul>
    </details>
  </section>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usageAPI } from '@/api'
import { formatCurrency, formatCurrencyTitle } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const refreshing = ref(false)
const balanceRefreshError = ref(false)
const statsLoadError = ref(false)
const monthlyCost = ref(0)
const requests = ref('0')
const tokens = ref('0')
let disposed = false
const dateKey = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`

async function refresh() {
  if (refreshing.value || disposed) return
  refreshing.value = true
  const today = new Date()
  // Account-wide calendar-month summary deliberately does not follow detail filters.
  await Promise.allSettled([
    authStore.refreshUser().then(() => { if (!disposed) balanceRefreshError.value = false }).catch(() => { if (!disposed) balanceRefreshError.value = true }),
    usageAPI.getStatsByDateRange(dateKey(new Date(today.getFullYear(), today.getMonth(), 1)), dateKey(today))
      .then((stats) => {
        if (disposed) return
        statsLoadError.value = false
        monthlyCost.value = stats.total_actual_cost || 0
        requests.value = Number(stats.total_requests || 0).toLocaleString()
        tokens.value = Number(stats.total_tokens || 0).toLocaleString()
      }).catch(() => { if (!disposed) statsLoadError.value = true })
  ])
  if (!disposed) refreshing.value = false
}

onMounted(refresh)
onUnmounted(() => { disposed = true })
</script>

<style scoped>
.usage-summary-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
.usage-summary-card { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 1.25rem; border: 1px solid var(--ssxz-border); border-radius: var(--ssxz-radius-card); background: var(--ssxz-surface-raised); }
.usage-summary-card strong { display: block; font-size: 1.65rem; font-variant-numeric: tabular-nums; color: var(--ssxz-text); }
.usage-summary-card span, .usage-summary-card p, .usage-billing-help { color: var(--ssxz-text-secondary); font-size: .8rem; }
.usage-billing-help { grid-column: 1 / -1; padding: 1rem; border: 1px solid var(--ssxz-border); border-radius: var(--ssxz-radius-card); }
.usage-billing-help summary { cursor: pointer; font-weight: 600; }
.usage-billing-help p, .usage-billing-help ul { margin-top: .5rem; }
@media (max-width: 767px) { .usage-summary-grid { grid-template-columns: 1fr; } .usage-summary-card { flex-wrap: wrap; } }
</style>
