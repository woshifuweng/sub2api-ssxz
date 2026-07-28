<template>
  <AppSectionShell
    title="补充额度 / 订阅"
    subtitle="按当前账号可用方式补充额度，账户变化以账户记录为准。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <div class="purchase-workbench">
      <PurchaseSubscriptionView v-if="legacyPurchaseEnabled && !paymentEnabled" embedded />
      <div v-else-if="!paymentEnabled" class="recharge-disabled-shell">
        <section class="recharge-flow" aria-label="充值流程">
          <div class="recharge-flow__step">
            <span class="recharge-flow__number">1</span>
            <ShoppingCart class="recharge-flow__icon" aria-hidden="true" />
            <div>
              <strong>选择面值购买</strong>
              <span>按需要选择充值码</span>
            </div>
          </div>
          <ChevronRight class="recharge-flow__arrow" aria-hidden="true" />
          <div class="recharge-flow__step">
            <span class="recharge-flow__number">2</span>
            <TicketCheck class="recharge-flow__icon" aria-hidden="true" />
            <div>
              <strong>收到兑换码</strong>
              <span>购买后获取卡密</span>
            </div>
          </div>
          <ChevronRight class="recharge-flow__arrow" aria-hidden="true" />
          <div class="recharge-flow__step">
            <span class="recharge-flow__number">3</span>
            <KeyRound class="recharge-flow__icon" aria-hidden="true" />
            <div>
              <strong>在兑换页输入到账</strong>
              <span>额度进入当前账户</span>
            </div>
          </div>
        </section>

        <section class="recharge-plans" aria-label="充值面值">
          <a
            v-for="plan in rechargePlans"
            :key="plan.amount"
            class="recharge-plan"
            :class="{ 'recharge-plan--featured': plan.featured }"
            :href="plan.href"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span v-if="plan.featured" class="recharge-plan__bar" aria-hidden="true" />
            <span v-if="plan.featured" class="recharge-plan__badge">推荐</span>
            <span v-else-if="plan.promotion" class="recharge-plan__badge recharge-plan__badge--promotion">
              {{ plan.promotion }}
            </span>
            <span class="recharge-plan__amount">¥{{ plan.amount }}</span>
            <strong class="recharge-plan__title">{{ plan.title }}</strong>
            <span class="recharge-plan__description">{{ plan.description }}</span>
            <span v-if="plan.promotionDetail" class="recharge-plan__promotion-detail">
              {{ plan.promotionDetail }}
            </span>
            <span class="recharge-plan__action">
              前往购买
              <ExternalLink aria-hidden="true" />
            </span>
          </a>
        </section>

        <section class="recharge-redeem-link" aria-label="兑换入口">
          <div>
            <strong>已有兑换码？</strong>
            <span>直接输入兑换码，额度到账后可在账户记录查看。</span>
          </div>
          <RouterLink to="/app/redeem" class="recharge-secondary-action">
            去兑换
            <ArrowRight aria-hidden="true" />
          </RouterLink>
        </section>

        <section class="recharge-faq" aria-labelledby="recharge-faq-title">
          <h2 id="recharge-faq-title">常见问题</h2>
          <details>
            <summary>余额有有效期吗？</summary>
            <p>无有效期，充值后永久有效。</p>
          </details>
          <details>
            <summary>支持哪些模型？</summary>
            <p>支持 Claude、GPT、Gemini 等主流模型，具体可用模型以模型价格页为准。</p>
          </details>
          <details>
            <summary>可以退款吗？</summary>
            <p>虚拟商品不支持退款，请按需购买。</p>
          </details>
        </section>
      </div>
      <PaymentCheckoutContent v-else variant="workspace" />
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { ArrowRight, ChevronRight, ExternalLink, KeyRound, ShoppingCart, TicketCheck } from '@lucide/vue'
import { useAppStore } from '@/stores'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import PaymentCheckoutContent from './PaymentCheckoutContent.vue'
import PurchaseSubscriptionView from './PurchaseSubscriptionView.vue'

const appStore = useAppStore()
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
const legacyPurchaseEnabled = computed(() => !!appStore.cachedPublicSettings?.purchase_subscription_enabled)

const purchaseShopUrl = 'https://pay.ldxp.cn/shop/VT7XKDFI'

const rechargePlans = computed(() => [
  {
    amount: 10,
    title: '入门体验',
    description: '适合轻度尝鲜',
    href: purchaseShopUrl,
    featured: false,
  },
  {
    amount: 30,
    title: '日常使用',
    description: '最受欢迎',
    href: purchaseShopUrl,
    featured: true,
  },
  {
    amount: 100,
    title: '重度使用',
    description: '余额更充裕',
    promotion: '赠$10',
    promotionDetail: '到账 $110 额度',
    href: purchaseShopUrl,
    featured: false,
  },
])
</script>

<style scoped>
.purchase-workbench {
  display: grid;
  gap: 1.5rem;
}

.purchase-empty-state {
  display: grid;
  min-height: 22rem;
  place-items: center;
  align-content: center;
  gap: 0.75rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
  padding: 2rem;
  text-align: center;
}

.purchase-empty-state__icon {
  display: grid;
  width: 3.75rem;
  height: 3.75rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
}

.purchase-empty-state h2 {
  margin: 0;
  color: var(--ssxz-text);
  font-size: 1.125rem;
  font-weight: 700;
}

.purchase-empty-state p {
  margin: 0;
  color: var(--ssxz-text-muted);
}

.purchase-workbench :deep(.payment-checkout .card),
.purchase-workbench :deep(.checkout-overview),
.purchase-workbench :deep(.purchase-page-layout .card) {
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.purchase-workbench :deep(.payment-checkout) {
  gap: 1.5rem;
}

.recharge-disabled-shell {
  display: grid;
  gap: 24px;
  max-width: 1120px;
  margin: 0 auto;
  padding: 8px 0 40px;
  color: var(--ssxz-text);
}

.recharge-flow {
  display: grid;
  grid-template-columns: 1fr auto 1fr auto 1fr;
  align-items: center;
  gap: 16px;
  padding: 18px 24px;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card, 10px);
  background: transparent;
}

.recharge-flow__step {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.recharge-flow__step > div {
  display: grid;
  gap: 3px;
}

.recharge-flow__step strong {
  font-size: 14px;
  font-weight: 650;
}

.recharge-flow__step span:not(.recharge-flow__number) {
  color: var(--ssxz-text-secondary);
  font-size: 12px;
}

.recharge-flow__number {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  place-items: center;
  border: 1px solid var(--ssxz-border-strong);
  border-radius: 999px;
  color: var(--ssxz-text-secondary);
  font-size: 13px;
  font-weight: 650;
}

.recharge-flow__icon {
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  color: var(--ssxz-text-secondary);
}

.recharge-flow__arrow {
  width: 17px;
  height: 17px;
  color: var(--ssxz-text-subtle);
}

.recharge-plans {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 20px;
}

.recharge-plan {
  position: relative;
  display: flex;
  min-height: 268px;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  overflow: hidden;
  padding: 32px;
  border: 1px solid var(--ssxz-border);
  border-radius: 16px;
  background: transparent;
  color: inherit;
  text-decoration: none;
  transition: border-color 160ms ease, background-color 160ms ease, transform 160ms ease;
}

.recharge-plan:hover {
  border-color: var(--ssxz-border-strong);
  background: var(--ssxz-primary-soft);
  transform: translateY(-1px);
}

.recharge-plan:focus-visible,
.recharge-secondary-action:focus-visible {
  outline: none;
  box-shadow: var(--ssxz-focus-ring);
}

.recharge-plan--featured {
  border-color: var(--ssxz-border-strong);
}

.recharge-plan__bar {
  position: absolute;
  inset: 0 0 auto;
  height: 4px;
  background: var(--ssxz-text-primary, var(--ssxz-text));
}

.recharge-plan__badge {
  position: absolute;
  top: 18px;
  right: 20px;
  color: var(--ssxz-text-secondary);
  font-size: 12px;
  font-weight: 650;
}

.recharge-plan__badge--promotion {
  padding: 4px 8px;
  border: 1px solid color-mix(in srgb, var(--ssxz-warning) 35%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-warning) 12%, transparent);
  color: var(--ssxz-warning);
}

.recharge-plan__amount {
  margin-top: 4px;
  color: var(--ssxz-text);
  font-size: 38px;
  font-weight: 750;
  letter-spacing: -0.02em;
  line-height: 1;
}

.recharge-plan__title {
  color: var(--ssxz-text);
  font-size: 16px;
  font-weight: 650;
}

.recharge-plan__description {
  color: var(--ssxz-text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.recharge-plan__promotion-detail {
  color: var(--ssxz-warning);
  font-size: 13px;
  font-weight: 600;
}

.recharge-plan__action {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: auto;
  padding: 10px 14px;
  border: 1px solid var(--ssxz-text-primary, var(--ssxz-text));
  border-radius: 8px;
  background: var(--ssxz-text-primary, var(--ssxz-text));
  color: var(--ssxz-action-text);
  font-size: 14px;
  font-weight: 650;
  box-shadow: var(--ssxz-shadow-button);
  transition: background-color 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
}

.recharge-plan:hover .recharge-plan__action {
  border-color: var(--ssxz-text-secondary);
  background: var(--ssxz-text-secondary);
  box-shadow: var(--ssxz-shadow-button-hover);
}

.recharge-plan__action svg,
.recharge-secondary-action svg {
  width: 16px;
  height: 16px;
}

.recharge-redeem-link {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px;
  border-top: 1px solid var(--ssxz-border);
  border-bottom: 1px solid var(--ssxz-border);
}

.recharge-redeem-link > div {
  display: grid;
  gap: 5px;
}

.recharge-redeem-link strong {
  font-size: 15px;
  font-weight: 650;
}

.recharge-redeem-link span {
  color: var(--ssxz-text-secondary);
  font-size: 13px;
}

.recharge-secondary-action {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 7px;
  padding: 9px 14px;
  border: 1px solid var(--ssxz-border-strong);
  border-radius: var(--ssxz-radius-button, 8px);
  color: var(--ssxz-text);
  font-size: 13px;
  font-weight: 650;
  text-decoration: none;
}

.recharge-secondary-action:hover {
  background: var(--ssxz-primary-soft);
}

.recharge-faq {
  display: grid;
  gap: 8px;
}

.recharge-faq h2 {
  margin: 0 0 4px;
  color: var(--ssxz-text);
  font-size: 18px;
  font-weight: 700;
}

.recharge-faq details {
  border-bottom: 1px solid var(--ssxz-border);
}

.recharge-faq summary {
  cursor: pointer;
  padding: 14px 2px;
  color: var(--ssxz-text);
  font-size: 14px;
  font-weight: 600;
  list-style-position: inside;
}

.recharge-faq p {
  margin: -2px 0 14px 20px;
  color: var(--ssxz-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

@media (max-width: 760px) {
  .recharge-disabled-shell {
    gap: 18px;
    padding: 0 0 28px;
  }

  .recharge-flow {
    grid-template-columns: 1fr;
    gap: 14px;
    padding: 18px;
  }

  .recharge-flow__arrow {
    display: none;
  }

  .recharge-plans {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .recharge-plan {
    min-height: 220px;
    padding: 26px 24px;
  }

  .recharge-redeem-link {
    align-items: flex-start;
    flex-direction: column;
    padding: 18px 2px;
  }
}
</style>
