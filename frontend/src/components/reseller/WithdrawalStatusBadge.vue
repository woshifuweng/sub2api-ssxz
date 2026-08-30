<template>
  <span :class="['reseller-status-badge', `reseller-status-badge--${status}`]">
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WithdrawStatus } from '@/api/reseller'

const props = defineProps<{
  status: WithdrawStatus
}>()
const { t } = useI18n()

const label = computed(() => ({
  pending: t('reseller.status.pending'),
  approved: t('reseller.status.approved'),
  rejected: t('reseller.status.rejected'),
  cancelled: t('reseller.status.cancelled')
})[props.status])
</script>

<style scoped>
.reseller-status-badge {
  display: inline-flex;
  min-height: 1.5rem;
  align-items: center;
  border: 1px solid light-dark(#e5e7eb, #374151);
  border-radius: 999px;
  padding: 0 0.55rem;
  color: light-dark(#6b7280, #9ca3af);
  font-size: 0.72rem;
  font-weight: 600;
  white-space: nowrap;
}

.reseller-status-badge--pending {
  border-color: #d97706;
  color: #f59e0b;
}

.reseller-status-badge--approved {
  border-color: #16a34a;
  color: #22c55e;
}

.reseller-status-badge--rejected {
  border-color: #dc2626;
  color: #ef4444;
}
</style>
