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
          <div class="orders-toolbar">
            <label class="status-filter-label" for="app-order-status-filter">状态</label>
            <select
              id="app-order-status-filter"
              v-model="currentFilter"
              class="status-filter"
              data-testid="status-filter"
              :disabled="loading"
              @change="handleFilterChange"
            >
              <option v-for="option in statusFilters" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
            <button type="button" class="refresh-button" :disabled="loading" @click="loadOrders">
              <Icon name="refresh" size="xs" />
              刷新
            </button>
          </div>
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
                <th>操作</th>
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
                <td>
                  <div class="order-actions">
                    <button
                      v-if="order.status === 'PENDING'"
                      type="button"
                      class="text-action warning"
                      :data-testid="`cancel-order-${order.id}`"
                      @click="openCancelDialog(order.id)"
                    >
                      取消订单
                    </button>
                    <button
                      v-if="canRequestRefund(order)"
                      type="button"
                      class="text-action"
                      :data-testid="`request-refund-${order.id}`"
                      @click="openRefundDialog(order)"
                    >
                      申请退款
                    </button>
                    <span v-if="order.status !== 'PENDING' && !canRequestRefund(order)" class="muted-action">
                      -
                    </span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <footer v-if="paymentEnabled && totalOrders > 0" class="pagination-row">
          <span>第 {{ pagination.page }} / {{ totalPages }} 页，共 {{ totalOrders }} 条</span>
          <div>
            <button
              type="button"
              class="page-button"
              data-testid="prev-page"
              :disabled="loading || pagination.page <= 1"
              @click="handlePageChange(pagination.page - 1)"
            >
              上一页
            </button>
            <button
              type="button"
              class="page-button"
              data-testid="next-page"
              :disabled="loading || pagination.page >= totalPages"
              @click="handlePageChange(pagination.page + 1)"
            >
              下一页
            </button>
          </div>
        </footer>
      </section>
    </section>

    <template v-if="paymentEnabled">
      <BaseDialog :show="cancelTargetId !== null" title="确认取消订单" width="narrow" @close="cancelTargetId = null">
        <p class="dialog-copy">取消后该订单不会继续等待支付。如仍需充值，请重新创建订单。</p>
        <template #footer>
          <div class="dialog-actions">
            <button type="button" class="secondary-button" :disabled="actionLoading" @click="cancelTargetId = null">
              关闭
            </button>
            <button
              type="button"
              class="danger-button"
              data-testid="confirm-cancel-order"
              :disabled="actionLoading"
              @click="confirmCancel"
            >
              {{ actionLoading ? '处理中' : '确认取消' }}
            </button>
          </div>
        </template>
      </BaseDialog>

      <BaseDialog :show="refundTarget !== null" title="申请退款" @close="refundTarget = null">
        <div class="refund-dialog" v-if="refundTarget">
          <dl class="refund-summary">
            <div>
              <dt>订单号</dt>
              <dd>{{ refundTarget.out_trade_no || `#${refundTarget.id}` }}</dd>
            </div>
            <div>
              <dt>金额</dt>
              <dd>{{ formatOrderAmount(refundTarget) }}</dd>
            </div>
          </dl>
          <label class="refund-label" for="app-refund-reason">退款原因</label>
          <textarea
            id="app-refund-reason"
            v-model="refundReason"
            class="refund-textarea"
            data-testid="refund-reason"
            rows="3"
            placeholder="请填写退款原因，方便管理员处理。"
          />
        </div>
        <template #footer>
          <div class="dialog-actions">
            <button type="button" class="secondary-button" :disabled="actionLoading" @click="refundTarget = null">
              关闭
            </button>
            <button
              type="button"
              class="primary-button"
              data-testid="confirm-refund-request"
              :disabled="actionLoading || !refundReason.trim()"
              @click="confirmRefund"
            >
              {{ actionLoading ? '处理中' : '提交申请' }}
            </button>
          </div>
        </template>
      </BaseDialog>
    </template>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { paymentAPI } from '@/api/payment'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import type { PaymentOrder } from '@/types/payment'

const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(false)
const actionLoading = ref(false)
const loadError = ref('')
const orders = ref<PaymentOrder[]>([])
const totalOrders = ref(0)
const currentFilter = ref('')
const refundEligibleProviders = ref<Set<string>>(new Set())
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const pagination = reactive({ page: 1, page_size: 10, total: 0 })

const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
const balanceText = computed(() => `$${Number(authStore.user?.balance || 0).toFixed(2)}`)
const orderCountText = computed(() => String(totalOrders.value || orders.value.length))
const totalPages = computed(() => Math.max(1, Math.ceil(totalOrders.value / pagination.page_size)))
const statusFilters = [
  { value: '', label: '全部状态' },
  { value: 'PENDING', label: '待支付' },
  { value: 'COMPLETED', label: '已完成' },
  { value: 'FAILED', label: '失败' },
  { value: 'REFUNDED', label: '已退款' }
]

let ordersBootstrapped = false

watch(paymentEnabled, (enabled) => {
  if (!enabled || ordersBootstrapped) return
  ordersBootstrapped = true
  void loadOrders()
  void loadRefundEligibility()
}, { immediate: true })

async function loadOrders() {
  if (!paymentEnabled.value) return
  loading.value = true
  loadError.value = ''
  try {
    const params: { page: number; page_size: number; status?: string } = {
      page: pagination.page,
      page_size: pagination.page_size
    }
    if (currentFilter.value) params.status = currentFilter.value
    const response = await paymentAPI.getMyOrders(params)
    orders.value = Array.isArray(response.data.items) ? response.data.items : []
    totalOrders.value = Number(response.data.total || orders.value.length || 0)
    pagination.total = totalOrders.value
  } catch {
    orders.value = []
    totalOrders.value = 0
    pagination.total = 0
    loadError.value = '订单记录暂时无法加载'
  } finally {
    loading.value = false
  }
}

async function loadRefundEligibility() {
  if (!paymentEnabled.value) return
  try {
    const response = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(response.data.provider_instance_ids || [])
  } catch {
    refundEligibleProviders.value = new Set()
  }
}

function handleFilterChange() {
  pagination.page = 1
  void loadOrders()
}

function handlePageChange(page: number) {
  if (!paymentEnabled.value) return
  if (page < 1 || page > totalPages.value) return
  pagination.page = page
  void loadOrders()
}

function openCancelDialog(orderId: number) {
  cancelTargetId.value = orderId
}

async function confirmCancel() {
  if (!paymentEnabled.value || cancelTargetId.value === null) return
  actionLoading.value = true
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value)
    appStore.showSuccess('订单已取消')
    cancelTargetId.value = null
    await loadOrders()
  } catch {
    appStore.showError('取消订单失败，请稍后重试')
  } finally {
    actionLoading.value = false
  }
}

function openRefundDialog(order: PaymentOrder) {
  refundTarget.value = order
  refundReason.value = ''
}

async function confirmRefund() {
  if (!paymentEnabled.value || !refundTarget.value || !refundReason.value.trim()) return
  actionLoading.value = true
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, { reason: refundReason.value.trim() })
    appStore.showSuccess('退款申请已提交')
    refundTarget.value = null
    refundReason.value = ''
    await loadOrders()
  } catch {
    appStore.showError('退款申请提交失败，请稍后重试')
  } finally {
    actionLoading.value = false
  }
}

function canRequestRefund(order: PaymentOrder) {
  if (!paymentEnabled.value) return false
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
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
  const payAmount = Number(order.pay_amount || order.amount || 0)
  const accountAmount = Number(order.amount || 0)
  if (order.order_type === 'balance') {
    return `支付 ¥${payAmount.toFixed(2)}，到账 $${accountAmount.toFixed(2)} 额度`
  }
  return `支付 ¥${payAmount.toFixed(2)}`
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

.orders-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
}

.status-filter-label {
  color: var(--ssxz-text-muted);
  font-size: 0.78rem;
  font-weight: 850;
}

.status-filter {
  min-height: 2.25rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.7rem;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 750;
  padding: 0 0.65rem;
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
  min-width: 58rem;
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

.order-actions {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.text-action {
  border: 0;
  border-radius: 0.55rem;
  background: color-mix(in srgb, var(--ssxz-action-soft) 74%, transparent);
  color: var(--ssxz-action);
  cursor: pointer;
  font-size: 0.78rem;
  font-weight: 850;
  padding: 0.35rem 0.5rem;
}

.text-action.warning {
  background: color-mix(in srgb, #f59e0b 16%, transparent);
  color: #b45309;
}

.muted-action {
  color: var(--ssxz-text-muted);
}

.pagination-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
}

.pagination-row > div {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.page-button,
.secondary-button,
.danger-button,
.primary-button {
  min-height: 2.25rem;
  border-radius: 0.7rem;
  cursor: pointer;
  font-size: 0.82rem;
  font-weight: 850;
  padding: 0 0.8rem;
}

.page-button,
.secondary-button {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
}

.danger-button {
  border: 1px solid color-mix(in srgb, #dc2626 32%, transparent);
  background: color-mix(in srgb, #dc2626 12%, var(--ssxz-surface));
  color: #b91c1c;
}

.primary-button {
  border: 1px solid transparent;
  background: var(--ssxz-action);
  color: var(--ssxz-action-text);
}

.page-button:disabled,
.secondary-button:disabled,
.danger-button:disabled,
.primary-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.dialog-copy {
  margin: 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.9rem;
  line-height: 1.65;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.65rem;
}

.refund-dialog {
  display: grid;
  gap: 0.85rem;
}

.refund-summary {
  display: grid;
  gap: 0.45rem;
  margin: 0;
  border-radius: 0.85rem;
  background: var(--ssxz-surface-subtle);
  padding: 0.85rem;
}

.refund-summary div {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.refund-summary dt {
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
  font-weight: 850;
}

.refund-summary dd {
  margin: 0;
  color: var(--ssxz-text-primary);
  font-size: 0.82rem;
  font-weight: 850;
}

.refund-label {
  color: var(--ssxz-text-primary);
  font-size: 0.84rem;
  font-weight: 850;
}

.refund-textarea {
  width: 100%;
  resize: vertical;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.8rem;
  background: var(--ssxz-surface);
  color: var(--ssxz-text-primary);
  font-size: 0.9rem;
  line-height: 1.5;
  padding: 0.75rem;
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

  .orders-toolbar {
    justify-content: flex-start;
  }
}
</style>
