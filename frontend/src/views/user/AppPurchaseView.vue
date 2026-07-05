<template>
  <PurchaseSubscriptionView v-if="legacyPurchaseEnabled && !paymentEnabled" />
  <AppSectionShell
    v-else
    title="充值"
    subtitle="选择充值金额或套餐，支付完成后额度会自动到账。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div v-if="!paymentEnabled" class="rounded-xl border border-gray-200 bg-white p-8 text-center shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">充值暂未开启</h2>
      <p class="mx-auto mt-2 max-w-xl text-sm leading-6 text-gray-600 dark:text-gray-300">
        当前暂未开放在线充值，可先使用已有额度或兑换码。
      </p>
    </div>
    <PaymentCheckoutContent v-else variant="workspace" />
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import PaymentCheckoutContent from './PaymentCheckoutContent.vue'
import PurchaseSubscriptionView from './PurchaseSubscriptionView.vue'

const appStore = useAppStore()
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
const legacyPurchaseEnabled = computed(() => !!appStore.cachedPublicSettings?.purchase_subscription_enabled)
</script>
