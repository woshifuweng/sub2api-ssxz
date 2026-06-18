<template>
  <AppSectionShell
    title="充值 / 订阅"
    subtitle="工作台版充值入口正在收口。这里先保留余额和状态说明，不跳转到旧后台壳，也不伪造支付能力。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <section class="purchase-workbench" aria-label="充值 / 订阅">
      <div class="purchase-summary-grid">
        <article class="purchase-summary-card">
          <div class="summary-icon">
            <Icon name="creditCard" size="sm" />
          </div>
          <div>
            <span>账户余额</span>
            <strong>{{ balanceText }}</strong>
            <p>余额可用于站内聊天、图片生成和 API Key / 第三方接入调用。</p>
          </div>
        </article>

        <article class="purchase-summary-card">
          <div class="summary-icon">
            <Icon name="chartBar" size="sm" />
          </div>
          <div>
            <span>充值状态</span>
            <strong>待迁入</strong>
            <p>真实支付链路仍保留在系统里，后续会单独迁到新版工作台页面。</p>
          </div>
        </article>
      </div>

      <section class="purchase-panel">
        <div class="panel-icon">
          <Icon name="creditCard" size="lg" />
        </div>
        <strong>工作台版充值页正在整理</strong>
        <p>
          这一页先解决普通用户入口不再跳到旧后台壳的问题。当前不创建订单、不发起支付、不展示假套餐。
          如需充值，请先联系管理员；支付页迁入会单独做小 PR 验收。
        </p>
        <div class="purchase-actions">
          <RouterLink to="/app/usage" class="purchase-action primary">返回用量中心</RouterLink>
          <RouterLink to="/app/keys" class="purchase-action">查看 API Key / 第三方接入</RouterLink>
        </div>
      </section>
    </section>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

const balanceText = computed(() => `$${Number(authStore.user?.balance || 0).toFixed(2)}`)
</script>

<style scoped>
.purchase-workbench {
  display: grid;
  gap: 1rem;
}

.purchase-summary-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.purchase-summary-card,
.purchase-panel {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface);
  box-shadow: var(--ssxz-shadow);
}

.purchase-summary-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.85rem;
  align-items: center;
  border-radius: 1.25rem;
  padding: 1rem;
}

.summary-icon,
.panel-icon {
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--ssxz-action-soft) 78%, transparent);
  color: var(--ssxz-action);
}

.summary-icon {
  width: 2.45rem;
  height: 2.45rem;
  border-radius: 0.85rem;
}

.purchase-summary-card span {
  color: var(--ssxz-text-muted);
  font-size: 0.82rem;
  font-weight: 800;
}

.purchase-summary-card strong {
  display: block;
  margin-top: 0.15rem;
  color: var(--ssxz-text-primary);
  font-size: clamp(1.45rem, 3vw, 2rem);
  letter-spacing: 0;
}

.purchase-summary-card p {
  margin: 0.28rem 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  line-height: 1.55;
}

.purchase-panel {
  display: grid;
  min-height: 20rem;
  place-items: center;
  align-content: center;
  gap: 0.75rem;
  border-radius: 1.25rem;
  padding: 2rem;
  text-align: center;
}

.panel-icon {
  width: 4rem;
  height: 4rem;
  border-radius: 1.25rem;
}

.purchase-panel strong {
  color: var(--ssxz-text-primary);
  font-size: 1.1rem;
  font-weight: 850;
}

.purchase-panel p {
  max-width: 36rem;
  margin: 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.9rem;
  line-height: 1.7;
}

.purchase-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.65rem;
  margin-top: 0.45rem;
}

.purchase-action {
  display: inline-flex;
  min-height: 2.35rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
  font-size: 0.84rem;
  font-weight: 850;
  padding: 0 0.9rem;
}

.purchase-action:hover {
  border-color: var(--ssxz-border-strong);
  color: var(--ssxz-text-primary);
}

.purchase-action.primary {
  border-color: transparent;
  background: var(--ssxz-action);
  color: var(--ssxz-action-text);
}

@media (max-width: 860px) {
  .purchase-summary-grid {
    grid-template-columns: 1fr;
  }
}
</style>
