<template>
  <AppSectionShell
    title="充值 / 订阅"
    subtitle="为账户充值余额，或选择适合你的订阅套餐。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div v-if="!paymentEnabled" class="rounded-xl border border-gray-200 bg-white p-8 text-center shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">充值 / 订阅暂未开启</h2>
      <p class="mx-auto mt-2 max-w-xl text-sm leading-6 text-gray-600 dark:text-gray-300">
        当前暂未开放在线充值或订阅入口，请先使用已有额度或兑换码，稍后再回来查看。
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

const appStore = useAppStore()
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
</script>
