<template>
  <AppSectionShell
    title="充值"
    subtitle="按当前账号可用方式补充额度，到账结果以账户记录为准。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <PurchaseSubscriptionView v-if="legacyPurchaseEnabled && !paymentEnabled" embedded />
    <div v-else-if="!paymentEnabled" class="rounded-xl border border-gray-200 bg-white p-8 text-center shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">当前使用兑换码补充额度</h2>
      <p class="mx-auto mt-2 max-w-xl text-sm leading-6 text-gray-600 dark:text-gray-300">
        输入可用兑换码后，额度会进入当前账户。已有充值、兑换和账户调整记录可在订单页查看。
      </p>
      <div class="mt-6 flex flex-col justify-center gap-3 sm:flex-row">
        <RouterLink to="/app/redeem" class="btn btn-primary">
          使用兑换码
        </RouterLink>
        <RouterLink to="/app/orders" class="btn btn-secondary">
          查看账户记录
        </RouterLink>
      </div>
    </div>
    <PaymentCheckoutContent v-else variant="workspace" />
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useAppStore } from '@/stores'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import PaymentCheckoutContent from './PaymentCheckoutContent.vue'
import PurchaseSubscriptionView from './PurchaseSubscriptionView.vue'

const appStore = useAppStore()
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
const legacyPurchaseEnabled = computed(() => !!appStore.cachedPublicSettings?.purchase_subscription_enabled)
</script>
