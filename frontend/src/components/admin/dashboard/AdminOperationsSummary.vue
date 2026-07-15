<template>
  <section
    class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-900"
    data-testid="operations-summary"
  >
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-gray-800 sm:px-5">
      <div>
        <div class="flex items-center gap-2">
          <h2 class="text-sm font-semibold text-gray-950 dark:text-white">
            {{ t('admin.dashboard.operations.title') }}
          </h2>
          <span class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('admin.dashboard.operations.rangeLabel') }}
          </span>
        </div>
        <p v-if="error" class="mt-1 text-xs text-red-600 dark:text-red-400">
          {{ t('admin.dashboard.operations.loadFailed') }}
        </p>
      </div>

      <div class="flex items-center gap-2">
        <div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-0.5 dark:border-gray-700 dark:bg-gray-800" role="group">
          <button
            v-for="option in rangeOptions"
            :key="option.value"
            type="button"
            class="min-w-12 rounded px-2.5 py-1.5 text-xs font-medium transition-colors"
            :class="range === option.value
              ? 'bg-white text-gray-950 shadow-sm dark:bg-gray-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'"
            :aria-pressed="range === option.value"
            :data-testid="`operations-range-${option.value}`"
            @click="emit('update:range', option.value)"
          >
            {{ option.label }}
          </button>
        </div>
        <button
          type="button"
          class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-900 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white"
          :disabled="loading"
          :title="t('common.refresh')"
          data-testid="operations-refresh"
          @click="emit('refresh')"
        >
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        </button>
      </div>
    </div>

    <div class="grid grid-cols-2 border-b border-gray-100 dark:border-gray-800 lg:grid-cols-4">
      <button
        v-for="metric in primaryMetrics"
        :key="metric.key"
        type="button"
        class="group min-h-24 border-gray-100 px-4 py-4 text-left transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500 dark:border-gray-800 dark:hover:bg-gray-800/70 sm:px-5"
        :class="metric.borderClass"
        :data-testid="`operations-metric-${metric.key}`"
        @click="emit('drilldown', metric.target)"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ metric.label }}</span>
          <Icon name="chevronRight" size="xs" class="text-gray-300 transition-transform group-hover:translate-x-0.5 group-hover:text-gray-500 dark:text-gray-600" />
        </div>
        <p class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">
          {{ loading && !summary ? '—' : metric.value }}
        </p>
        <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">{{ metric.detail }}</p>
      </button>
    </div>

    <div class="grid lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.35fr)]">
      <div class="border-b border-gray-100 p-4 dark:border-gray-800 sm:p-5 lg:border-b-0 lg:border-r">
        <div class="mb-3 flex items-center justify-between">
          <h3 class="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
            {{ t('admin.dashboard.operations.rebateTitle') }}
          </h3>
          <button
            type="button"
            class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
            @click="emit('drilldown', 'affiliates')"
          >
            {{ t('admin.dashboard.operations.viewDetails') }}
          </button>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-gray-800">
          <button
            v-for="item in rebateMetrics"
            :key="item.key"
            type="button"
            class="flex w-full items-center justify-between gap-4 py-3 text-left first:pt-1 last:pb-1 hover:text-primary-700 dark:hover:text-primary-300"
            :title="item.hint"
            @click="emit('drilldown', 'affiliates')"
          >
            <span>
              <span class="block text-sm font-medium text-gray-800 dark:text-gray-200">{{ item.label }}</span>
              <span class="mt-0.5 block text-xs text-gray-400 dark:text-gray-500">{{ item.hint }}</span>
            </span>
            <span class="shrink-0 text-sm font-semibold text-gray-950 dark:text-white">{{ item.value }}</span>
          </button>
        </div>
      </div>

      <div class="p-4 sm:p-5">
        <div class="mb-3 flex items-center justify-between">
          <div>
            <h3 class="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
              {{ t('admin.dashboard.operations.topCustomers') }}
            </h3>
            <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
              {{ t('admin.dashboard.operations.byActualCost') }}
            </p>
          </div>
          <button
            type="button"
            class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
            @click="emit('drilldown', 'usage')"
          >
            {{ t('admin.dashboard.operations.viewAll') }}
          </button>
        </div>

        <div v-if="topCustomers.length" class="divide-y divide-gray-100 dark:divide-gray-800">
          <button
            v-for="(customer, index) in topCustomers"
            :key="customer.user_id"
            type="button"
            class="grid w-full grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-2 py-2.5 text-left hover:text-primary-700 dark:hover:text-primary-300"
            :data-testid="`operations-top-${customer.user_id}`"
            @click="emit('drilldown', 'customer', customer.user_id)"
          >
            <span class="text-xs tabular-nums text-gray-400 dark:text-gray-500">{{ index + 1 }}</span>
            <span class="min-w-0">
              <span class="block truncate text-sm font-medium text-gray-800 dark:text-gray-200">{{ customerName(customer) }}</span>
              <span class="block text-xs text-gray-400 dark:text-gray-500">
                {{ formatInteger(customer.requests) }} {{ t('admin.dashboard.operations.requestsUnit') }} ·
                {{ formatInteger(customer.active_keys) }} {{ t('admin.dashboard.operations.keysUnit') }}
              </span>
            </span>
            <span class="text-sm font-semibold tabular-nums text-gray-950 dark:text-white">{{ formatMoney(customer.actual_cost) }}</span>
          </button>
        </div>
        <div v-else class="flex min-h-32 items-center justify-center text-sm text-gray-400 dark:text-gray-500">
          {{ t('admin.dashboard.operations.zeroCustomers') }}
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import type {
  DashboardOperationsDrilldown,
  DashboardOperationsRange,
  DashboardOperationsSummary,
  DashboardOperationsTopCustomer
} from '@/api/admin/dashboard'
import Icon from '@/components/icons/Icon.vue'
import { formatCurrency } from '@/utils/format'

const props = defineProps<{
  summary: DashboardOperationsSummary | null
  range: DashboardOperationsRange
  loading?: boolean
  error?: boolean
}>()

const emit = defineEmits<{
  'update:range': [value: DashboardOperationsRange]
  refresh: []
  drilldown: [target: DashboardOperationsDrilldown, userId?: number]
}>()

const { t } = useI18n()

const rangeOptions = computed(() => [
  { value: 'today' as const, label: t('admin.dashboard.operations.today') },
  { value: '7d' as const, label: t('admin.dashboard.operations.days7') },
  { value: '30d' as const, label: t('admin.dashboard.operations.days30') }
])

const formatMoney = (value = 0): string => formatCurrency(value)

const formatInteger = (value = 0): string => Math.max(0, value).toLocaleString()

const primaryMetrics = computed(() => [
  {
    key: 'customers',
    label: t('admin.dashboard.operations.newCustomers'),
    value: formatInteger(props.summary?.new_customers),
    detail: t('admin.dashboard.operations.registeredInRange'),
    target: 'users' as const,
    borderClass: 'border-r border-b lg:border-b-0'
  },
  {
    key: 'spend',
    label: t('admin.dashboard.operations.customerSpend'),
    value: formatMoney(props.summary?.customer_actual_cost),
    detail: t('admin.dashboard.operations.actualCostOnly'),
    target: 'usage' as const,
    borderClass: 'border-b lg:border-r lg:border-b-0'
  },
  {
    key: 'recharge',
    label: t('admin.dashboard.operations.inviteeRecharge'),
    value: formatMoney(props.summary?.invitee_recharge_amount),
    detail: t('admin.dashboard.operations.completedBalanceOrders'),
    target: 'orders' as const,
    borderClass: 'border-r lg:border-b-0'
  },
  {
    key: 'active',
    label: t('admin.dashboard.operations.activeCustomersAndKeys'),
    value: `${formatInteger(props.summary?.active_customers)} / ${formatInteger(props.summary?.active_api_keys)}`,
    detail: t('admin.dashboard.operations.distinctUsage'),
    target: 'apiKeys' as const,
    borderClass: ''
  }
])

const rebateMetrics = computed(() => [
  {
    key: 'pending',
    label: t('admin.dashboard.operations.rebatePending'),
    value: formatMoney(props.summary?.rebate_pending),
    hint: t('admin.dashboard.operations.rebatePendingHint')
  },
  {
    key: 'available',
    label: t('admin.dashboard.operations.rebateAvailable'),
    value: formatMoney(props.summary?.rebate_available),
    hint: t('admin.dashboard.operations.rebateAvailableHint')
  },
  {
    key: 'transferred',
    label: t('admin.dashboard.operations.rebateTransferred'),
    value: formatMoney(props.summary?.rebate_transferred),
    hint: t('admin.dashboard.operations.rebateTransferredHint')
  }
])

const topCustomers = computed(() => props.summary?.top_customers?.slice(0, 5) ?? [])

const customerName = (customer: DashboardOperationsTopCustomer): string => {
  return customer.username?.trim() || customer.email?.trim() || `#${customer.user_id}`
}
</script>
