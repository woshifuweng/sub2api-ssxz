<template>
  <AppSectionShell
    :title="t('nav.shop')"
    subtitle="第三方商店是独立购买入口；站内充值与订阅请使用对应菜单。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div class="purchase-workbench">
      <iframe
        src="https://wzyp.cn/shop/VT7XKDFI"
        class="purchase-embed-frame"
        :title="t('nav.shop')"
        referrerpolicy="no-referrer"
      ></iframe>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
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
