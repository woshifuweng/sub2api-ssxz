<template>
  <AppSectionShell
    title="订单记录"
    subtitle="查看自己的充值、订阅和支付状态，方便核对每笔订单。"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <section class="orders-workbench" aria-label="订单记录">
      <div class="orders-summary-grid">
        <article class="orders-summary-card">
          <div class="summary-icon">
            <Icon name="creditCard" size="sm" />
          </div>
          <div>
            <span>账户余额</span>
            <strong>{{ balanceText }}</strong>
            <p>余额可用于站内聊天、图片生成和 API Key / 第三方接入调用。</p>
          </div>
          <RouterLink to="/app/purchase" class="summary-action">充值</RouterLink>
        </article>

        <article class="orders-summary-card">
          <div class="summary-icon">
            <Icon name="chartBar" size="sm" />
          </div>
          <div>
            <span>订单数量</span>
            <strong>{{ orderCountText }}</strong>
            <p>这里汇总你的充值和订阅订单，暂无记录时会显示空状态。</p>
          </div>
        </article>
      </div>

      <section class="orders-panel">
        <header class="panel-heading">
          <div>
            <h3>订单明细</h3>
            <p>查看订单金额、支付方式和当前状态。</p>
          </div>
          <button type="button" class="refresh-button" :disabled="loading" @click="loadOrders">
            <Icon name="refresh" size="xs" />
            刷新
          </button>
        </header>

        <div v-if="!paymentEnabled" class="orders-empty">
          <Icon name="creditCard" size="lg" />
          <strong>充值 / 订阅暂未开启</strong>
          <span>管理员暂未开启充值或订阅功能，请稍后再试或联系管理员。</span>
          <RouterLink to="/app/purchase" class="empty-action">查看充值说明</RouterLink>
        </div>

        <div v-else-if="loading" class="orders-empty compact">
          <Icon name="sync" size="md" />
          <strong>正在加载订单</strong>
        </div>

        <div v-else-if="loadError" class="orders-empty compact">
          <Icon name="exclamationTriangle" size="md" />
          <strong>{{ loadError }}</strong>
          <span>请稍后重试，或联系管理员协助查询。</span>
        </div>

        <div v-else-if="orders.length === 0" class="orders-empty compact">
          <Icon name="inbox" size="md" />
          <strong>暂无订单记录</strong>
          <span>完成充值或购买订阅后，订单会显示在这里。</span>
        </div>

        <div v-else class="orders-table-wrap">
          <table class="orders-table">
            <thead>
              <tr>
                <th>创建时间</th>
                <th>类型</th>
                <th>金额</th>
                <th>支付方式</th>
                <th>状态</th>
                <th>订单号</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="order in orders" :key="order.id">
                <td>{{ formatDateTime(order.created_at) }}</td>
                <td>{{ formatOrderType(order.order_type) }}</td>
                <td>{{ formatOrderAmount(order) }}</td>
                <td>{{ formatPaymentType(order.payment_type) }}</td>
                <td><OrderStatusBadge :status="order.status" /></td>
                <td class="order-no">{{ order.out_trade_no || `#${order.id}` }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </section>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import { paymentAPI } from '@/api/payment'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import type { PaymentOrder } from '@/types/payment'

const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(false)
const loadError = ref('')
const orders = ref<PaymentOrder[]>([])
const totalOrders = ref(0)

const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
const balanceText = computed(() => `$${Number(authStore.user?.balance || 0).toFixed(2)}`)
const orderCountText = computed(() => String(totalOrders.value || orders.value.length))

let ordersBootstrapped = false

watch(paymentEnabled, (enabled) => {
  if (!enabled || ordersBootstrapped) return
  ordersBootstrapped = true
  void loadOrders()
}, { immediate: true })

async function loadOrders() {
  if (!paymentEnabled.value) return
  loading.value = true
  loadError.value = ''
  try {
    const response = await paymentAPI.getMyOrders({ page: 1, page_size: 10 })
    orders.value = Array.isArray(response.data.items) ? response.data.items : []
    totalOrders.value = Number(response.data.total || orders.value.length || 0)
  } catch {
    orders.value = []
    totalOrders.value = 0
    loadError.value = '订单记录暂时无法加载'
  } finally {
    loading.value = false
  }
}

function formatDateTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatOrderType(type: PaymentOrder['order_type']) {
  if (type === 'subscription') return '订阅'
  return '余额充值'
}

function formatOrderAmount(order: PaymentOrder) {
  const amount = Number(order.pay_amount || order.amount || 0)
  return `¥${amount.toFixed(2)}`
}

function formatPaymentType(type: string) {
  const normalized = String(type || '').toLowerCase()
  if (normalized.includes('alipay')) return '支付宝'
  if (normalized.includes('wxpay') || normalized.includes('wechat')) return '微信支付'
  if (normalized.includes('stripe')) return 'Stripe'
  if (normalized.includes('easypay')) return '易支付'
  return type || '-'
}
</script>

<style scoped>
.orders-workbench {
  display: grid;
  gap: 1rem;
}

.orders-summary-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.orders-summary-card,
.orders-panel {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface);
  box-shadow: var(--ssxz-shadow);
}

.orders-summary-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.85rem;
  align-items: center;
  border-radius: 1.25rem;
  padding: 1rem;
}

.summary-icon {
  display: grid;
  width: 2.45rem;
  height: 2.45rem;
  place-items: center;
  border-radius: 0.85rem;
  background: color-mix(in srgb, var(--ssxz-action-soft) 78%, transparent);
  color: var(--ssxz-action);
}

.orders-summary-card span {
  color: var(--ssxz-text-muted);
  font-size: 0.82rem;
  font-weight: 800;
}

.orders-summary-card strong {
  display: block;
  margin-top: 0.15rem;
  color: var(--ssxz-text-primary);
  font-size: clamp(1.45rem, 3vw, 2rem);
  letter-spacing: 0;
}

.orders-summary-card p {
  margin: 0.28rem 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  line-height: 1.55;
}

.summary-action,
.empty-action {
  display: inline-flex;
  min-height: 2.35rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--ssxz-action);
  color: var(--ssxz-action-text);
  font-size: 0.84rem;
  font-weight: 850;
  padding: 0 0.9rem;
  text-decoration: none;
}

.orders-panel {
  display: grid;
  gap: 1rem;
  border-radius: 1.25rem;
  padding: 1rem;
}

.panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.panel-heading h3 {
  margin: 0;
  color: var(--ssxz-text-primary);
  font-size: 1rem;
  font-weight: 850;
}

.panel-heading p {
  margin: 0.25rem 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.84rem;
  line-height: 1.55;
}

.refresh-button {
  display: inline-flex;
  min-height: 2.25rem;
  align-items: center;
  gap: 0.35rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 800;
  padding: 0 0.8rem;
}

.refresh-button:disabled {
  cursor: wait;
  opacity: 0.65;
}

.orders-empty {
  display: grid;
  min-height: 18rem;
  place-items: center;
  align-content: center;
  gap: 0.6rem;
  border: 1px dashed var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
  padding: 2rem;
  text-align: center;
}

.orders-empty.compact {
  min-height: 12rem;
}

.orders-empty strong {
  color: var(--ssxz-text-primary);
  font-size: 1rem;
  font-weight: 850;
}

.orders-empty span {
  max-width: 30rem;
  font-size: 0.86rem;
  line-height: 1.6;
}

.orders-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
}

.orders-table {
  width: 100%;
  min-width: 48rem;
  border-collapse: collapse;
}

.orders-table th,
.orders-table td {
  border-bottom: 1px solid var(--ssxz-border);
  color: var(--ssxz-text-secondary);
  font-size: 0.84rem;
  padding: 0.82rem 0.9rem;
  text-align: left;
  white-space: nowrap;
}

.orders-table th {
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-muted);
  font-weight: 850;
}

.orders-table tr:last-child td {
  border-bottom: 0;
}

.order-no {
  max-width: 14rem;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 860px) {
  .orders-summary-grid {
    grid-template-columns: 1fr;
  }

  .orders-summary-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .summary-action {
    grid-column: 1 / -1;
  }

  .panel-heading {
    display: grid;
  }
}
</style>
