<template>
  <AppSectionShell
    title="充值 / 订阅"
    subtitle="为账户充值余额，或选择适合你的订阅套餐。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <PaymentCheckoutContent variant="workspace" />
  </AppSectionShell>
</template>

<script setup lang="ts">
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import PaymentCheckoutContent from './PaymentCheckoutContent.vue'
</script>
