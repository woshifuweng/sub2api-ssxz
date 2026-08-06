<template>
  <AppSectionShell
    title="补充额度 / 订阅"
    subtitle="在充值中心选择适合你的商品，账户变化以账户记录为准。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div class="purchase-workbench">
      <iframe
        src="https://pay.ldxp.cn/shop/VT7XKDFI"
        class="purchase-embed-frame"
        title="充值中心"
        referrerpolicy="no-referrer"
      ></iframe>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import AppSectionShell from '@/components/user/AppSectionShell.vue'
</script>

<style scoped>
.purchase-workbench {
  width: 100%;
  min-height: 80vh;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card, 10px);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.purchase-embed-frame {
  display: block;
  width: 100%;
  min-height: 80vh;
  border: none;
  background: #fff;
}

@media (max-width: 760px) {
  .purchase-workbench,
  .purchase-embed-frame {
    min-height: 85vh;
  }
}
</style>
