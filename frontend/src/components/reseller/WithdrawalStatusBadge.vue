<template>
  <span :class="['reseller-status-badge', `reseller-status-badge--${status}`]">
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WithdrawStatus } from '@/api/reseller'

const props = defineProps<{
  status: WithdrawStatus
}>()

const label = computed(() => ({
  pending: '待审核',
  approved: '已完成',
  rejected: '已拒绝',
  cancelled: '已撤销'
})[props.status])
</script>

<style scoped>
.reseller-status-badge {
  display: inline-flex;
  min-height: 1.5rem;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  padding: 0 0.55rem;
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
  font-weight: 600;
  white-space: nowrap;
}

.reseller-status-badge--pending {
  border-color: var(--ssxz-warning-border, #854d0e);
  color: var(--ssxz-warning-text, #f59e0b);
}

.reseller-status-badge--approved {
  border-color: var(--ssxz-success-border, #166534);
  color: var(--ssxz-success-text, #22c55e);
}

.reseller-status-badge--rejected {
  border-color: var(--ssxz-danger-border, #991b1b);
  color: var(--ssxz-danger-text, #ef4444);
}
</style>
