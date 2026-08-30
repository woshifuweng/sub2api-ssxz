<template>
  <section class="dashboard-actions-panel">
    <header>
      <p>常用任务</p>
      <h2>{{ t('dashboard.quickActions') }}</h2>
    </header>
    <div class="dashboard-actions-panel__list">
      <button
        v-for="action in actions"
        :key="action.title"
        type="button"
        @click="router.push(action.to)"
      >
        <span class="dashboard-actions-panel__icon" aria-hidden="true">
          <component :is="action.icon" />
        </span>
        <span class="dashboard-actions-panel__copy">
          <strong>{{ action.title }}</strong>
          <small>{{ action.description }}</small>
        </span>
        <ChevronRight class="dashboard-actions-panel__chevron" aria-hidden="true" />
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChartSpline, ChevronRight, CreditCard, KeyRound, ReceiptText } from '@lucide/vue'
import { useAppStore } from '@/stores'

const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)

const actions = computed(() => [
  {
    title: 'API 密钥',
    description: '创建 Key 并接入第三方客户端',
    to: '/app/keys',
    icon: KeyRound
  },
  {
    title: t('dashboard.viewUsage'),
    description: t('dashboard.checkDetailedLogs'),
    to: '/app/usage',
    icon: ChartSpline
  },
  paymentEnabled.value
    ? {
        title: '补充额度',
        description: '补充额度或查看账户记录',
        to: '/app/purchase',
        icon: CreditCard
      }
    : {
        title: '账户记录',
        description: '查看账户记录和到账状态',
        to: '/app/orders',
        icon: ReceiptText
      }
])
</script>

<style scoped>
.dashboard-actions-panel {
  min-width: 0;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  box-shadow: 0 1px 2px hsl(var(--shadow));
}

.dashboard-actions-panel header {
  min-height: 4.25rem;
  padding: 0.9rem 1rem;
  border-bottom: 1px solid hsl(var(--border));
}

.dashboard-actions-panel header p,
.dashboard-actions-panel header h2 {
  margin: 0;
}

.dashboard-actions-panel header p {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1rem;
}

.dashboard-actions-panel header h2 {
  margin-top: 0.1rem;
  font-size: 0.875rem;
  font-weight: 650;
  line-height: 1.25rem;
}

.dashboard-actions-panel__list {
  display: grid;
  padding: 0.45rem;
}

.dashboard-actions-panel button {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.7rem;
  min-height: 4.75rem;
  border: 0;
  border-bottom: 1px solid hsl(var(--border) / 0.68);
  border-radius: var(--radius);
  padding: 0.65rem;
  color: hsl(var(--foreground));
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.dashboard-actions-panel button:last-child {
  border-bottom: 0;
}

.dashboard-actions-panel button:hover {
  background: hsl(var(--muted) / 0.72);
}

.dashboard-actions-panel__icon {
  display: inline-flex;
  width: 2.25rem;
  height: 2.25rem;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted));
}

.dashboard-actions-panel__icon :deep(svg),
.dashboard-actions-panel__chevron {
  width: 1rem;
  height: 1rem;
  stroke-width: 1.8;
}

.dashboard-actions-panel__copy {
  display: grid;
  min-width: 0;
  gap: 0.15rem;
}

.dashboard-actions-panel__copy strong {
  font-size: 0.75rem;
  font-weight: 650;
}

.dashboard-actions-panel__copy small,
.dashboard-actions-panel__chevron {
  color: hsl(var(--muted-foreground));
}

.dashboard-actions-panel__copy small {
  overflow: hidden;
  font-size: 0.6875rem;
  line-height: 1rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
