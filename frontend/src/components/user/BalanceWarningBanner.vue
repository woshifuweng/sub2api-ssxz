<template>
  <section
    v-if="showWarning"
    class="balance-warning-banner"
    data-testid="balance-warning-banner"
    role="alert"
  >
    <div class="balance-warning-banner__message">
      <Icon name="exclamationCircle" size="sm" aria-hidden="true" />
      <span>{{ t('balanceWarning.message') }}</span>
    </div>
    <RouterLink to="/app/redeem" class="balance-warning-banner__action">
      <Icon name="gift" size="sm" aria-hidden="true" />
      <span>{{ t('balanceWarning.action') }}</span>
    </RouterLink>
  </section>
  <section
    v-if="showLowBalanceWarning"
    class="balance-warning-banner balance-warning-banner--low"
    data-testid="low-balance-warning-banner"
    role="alert"
  >
    <div class="balance-warning-banner__message">
      <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
      <span>{{ t('lowBalanceWarning.message', { threshold: lowBalanceThreshold.toFixed(2) }) }}</span>
    </div>
    <RouterLink to="/app/purchase" class="balance-warning-banner__action">
      <Icon name="creditCard" size="sm" aria-hidden="true" />
      <span>{{ t('lowBalanceWarning.action') }}</span>
    </RouterLink>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const showWarning = computed(() => Boolean(authStore.user) && (authStore.user?.balance ?? 0) <= 0)
const lowBalanceThreshold = computed(() => {
  const configured = Number(appStore.cachedPublicSettings?.balance_low_notify_threshold)
  return Number.isFinite(configured) && configured >= 0 ? configured : 1
})
const showLowBalanceWarning = computed(() => {
  const balance = authStore.user?.balance ?? 0
  const enabled = appStore.cachedPublicSettings?.balance_low_notify_enabled ?? true
  return Boolean(authStore.user) && enabled && balance > 0 && balance < lowBalanceThreshold.value
})
</script>

<style scoped>
.balance-warning-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid color-mix(in srgb, var(--ssxz-danger) 38%, var(--ssxz-border));
  border-radius: var(--ssxz-radius-card);
  padding: 0.8rem 1rem;
  color: var(--ssxz-danger);
  background: color-mix(in srgb, var(--ssxz-danger) 9%, var(--ssxz-surface));
}

.balance-warning-banner__message,
.balance-warning-banner__action {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.balance-warning-banner__message {
  min-width: 0;
  font-size: 0.875rem;
  font-weight: 650;
  line-height: 1.35rem;
}

.balance-warning-banner__message span {
  overflow-wrap: anywhere;
}

.balance-warning-banner__action {
  flex: 0 0 auto;
  min-height: 2.25rem;
  border: 1px solid var(--ssxz-danger);
  border-radius: var(--ssxz-radius-button);
  padding: 0.45rem 0.75rem;
  color: var(--ssxz-danger);
  font-size: 0.8125rem;
  font-weight: 650;
  line-height: 1.1rem;
  transition: background-color 150ms ease, color 150ms ease;
}

.balance-warning-banner__action:hover,
.balance-warning-banner__action:focus-visible {
  color: var(--ssxz-action-text);
  background: var(--ssxz-danger);
}

.balance-warning-banner--low {
  border-color: color-mix(in srgb, var(--ssxz-warning) 38%, var(--ssxz-border));
  color: var(--ssxz-warning);
  background: color-mix(in srgb, var(--ssxz-warning) 9%, var(--ssxz-surface));
}

.balance-warning-banner--low .balance-warning-banner__action {
  border-color: var(--ssxz-warning);
  color: var(--ssxz-warning);
}

.balance-warning-banner--low .balance-warning-banner__action:hover,
.balance-warning-banner--low .balance-warning-banner__action:focus-visible {
  background: var(--ssxz-warning);
}

.balance-warning-banner__action:focus-visible {
  outline: 2px solid var(--ssxz-danger);
  outline-offset: 2px;
}

@media (max-width: 640px) {
  .balance-warning-banner,
  .balance-warning-banner--low {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
