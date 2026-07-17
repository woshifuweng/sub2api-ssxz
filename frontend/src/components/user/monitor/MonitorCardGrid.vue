<template>
  <div class="channel-monitor-grid">
    <div
      v-if="loading && items.length === 0"
      class="grid gap-5 grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
    >
      <div
        v-for="i in 6"
        :key="i"
        class="channel-monitor-skeleton min-h-[280px] animate-pulse"
      >
        <div class="flex items-start gap-3">
          <div class="w-9 h-9 rounded-xl bg-gray-200 dark:bg-dark-700"></div>
          <div class="flex-1 space-y-2">
            <div class="h-4 w-2/3 rounded bg-gray-200 dark:bg-dark-700"></div>
            <div class="h-3 w-1/2 rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
          <div class="h-6 w-16 rounded-full bg-gray-200 dark:bg-dark-700"></div>
        </div>
        <div class="mt-5 grid grid-cols-2 gap-2">
          <div class="h-16 rounded-xl bg-gray-100 dark:bg-dark-900/40"></div>
          <div class="h-16 rounded-xl bg-gray-100 dark:bg-dark-900/40"></div>
        </div>
        <div class="mt-6 h-5 w-full rounded bg-gray-100 dark:bg-dark-900/40"></div>
      </div>
    </div>

    <EmptyState
      v-else-if="items.length === 0"
      class="channel-monitor-empty"
      :title="t('channelStatus.empty.title')"
      :description="emptyDescription || t('channelStatus.empty.description')"
    >
      <template #icon>
        <Icon name="inbox" size="lg" aria-hidden="true" />
      </template>
    </EmptyState>

    <div
      v-else
      class="grid gap-5 grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
    >
      <MonitorCard
        v-for="item in items"
        :key="item.id"
        :item="item"
        :window="window"
        :availability-value="resolveAvailability(item)"
        :countdown-seconds="countdownSeconds"
        @click="emit('cardClick', item)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UserMonitorView, UserMonitorDetail } from '@/api/channelMonitor'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import MonitorCard from './MonitorCard.vue'

const props = defineProps<{
  items: UserMonitorView[]
  window: '7d' | '15d' | '30d'
  countdownSeconds: number
  loading: boolean
  detailCache: Record<number, UserMonitorDetail>
  emptyDescription?: string
}>()

const emit = defineEmits<{
  (e: 'cardClick', item: UserMonitorView): void
}>()

const { t } = useI18n()

function resolveAvailability(item: UserMonitorView): number | null {
  if (props.window === '7d') {
    return item.availability_7d ?? null
  }
  const detail = props.detailCache[item.id]
  if (!detail) return null
  const primary = detail.models.find(m => m.model === item.primary_model)
  if (!primary) return null
  return props.window === '15d' ? primary.availability_15d ?? null : primary.availability_30d ?? null
}
</script>

<style scoped>
.channel-monitor-grid {
  min-width: 0;
}

.channel-monitor-grid > :deep(.channel-monitor-empty) {
  min-height: 17rem;
  border-color: var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.channel-monitor-empty :deep(.empty-state-visual) {
  width: 3.75rem;
  height: 3.75rem;
  margin-bottom: 1.25rem;
  border-color: var(--ssxz-border);
  border-radius: 1rem;
  color: var(--ssxz-text-muted);
  background: var(--ssxz-surface-muted);
}

.channel-monitor-empty :deep(.empty-state-visual svg) {
  width: 1.75rem;
  height: 1.75rem;
  stroke-width: 1.6;
}

.channel-monitor-empty :deep(.empty-state-title) {
  margin-bottom: 0.5rem;
  color: var(--ssxz-text);
  font-size: 1rem;
  font-weight: 650;
}

.channel-monitor-empty :deep(.empty-state-description) {
  max-width: 28rem;
  color: var(--ssxz-text-muted);
  line-height: 1.7;
}

.channel-monitor-skeleton {
  min-width: 0;
  border-color: var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}
</style>
