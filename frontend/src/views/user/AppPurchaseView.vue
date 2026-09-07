<template>
  <AppSectionShell
    title="补充额度 / 订阅"
    subtitle="购买充值码后，在兑换页输入卡密，额度会进入当前账户。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div class="purchase-workbench">
      <section class="purchase-card" aria-labelledby="purchase-card-title">
        <div class="purchase-card__content">
          <span class="purchase-card__eyebrow">充值中心</span>
          <h2 id="purchase-card-title">选择充值面值</h2>
          <p>
            充值中心将在新窗口打开。完成购买后，回到这里进入兑换页，输入兑换码即可到账。
          </p>
        </div>
        <div class="purchase-card__actions">
          <a
            class="purchase-primary-action"
            href="https://pay.ldxp.cn/shop/VT7XKDFI"
            target="_blank"
            rel="noopener noreferrer"
            referrerpolicy="no-referrer"
          >
            打开充值中心
          </a>
          <RouterLink class="purchase-secondary-action" to="/app/redeem">
            已有兑换码，去兑换
          </RouterLink>
        </div>
      </section>

      <section class="purchase-steps" aria-label="充值流程">
        <div>
          <strong>1. 购买充值码</strong>
          <span>选择需要的充值面值并完成购买。</span>
        </div>
        <div>
          <strong>2. 输入兑换码</strong>
          <span>在兑换页输入卡密，提交后自动到账。</span>
        </div>
        <div>
          <strong>3. 查看账户记录</strong>
          <span>额度和消费明细以账户记录为准。</span>
        </div>
      </section>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
</script>

<style scoped>
.purchase-workbench {
  display: grid;
  gap: 1rem;
  max-width: 1120px;
  margin: 0 auto;
}

.purchase-card,
.purchase-steps {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card, 10px);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.purchase-card {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1.5rem;
  padding: clamp(1.25rem, 3vw, 2rem);
}

.purchase-card__content {
  max-width: 40rem;
}

.purchase-card__eyebrow {
  color: var(--ssxz-text-secondary);
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.purchase-card h2 {
  margin: 0.4rem 0 0;
  color: var(--ssxz-text);
  font-size: clamp(1.25rem, 2vw, 1.6rem);
}

.purchase-card p {
  margin: 0.75rem 0 0;
  color: var(--ssxz-text-secondary);
  line-height: 1.6;
}

.purchase-card__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.65rem;
}

.purchase-primary-action,
.purchase-secondary-action {
  display: inline-flex;
  min-height: 2.5rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.55rem;
  padding: 0.65rem 0.95rem;
  font-size: 0.88rem;
  font-weight: 750;
  text-decoration: none;
  transition: background-color 160ms ease, border-color 160ms ease;
}

.purchase-primary-action {
  border: 1px solid var(--ssxz-text);
  background: var(--ssxz-text);
  color: var(--ssxz-surface-raised);
}

.purchase-primary-action:hover {
  background: var(--ssxz-text-secondary);
  border-color: var(--ssxz-text-secondary);
}

.purchase-secondary-action {
  border: 1px solid var(--ssxz-border-strong);
  color: var(--ssxz-text);
}

.purchase-secondary-action:hover {
  background: var(--ssxz-primary-soft);
}

.purchase-primary-action:focus-visible,
.purchase-secondary-action:focus-visible {
  outline: none;
  box-shadow: var(--ssxz-focus-ring);
}

.purchase-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
  padding: 1.25rem;
}

.purchase-steps div {
  display: grid;
  gap: 0.35rem;
  padding: 0.35rem 0.5rem;
}

.purchase-steps strong {
  color: var(--ssxz-text);
  font-size: 0.9rem;
}

.purchase-steps span {
  color: var(--ssxz-text-secondary);
  font-size: 0.8rem;
  line-height: 1.5;
}

@media (max-width: 760px) {
  .purchase-card {
    align-items: stretch;
    flex-direction: column;
  }

  .purchase-card__actions {
    justify-content: flex-start;
  }

  .purchase-steps {
    grid-template-columns: 1fr;
  }
}
</style>
