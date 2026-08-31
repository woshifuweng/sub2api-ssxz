<template>
  <AppSectionShell
    title="补充额度 / 订阅"
    subtitle="在充值中心选择适合你的商品，账户变化以账户记录为准。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div class="purchase-workbench">
      <div class="purchase-toolbar">
        <div>
          <p class="purchase-toolbar-title">官方充值中心</p>
          <p class="purchase-toolbar-copy">付款和验证码由充值中心独立完成。</p>
        </div>
        <a
          :href="purchaseUrl"
          class="purchase-link purchase-link-secondary"
          target="_blank"
          rel="noopener noreferrer"
        >
          在新窗口打开
        </a>
      </div>

      <div class="purchase-mobile-card">
        <p class="purchase-mobile-title">前往充值中心</p>
        <p class="purchase-mobile-copy">
          手机内嵌页面会压缩验证码，建议在独立窗口完成充值。完成后返回本站刷新余额即可。
        </p>
        <a
          :href="purchaseUrl"
          class="purchase-link purchase-link-primary"
          target="_blank"
          rel="noopener noreferrer"
        >
          打开充值中心
        </a>
      </div>

      <iframe
        :src="purchaseUrl"
        class="purchase-embed-frame"
        title="充值中心"
        referrerpolicy="no-referrer"
        loading="lazy"
      ></iframe>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import AppSectionShell from '@/components/user/AppSectionShell.vue'

const purchaseUrl = 'https://pay.ldxp.cn/shop/VT7XKDFI'
</script>

<style scoped>
.purchase-workbench {
  width: 100%;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card, 10px);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.purchase-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--ssxz-border);
}

.purchase-toolbar-title,
.purchase-mobile-title {
  color: var(--ssxz-text-primary);
  font-weight: 650;
}

.purchase-toolbar-copy,
.purchase-mobile-copy {
  margin-top: 0.25rem;
  color: var(--ssxz-text-muted);
  font-size: 0.8125rem;
  line-height: 1.5;
}

.purchase-link {
  display: inline-flex;
  min-height: 2.5rem;
  align-items: center;
  justify-content: center;
  border-radius: var(--ssxz-radius-control, 8px);
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 650;
  text-decoration: none;
  transition: border-color 150ms ease, background-color 150ms ease, color 150ms ease;
}

.purchase-link-secondary {
  flex: 0 0 auto;
  border: 1px solid var(--ssxz-border-strong, var(--ssxz-border));
  color: var(--ssxz-text-primary);
}

.purchase-link-secondary:hover {
  background: var(--ssxz-surface-hover, rgba(255, 255, 255, 0.04));
}

.purchase-link-primary {
  width: 100%;
  margin-top: 1.25rem;
  background: var(--ssxz-accent, #f2b84b);
  color: #17120a;
}

.purchase-link-primary:hover {
  background: var(--ssxz-accent-strong, #d99b3d);
}

.purchase-mobile-card {
  display: none;
}

.purchase-embed-frame {
  display: block;
  width: 100%;
  min-height: 80vh;
  border: none;
  background: #fff;
}

@media (max-width: 1100px) {
  .purchase-workbench {
    min-height: 0;
  }

  .purchase-toolbar,
  .purchase-embed-frame {
    display: none;
  }

  .purchase-mobile-card {
    display: block;
    padding: 1.25rem;
  }
}
</style>
