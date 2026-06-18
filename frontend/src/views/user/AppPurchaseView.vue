<template>
  <AppSectionShell
    title="充值 / 订阅"
    subtitle="在新版工作台里完成余额充值或订阅购买。这里复用现有支付链路，不创建新的账务逻辑。"
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
