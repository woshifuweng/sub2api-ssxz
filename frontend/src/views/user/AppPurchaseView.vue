<template>
  <AppSectionShell
    title="补充额度 / 订阅"
    subtitle="按当前账号可用方式补充额度，账户变化以账户记录为准。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div class="purchase-workbench">
    <PurchaseSubscriptionView v-if="legacyPurchaseEnabled && !paymentEnabled" embedded />
    <div v-else-if="!paymentEnabled" class="card purchase-empty-state">
      <div class="purchase-empty-state__icon"><Icon name="gift" size="lg" /></div>
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">当前可用方式：兑换码</h2>
      <p class="mx-auto mt-2 max-w-xl text-sm leading-6 text-gray-600 dark:text-gray-300">
        输入可用兑换码后，额度会进入当前账户。充值、兑换和账户调整记录可在账户记录查看。
      </p>
      <div class="mt-6 flex flex-col justify-center gap-3 sm:flex-row">
        <RouterLink to="/app/redeem" class="btn btn-primary">
          去兑换额度
        </RouterLink>
        <RouterLink to="/app/orders" class="btn btn-secondary">
          查看账户记录
        </RouterLink>
      </div>
    </div>
    <PaymentCheckoutContent v-else variant="workspace" />
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useAppStore } from '@/stores'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import PaymentCheckoutContent from './PaymentCheckoutContent.vue'
import PurchaseSubscriptionView from './PurchaseSubscriptionView.vue'

const appStore = useAppStore()
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
const legacyPurchaseEnabled = computed(() => !!appStore.cachedPublicSettings?.purchase_subscription_enabled)
</script>

<style scoped>
.purchase-workbench {
  display: grid;
  gap: 1.5rem;
}

.purchase-empty-state {
  display: grid;
  min-height: 22rem;
  place-items: center;
  align-content: center;
  gap: 0.75rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
  padding: 2rem;
  text-align: center;
}

.purchase-empty-state__icon {
  display: grid;
  width: 3.75rem;
  height: 3.75rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
}

.purchase-empty-state h2 {
  margin: 0;
  color: var(--ssxz-text);
  font-size: 1.125rem;
  font-weight: 700;
}

.purchase-empty-state p {
  margin: 0;
  color: var(--ssxz-text-muted);
}

.purchase-workbench :deep(.payment-checkout .card),
.purchase-workbench :deep(.checkout-overview),
.purchase-workbench :deep(.purchase-page-layout .card) {
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.purchase-workbench :deep(.payment-checkout) {
  gap: 1.5rem;
}
</style>
