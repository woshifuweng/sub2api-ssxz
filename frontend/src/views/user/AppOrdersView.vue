<template>
  <AppSectionShell
    :title="shellTitle"
    :subtitle="shellSubtitle"
    eyebrow="账户计费"
    icon="creditCard"
  >
    <section class="orders-workbench" :aria-label="shellTitle">
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
          <RouterLink
            v-if="purchaseEnabled"
            to="/app/purchase"
            class="btn btn-primary summary-action"
            >充值 / 订阅</RouterLink
          >
          <RouterLink
            v-else
            to="/app/redeem"
            class="btn btn-primary summary-action"
            >兑换码</RouterLink
          >
        </article>

        <article class="orders-summary-card">
          <div class="summary-icon">
            <Icon name="chartBar" size="sm" />
          </div>
          <div>
            <span>记录数量</span>
            <strong>{{ orderCountText }}</strong>
            <p>{{ recordCountDescription }}</p>
          </div>
        </article>
      </div>

      <section class="orders-panel">
        <header class="panel-heading">
          <div>
            <h3>{{ paymentEnabled ? "记录明细" : "账单记录" }}</h3>
            <p>
              {{
                paymentEnabled
                  ? "查看金额、来源和当前状态。"
                  : "显示兑换码入账、返利转入的时间、内容和状态。"
              }}
            </p>
          </div>
          <div class="orders-toolbar">
            <template v-if="paymentEnabled">
              <label class="status-filter-label" for="app-order-status-filter"
                >状态</label
              >
              <select
                id="app-order-status-filter"
                v-model="currentFilter"
                class="status-filter"
                data-testid="status-filter"
                :disabled="loading"
                @change="handleFilterChange"
              >
                <option
                  v-for="option in statusFilters"
                  :key="option.value"
                  :value="option.value"
                >
                  {{ option.label }}
                </option>
              </select>
            </template>
            <LiquidButton v-else as="RouterLink" to="/app/redeem" size="sm"
              >去兑换</LiquidButton
            >
            <LiquidButton
              type="button"
              class="refresh-button"
              :disabled="loading || billingLoading"
              @click="handleRefresh"
              variant="outline"
              size="sm"
            >
              <Icon name="refresh" size="xs" />
              刷新
            </LiquidButton>
          </div>
        </header>

        <div v-if="!paymentEnabled" class="orders-disabled-note">
          <div class="orders-empty__icon">
            <Icon name="creditCard" size="lg" />
          </div>
          <strong>{{ disabledOrdersTitle }}</strong>
          <span>{{ disabledOrdersDescription }}</span>
          <RouterLink
            v-if="purchaseEnabled"
            to="/app/purchase"
            class="btn btn-primary btn-sm empty-action"
            >查看充值方式</RouterLink
          >
          <RouterLink
            v-else
            to="/app/redeem"
            class="btn btn-primary btn-sm empty-action"
            >使用兑换码</RouterLink
          >
        </div>

        <template v-if="!paymentEnabled">
          <div v-if="billingLoading" class="orders-empty compact">
            <div class="orders-empty__icon"><Icon name="sync" size="md" /></div>
            <strong>正在加载账单记录</strong>
          </div>

          <div v-else-if="billingLoadError" class="orders-empty compact">
            <div class="orders-empty__icon is-warning">
              <Icon name="exclamationTriangle" size="md" />
            </div>
            <strong>{{ billingLoadError }}</strong>
            <span
              >账单记录正在更新，请稍后刷新。如果刚完成兑换，入账可能需要一点时间。</span
            >
          </div>

          <div
            v-else-if="billingRecords.length === 0"
            class="orders-empty compact"
          >
            <div class="orders-empty__icon">
              <Icon name="inbox" size="md" />
            </div>
            <strong>暂无账单记录</strong>
            <span>使用兑换码补充额度或返利转入后，入账记录会显示在这里。</span>
            <div class="empty-actions">
              <RouterLink
                to="/app/redeem"
                class="btn btn-primary btn-sm empty-action"
                >去兑换</RouterLink
              >
            </div>
          </div>

          <template v-else>
            <div class="orders-table-wrap" data-testid="billing-records-table">
              <table class="orders-table">
                <thead>
                  <tr>
                    <th>入账时间</th>
                    <th>类型</th>
                    <th>内容</th>
                    <th>编号</th>
                    <th>备注</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in billingRecords" :key="item.id">
                    <td>
                      {{ formatDateTime(item.used_at || item.created_at) }}
                    </td>
                    <td>{{ formatRedeemType(item) }}</td>
                    <td>{{ formatRedeemValue(item) }}</td>
                    <td class="order-no">{{ item.code || "-" }}</td>
                    <td>{{ item.notes || "-" }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <Pagination
              v-if="billingPagination.total > 0"
              :page="billingPagination.page"
              :total="billingPagination.total"
              :page-size="billingPagination.page_size"
              @update:page="handleBillingPageChange"
              @update:pageSize="handleBillingPageSizeChange"
            />
          </template>

          <div
            v-if="loadError"
            class="orders-empty compact"
            data-testid="legacy-orders-error"
          >
            <div class="orders-empty__icon is-warning">
              <Icon name="exclamationTriangle" size="md" />
            </div>
            <strong>历史充值订单暂时无法加载</strong>
            <span>不影响上方的账单记录，点击「刷新」可重试。</span>
          </div>
          <h4 v-else-if="orders.length > 0" class="legacy-orders-heading">
            历史充值订单
          </h4>
        </template>

        <div v-if="paymentEnabled && loading" class="orders-empty compact">
          <div class="orders-empty__icon"><Icon name="sync" size="md" /></div>
          <strong>正在加载账户记录</strong>
        </div>

        <div
          v-else-if="paymentEnabled && loadError"
          class="orders-empty compact"
        >
          <div class="orders-empty__icon is-warning">
            <Icon name="exclamationTriangle" size="md" />
          </div>
          <strong>{{ loadError }}</strong>
          <span
            >账户记录正在更新，请稍后刷新。如果刚完成支付，到账可能需要一点时间。</span
          >
        </div>

        <div
          v-else-if="paymentEnabled && orders.length === 0"
          class="orders-empty compact"
        >
          <div class="orders-empty__icon"><Icon name="inbox" size="md" /></div>
          <strong>暂无账户记录</strong>
          <span>完成充值、兑换或额度调整后，记录会显示在这里。</span>
          <div class="empty-actions">
            <RouterLink
              v-if="purchaseEnabled"
              to="/app/purchase"
              class="btn btn-primary btn-sm empty-action"
              >补充额度</RouterLink
            >
            <RouterLink
              to="/app/redeem"
              class="btn btn-secondary btn-sm empty-action"
              >使用兑换码</RouterLink
            >
          </div>
        </div>

        <div v-else-if="orders.length > 0" class="orders-table-wrap">
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
                <td class="order-no">
                  {{ order.out_trade_no || `#${order.id}` }}
                </td>
                <td>
                  <div class="order-actions">
                    <LiquidButton
                      v-if="paymentEnabled && order.status === 'PENDING'"
                      type="button"
                      class="text-action warning"
                      :data-testid="`cancel-order-${order.id}`"
                      @click="openCancelDialog(order.id)"
                      variant="plain"
                      size="sm"
                    >
                      取消订单
                    </LiquidButton>
                    <LiquidButton
                      v-if="canRequestRefund(order)"
                      type="button"
                      class="text-action"
                      :data-testid="`request-refund-${order.id}`"
                      @click="openRefundDialog(order)"
                      variant="plain"
                      size="sm"
                    >
                      申请退款
                    </LiquidButton>
                    <span
                      v-if="
                        order.status !== 'PENDING' && !canRequestRefund(order)
                      "
                      class="muted-action"
                    >
                      -
                    </span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <footer v-if="totalOrders > 0" class="pagination-row">
          <span
            >第 {{ pagination.page }} / {{ totalPages }} 页，共
            {{ totalOrders }} 条</span
          >
          <div>
            <LiquidButton
              type="button"
              class="page-button"
              data-testid="prev-page"
              :disabled="loading || pagination.page <= 1"
              @click="handlePageChange(pagination.page - 1)"
              variant="plain"
              size="sm"
            >
              上一页
            </LiquidButton>
            <LiquidButton
              type="button"
              class="page-button"
              data-testid="next-page"
              :disabled="loading || pagination.page >= totalPages"
              @click="handlePageChange(pagination.page + 1)"
              variant="plain"
              size="sm"
            >
              下一页
            </LiquidButton>
          </div>
        </footer>
      </section>
    </section>

    <template v-if="paymentEnabled">
      <BaseDialog
        :show="cancelTargetId !== null"
        title="确认取消订单"
        width="narrow"
        @close="cancelTargetId = null"
      >
        <p class="dialog-copy">
          取消后该订单不会继续等待支付。如仍需充值，请重新创建订单。
        </p>
        <template #footer>
          <div class="dialog-actions">
            <LiquidButton
              type="button"
              :disabled="actionLoading"
              @click="cancelTargetId = null"
              variant="outline"
              size="sm"
            >
              关闭
            </LiquidButton>
            <LiquidButton
              type="button"
              data-testid="confirm-cancel-order"
              :disabled="actionLoading"
              @click="confirmCancel"
              variant="destructive"
              size="sm"
            >
              {{ actionLoading ? "处理中" : "确认取消" }}
            </LiquidButton>
          </div>
        </template>
      </BaseDialog>

      <BaseDialog
        :show="refundTarget !== null"
        title="申请退款"
        @close="refundTarget = null"
      >
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
            placeholder="请简单说明原因，方便核对订单。"
          />
        </div>
        <template #footer>
          <div class="dialog-actions">
            <LiquidButton
              type="button"
              :disabled="actionLoading"
              @click="refundTarget = null"
              variant="outline"
              size="sm"
            >
              关闭
            </LiquidButton>
            <LiquidButton
              type="button"
              data-testid="confirm-refund-request"
              :disabled="actionLoading || !refundReason.trim()"
              @click="confirmRefund"
              size="default"
            >
              {{ actionLoading ? "处理中" : "提交申请" }}
            </LiquidButton>
          </div>
        </template>
      </BaseDialog>
    </template>
  </AppSectionShell>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import { computed, reactive, ref, watch } from "vue";
import { RouterLink } from "vue-router";
import AppSectionShell from "@/components/user/AppSectionShell.vue";
import Icon from "@/components/icons/Icon.vue";
import OrderStatusBadge from "@/components/payment/OrderStatusBadge.vue";
import BaseDialog from "@/components/common/BaseDialog.vue";
import Pagination from "@/components/common/Pagination.vue";
import { paymentAPI } from "@/api/payment";
import userAPI from "@/api/user";
import type { RedeemHistoryItem } from "@/api/redeem";
import { useAppStore } from "@/stores";
import { useAuthStore } from "@/stores/auth";
import type { PaymentOrder } from "@/types/payment";

const appStore = useAppStore();
const authStore = useAuthStore();

const loading = ref(false);
const actionLoading = ref(false);
const loadError = ref("");
const orders = ref<PaymentOrder[]>([]);
const billingRecords = ref<RedeemHistoryItem[]>([]);
const billingLoading = ref(false);
const billingLoadError = ref("");
const billingPagination = reactive({ page: 1, page_size: 10, total: 0 });
const totalOrders = ref(0);
const currentFilter = ref("");
const refundEligibleProviders = ref<Set<string>>(new Set());
const cancelTargetId = ref<number | null>(null);
const refundTarget = ref<PaymentOrder | null>(null);
const refundReason = ref("");
const pagination = reactive({ page: 1, page_size: 10, total: 0 });

const paymentEnabled = computed(
  () => !!appStore.cachedPublicSettings?.payment_enabled,
);
const purchaseEnabled = computed(
  () =>
    paymentEnabled.value ||
    !!appStore.cachedPublicSettings?.purchase_subscription_enabled,
);
const balanceText = computed(
  () => `$${Number(authStore.user?.balance || 0).toFixed(2)}`,
);
const orderCountText = computed(() =>
  paymentEnabled.value
    ? String(totalOrders.value || orders.value.length)
    : String(billingPagination.total + totalOrders.value),
);
const shellTitle = computed(() =>
  paymentEnabled.value ? "账户记录" : "账单记录",
);
const shellSubtitle = computed(() =>
  paymentEnabled.value
    ? "查看充值、兑换、订单状态和账户额度变化。"
    : "查看兑换码入账、返利转入和账户额度变化。",
);
const recordCountDescription = computed(() =>
  paymentEnabled.value
    ? "这里会显示充值、兑换、订单、账户调整和退款相关记录。"
    : "这里会显示兑换码入账、返利转入和账户额度变化记录。",
);
const disabledOrdersTitle = computed(() =>
  purchaseEnabled.value ? "当前账号可查看已有账户记录" : "当前可用方式：兑换码",
);
const disabledOrdersDescription = computed(() =>
  purchaseEnabled.value
    ? "已有订单和账户变化会保留在账户记录中；如需充值，可先查看补充额度方式。"
    : "可继续使用已有额度，或通过兑换码补充账户额度；已有账户记录仍可在这里查看。",
);
const totalPages = computed(() =>
  Math.max(1, Math.ceil(totalOrders.value / pagination.page_size)),
);
const statusFilters = [
  { value: "", label: "全部状态" },
  { value: "PENDING", label: "待支付" },
  { value: "COMPLETED", label: "已完成" },
  { value: "FAILED", label: "失败" },
  { value: "REFUNDED", label: "已退款" },
];

let ordersBootstrapped = false;

// 等公共设置就绪后再按 payment_enabled 分流，避免设置未加载时误按「支付关闭」发起兑换记录请求
watch(
  [() => appStore.cachedPublicSettings, paymentEnabled] as const,
  ([settings, enabled]) => {
    if (!settings) return;
    if (!ordersBootstrapped) {
      ordersBootstrapped = true;
      void loadOrders();
    }
    if (enabled) {
      void loadRefundEligibility();
    } else {
      refundEligibleProviders.value = new Set();
      void loadBillingHistory();
    }
    syncDocumentTitle();
  },
  { immediate: true },
);

async function loadBillingHistory() {
  if (billingLoading.value) return;
  billingLoading.value = true;
  billingLoadError.value = "";
  try {
    const response = await userAPI.getBalanceHistory({
      page: billingPagination.page,
      page_size: billingPagination.page_size,
    });
    billingRecords.value = Array.isArray(response.items) ? response.items : [];
    billingPagination.total = Number(response.total || 0);
  } catch {
    billingRecords.value = [];
    billingPagination.total = 0;
    billingLoadError.value = "账单记录暂时无法加载";
  } finally {
    billingLoading.value = false;
  }
}

function handleBillingPageChange(page: number) {
  if (page < 1) return;
  billingPagination.page = page;
  void loadBillingHistory();
}

function handleBillingPageSizeChange(pageSize: number) {
  billingPagination.page_size = pageSize;
  billingPagination.page = 1;
  void loadBillingHistory();
}

function handleRefresh() {
  void loadOrders();
  if (!paymentEnabled.value) {
    void loadBillingHistory();
  }
}

// 路由 meta 的标签页标题固定为「我的订单」；账单记录模式下同步替换，避免页内文案与标签页不一致
function syncDocumentTitle() {
  if (typeof document === "undefined") return;
  const current = document.title;
  const sepIndex = current.indexOf(" - ");
  const suffix = sepIndex >= 0 ? current.slice(sepIndex) : "";
  document.title = shellTitle.value + suffix;
}

async function loadOrders() {
  loading.value = true;
  loadError.value = "";
  try {
    const params: { page: number; page_size: number; status?: string } = {
      page: pagination.page,
      page_size: pagination.page_size,
    };
    if (currentFilter.value) params.status = currentFilter.value;
    const response = await paymentAPI.getMyOrders(params);
    orders.value = Array.isArray(response.data.items)
      ? response.data.items
      : [];
    totalOrders.value = Number(response.data.total || orders.value.length || 0);
    pagination.total = totalOrders.value;
  } catch {
    orders.value = [];
    totalOrders.value = 0;
    pagination.total = 0;
    loadError.value = "账户记录暂时无法加载";
  } finally {
    loading.value = false;
  }
}

async function loadRefundEligibility() {
  if (!paymentEnabled.value) return;
  try {
    const response = await paymentAPI.getRefundEligibleProviders();
    refundEligibleProviders.value = new Set(
      response.data.provider_instance_ids || [],
    );
  } catch {
    refundEligibleProviders.value = new Set();
  }
}

function handleFilterChange() {
  pagination.page = 1;
  void loadOrders();
}

function handlePageChange(page: number) {
  if (page < 1 || page > totalPages.value) return;
  pagination.page = page;
  void loadOrders();
}

function openCancelDialog(orderId: number) {
  cancelTargetId.value = orderId;
}

async function confirmCancel() {
  if (!paymentEnabled.value || cancelTargetId.value === null) return;
  actionLoading.value = true;
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value);
    appStore.showSuccess("订单已取消");
    cancelTargetId.value = null;
    await loadOrders();
  } catch {
    appStore.showError("取消订单失败，请稍后重试");
  } finally {
    actionLoading.value = false;
  }
}

function openRefundDialog(order: PaymentOrder) {
  refundTarget.value = order;
  refundReason.value = "";
}

async function confirmRefund() {
  if (
    !paymentEnabled.value ||
    !refundTarget.value ||
    !refundReason.value.trim()
  )
    return;
  actionLoading.value = true;
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, {
      reason: refundReason.value.trim(),
    });
    appStore.showSuccess("退款申请已提交");
    refundTarget.value = null;
    refundReason.value = "";
    await loadOrders();
  } catch {
    appStore.showError("退款申请提交失败，请稍后重试");
  } finally {
    actionLoading.value = false;
  }
}

function canRequestRefund(order: PaymentOrder) {
  if (!paymentEnabled.value) return false;
  if (order.status !== "COMPLETED") return false;
  if (!order.provider_instance_id) return false;
  return refundEligibleProviders.value.has(order.provider_instance_id);
}

function formatDateTime(value: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatOrderType(type: PaymentOrder["order_type"]) {
  if (type === "subscription") return "订阅";
  return "余额充值";
}

function formatOrderAmount(order: PaymentOrder) {
  const payAmount = Number(order.pay_amount || order.amount || 0);
  const accountAmount = Number(order.amount || 0);
  if (order.order_type === "balance") {
    return `支付 ¥${payAmount.toFixed(2)}，到账 $${accountAmount.toFixed(2)} 额度`;
  }
  return `支付 ¥${payAmount.toFixed(2)}`;
}

function formatRedeemType(item: RedeemHistoryItem) {
  if (item.type === "balance") return "兑换码入账";
  if (item.type === "affiliate_balance") return "返利转入";
  if (item.type === "admin_balance")
    return item.value >= 0 ? "管理员调整（增加）" : "管理员调整（扣减）";
  if (item.type === "concurrency") return "并发数提升";
  if (item.type === "admin_concurrency")
    return item.value >= 0 ? "并发数调整（增加）" : "并发数调整（减少）";
  if (item.type === "subscription") return "订阅开通";
  return item.type || "-";
}

function formatRedeemValue(item: RedeemHistoryItem) {
  if (item.type === "subscription") {
    const groupName = item.group?.name || "订阅";
    return item.validity_days
      ? `${groupName}（${item.validity_days} 天）`
      : groupName;
  }
  if (item.type === "concurrency" || item.type === "admin_concurrency") {
    return `${item.value >= 0 ? "+" : ""}${item.value} 并发`;
  }
  const amount = Number(item.value || 0);
  return `${amount >= 0 ? "+" : "-"}$${Math.abs(amount).toFixed(2)} 额度`;
}

function formatPaymentType(type: string) {
  const normalized = String(type || "").toLowerCase();
  if (normalized.includes("alipay")) return "支付宝";
  if (normalized.includes("wxpay") || normalized.includes("wechat"))
    return "微信支付";
  if (normalized.includes("stripe")) return "Stripe";
  if (normalized.includes("easypay")) return "易支付";
  return type || "-";
}
</script>

<style scoped>
.orders-workbench {
  display: grid;
  gap: 1.5rem;
}

.orders-summary-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.orders-summary-card,
.orders-panel {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.orders-summary-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.85rem;
  align-items: center;
  border-radius: var(--ssxz-radius-card);
  padding: 1.25rem;
}

.summary-icon {
  display: grid;
  width: 2.45rem;
  height: 2.45rem;
  place-items: center;
  border-radius: var(--ssxz-radius-button);
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
  text-decoration: none;
}

.empty-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.55rem;
}

.orders-panel {
  display: grid;
  gap: 1.25rem;
  border-radius: var(--ssxz-radius-card);
  padding: 1.25rem;
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
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-surface);
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 750;
  padding: 0 0.65rem;
}

.refresh-button {
  flex: 0 0 auto;
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
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface);
  color: var(--ssxz-text-secondary);
  padding: 2rem;
  text-align: center;
}

.orders-empty.compact {
  min-height: 12rem;
}

.orders-disabled-note {
  display: grid;
  gap: 0.45rem;
  border: 1px solid
    color-mix(in srgb, var(--ssxz-action) 26%, var(--ssxz-border));
  border-radius: var(--ssxz-radius-card);
  background: color-mix(
    in srgb,
    var(--ssxz-action-soft) 45%,
    var(--ssxz-surface)
  );
  color: var(--ssxz-text-secondary);
  padding: 0.95rem 1rem;
}

.orders-disabled-note strong {
  color: var(--ssxz-text-primary);
  font-size: 0.92rem;
  font-weight: 850;
}

.orders-disabled-note span {
  max-width: 42rem;
  font-size: 0.84rem;
  line-height: 1.55;
}

.orders-disabled-note .empty-action {
  justify-self: start;
}

.legacy-orders-heading {
  margin: 0.25rem 0 -0.5rem;
  color: var(--ssxz-text-muted);
  font-size: 0.84rem;
  font-weight: 850;
}

.orders-empty__icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
}

.orders-empty__icon.is-warning {
  background: color-mix(in srgb, var(--ssxz-warning) 12%, var(--ssxz-surface));
  color: var(--ssxz-warning);
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
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface);
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
