<template>
  <BaseDialog
    :show="show"
    title="Agent 详情"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="loading" class="py-12 text-center text-sm text-[var(--ssxz-text-muted)]">
      正在加载 Agent 详情
    </div>
    <div v-else-if="agent" class="space-y-5">
      <header class="detail-header">
        <div class="min-w-0">
          <h4 class="truncate text-base font-semibold text-[var(--ssxz-text)]">
            {{ agent.username || `用户 ${agent.user_id}` }}
          </h4>
          <p class="truncate text-sm text-[var(--ssxz-text-muted)]">{{ agent.email }}</p>
        </div>
        <span :class="['status-badge', `status-badge--${agent.status}`]">
          {{ statusLabel(agent.status) }}
        </span>
      </header>

      <nav class="detail-tabs" aria-label="Agent 详情分组">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          :class="['detail-tab', { 'detail-tab--active': activeTab === tab.value }]"
          @click="activeTab = tab.value"
        >
          {{ tab.label }}
        </button>
      </nav>

      <section v-if="activeTab === 'overview'" class="space-y-6">
        <section>
          <h5 class="section-title">合作配置</h5>
          <dl class="detail-grid">
            <div><dt>角色</dt><dd>{{ roleLabel(agent.role) }}</dd></div>
            <div><dt>上级 Manager</dt><dd>{{ agent.manager_email || '未设置' }}</dd></div>
            <div><dt>返利策略</dt><dd>{{ rebateLabel(agent) }}</dd></div>
            <div><dt>邀请码</dt><dd>{{ agent.aff_code || '--' }}</dd></div>
            <div class="md:col-span-2"><dt>备注</dt><dd>{{ agent.notes || '无' }}</dd></div>
          </dl>
        </section>

        <section>
          <h5 class="section-title">业务数据</h5>
          <dl class="detail-grid detail-grid--metrics">
            <div><dt>招募人数</dt><dd>{{ agent.recruit_count }}</dd></div>
            <div><dt>可兑换佣金</dt><dd>${{ formatMoney(agent.commission_balance) }}</dd></div>
            <div><dt>累计佣金</dt><dd>${{ formatMoney(agent.commission_total) }}</dd></div>
            <div><dt>待处理兑换</dt><dd>{{ agent.pending_redemption_count }}</dd></div>
          </dl>
        </section>

        <section>
          <h5 class="section-title">生命周期</h5>
          <dl class="detail-grid">
            <div><dt>授权时间</dt><dd>{{ formatDate(agent.granted_at) }}</dd></div>
            <div><dt>最近更新</dt><dd>{{ formatDate(agent.updated_at) }}</dd></div>
            <div v-if="agent.disabled_at">
              <dt>停用时间</dt><dd>{{ formatDate(agent.disabled_at) }}</dd>
            </div>
            <div v-if="agent.disabled_by_email">
              <dt>停用操作人</dt><dd>{{ agent.disabled_by_email }}</dd>
            </div>
            <div v-if="agent.disabled_reason" class="md:col-span-2">
              <dt>停用原因</dt><dd>{{ agent.disabled_reason }}</dd>
            </div>
            <div v-if="agent.revoked_at">
              <dt>最终撤销时间</dt><dd>{{ formatDate(agent.revoked_at) }}</dd>
            </div>
          </dl>
        </section>
      </section>

      <section v-else-if="activeTab === 'recruits'" class="space-y-4">
        <div class="summary-grid">
          <div class="summary-card"><span>下线总人数</span><strong>{{ recruits.total }}</strong></div>
          <div class="summary-card"><span>本页充值合计</span><strong>${{ formatMoney(recruitSummary.recharge) }}</strong></div>
          <div class="summary-card"><span>本页消费合计</span><strong>${{ formatMoney(recruitSummary.cost) }}</strong></div>
          <div class="summary-card"><span>本页贡献佣金</span><strong>${{ formatMoney(recruitSummary.commission) }}</strong></div>
        </div>

        <div v-if="recruitsError" class="empty-state text-[var(--ssxz-danger)]">
          {{ recruitsError }}
        </div>
        <DataTable
          v-else
          :columns="recruitColumns"
          :data="recruits.items"
          :loading="recruitsLoading"
          row-key="user_id"
          default-sort-key="created_at"
          default-sort-order="desc"
        >
          <template #empty>
            <div class="py-12 text-center text-gray-500 dark:text-dark-400">
              暂无下线代理
            </div>
          </template>

          <template #cell-email="{ value }">
            <div class="flex items-center gap-2">
              <div class="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
                <span class="text-sm font-medium text-primary-700 dark:text-primary-300">
                  {{ String(value || '').charAt(0).toUpperCase() }}
                </span>
              </div>
              <span class="font-medium text-gray-900 dark:text-white">{{ value || '-' }}</span>
            </div>
          </template>

          <template #cell-user_id="{ value }">
            <button
              type="button"
              class="font-mono text-sm text-gray-700 underline decoration-dashed decoration-gray-300 underline-offset-4 hover:text-primary-600 dark:text-gray-300 dark:decoration-dark-500 dark:hover:text-primary-400"
              :title="`复制用户ID ${value}`"
              @click="copyRecruitId(value)"
            >
              #{{ value }}
            </button>
          </template>

          <template #cell-status="{ value }">
            <div class="flex items-center gap-1.5">
              <span
                :class="[
                  'inline-block h-2 w-2 rounded-full',
                  value === 'active' ? 'bg-green-500' : 'bg-red-500'
                ]"
              ></span>
              <span class="text-sm text-gray-700 dark:text-gray-300">
                {{ value === 'active' ? 'Active' : 'Disabled' }}
              </span>
            </div>
          </template>

          <template #cell-total_recharge_usd="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ formatCurrency(value) }}</span>
          </template>

          <template #cell-total_consumption_usd="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ formatCurrency(value) }}</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ value ? formatDateTime(value) : '-' }}</span>
          </template>
        </DataTable>

        <div class="pagination-row">
          <span>第 {{ recruits.page }} / {{ recruits.pages || 1 }} 页，共 {{ recruits.total }} 人</span>
          <div class="pagination-actions">
            <LiquidButton type="button" variant="outline" size="sm" :disabled="recruitsLoading || recruits.page <= 1" @click="loadRecruits(recruits.page - 1)">
              上一页
            </LiquidButton>
            <LiquidButton type="button" variant="outline" size="sm" :disabled="recruitsLoading || !hasNextRecruits" @click="loadRecruits(recruits.page + 1)">
              下一页
            </LiquidButton>
          </div>
        </div>
      </section>

      <section v-else class="space-y-4">
        <div class="section-heading-row">
          <h5 class="section-title">提现记录</h5>
          <a class="subtle-link" href="/admin/reseller/withdrawals">查看全部提现</a>
        </div>

        <div v-if="withdrawalsLoading" class="py-10 text-center text-sm text-[var(--ssxz-text-muted)]">
          正在加载提现记录
        </div>
        <div v-else-if="withdrawalsError" class="empty-state text-[var(--ssxz-danger)]">
          {{ withdrawalsError }}
        </div>
        <div v-else-if="withdrawals.items.length === 0" class="empty-state">
          暂无提现记录
        </div>
        <div v-else class="table-scroll">
          <table class="detail-table">
            <thead>
              <tr>
                <th>申请时间</th>
                <th>金额</th>
                <th>方式</th>
                <th>状态</th>
                <th>审批时间</th>
                <th>备注</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="withdrawal in withdrawals.items" :key="withdrawal.id">
                <td>{{ formatDate(withdrawal.requested_at) }}</td>
                <td>${{ formatMoney(withdrawal.amount) }}</td>
                <td>余额转入</td>
                <td><span :class="['status-badge', `status-badge--${withdrawalStatusClass(withdrawal.status)}`]">{{ withdrawalStatusLabel(withdrawal.status) }}</span></td>
                <td>{{ formatDate(withdrawal.reviewed_at) }}</td>
                <td class="note-cell">{{ withdrawal.note || '--' }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="pagination-row">
          <span>第 {{ withdrawals.page }} / {{ withdrawals.pages || 1 }} 页，共 {{ withdrawals.total }} 条</span>
          <div class="pagination-actions">
            <LiquidButton type="button" variant="outline" size="sm" :disabled="withdrawalsLoading || withdrawals.page <= 1" @click="loadWithdrawals(withdrawals.page - 1)">
              上一页
            </LiquidButton>
            <LiquidButton type="button" variant="outline" size="sm" :disabled="withdrawalsLoading || !hasNextWithdrawals" @click="loadWithdrawals(withdrawals.page + 1)">
              下一页
            </LiquidButton>
          </div>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <LiquidButton type="button" variant="outline" size="sm" @click="emit('close')">
          <span>关闭</span>
        </LiquidButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type {
  AdminRecruitRecord,
  AgentDetail,
  AgentSummary,
  ResellerStatus,
  WithdrawRequest,
  WithdrawStatus
} from '@/api/reseller'
import type { Column } from '@/components/common/types'
import type { PaginatedResponse } from '@/types'
import resellerAPI from '@/api/reseller'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'

const props = defineProps<{
  show: boolean
  agent: AgentDetail | null
  loading: boolean
}>()

const emit = defineEmits<{ (event: 'close'): void }>()

type DetailTab = 'overview' | 'recruits' | 'withdrawals'

const tabs: Array<{ value: DetailTab; label: string }> = [
  { value: 'overview', label: '概览' },
  { value: 'recruits', label: '下线列表' },
  { value: 'withdrawals', label: '提现记录' }
]

const activeTab = ref<DetailTab>('overview')
const recruits = ref<PaginatedResponse<AdminRecruitRecord>>(emptyPage(20))
const withdrawals = ref<PaginatedResponse<WithdrawRequest>>(emptyPage(10))
const recruitsLoading = ref(false)
const withdrawalsLoading = ref(false)
const recruitsLoaded = ref(false)
const withdrawalsLoaded = ref(false)
const recruitsError = ref('')
const withdrawalsError = ref('')
const { copyToClipboard } = useClipboard()

const recruitColumns: Column[] = [
  { key: 'email', label: '邮箱', sortable: true },
  { key: 'user_id', label: 'ID', sortable: true },
  { key: 'status', label: '状态', sortable: true },
  { key: 'total_recharge_usd', label: '总充值', sortable: true },
  { key: 'total_consumption_usd', label: '总消费', sortable: true },
  { key: 'created_at', label: '注册时间', sortable: true }
]

const recruitSummary = computed(() =>
  recruits.value.items.reduce(
    (summary, recruit) => ({
      recharge: summary.recharge + Number(recruit.total_recharge_usd || 0),
      cost: summary.cost + Number(recruit.total_consumption_usd || 0),
      commission: summary.commission + Number(recruit.commission_contributed_usd || 0)
    }),
    { recharge: 0, cost: 0, commission: 0 }
  )
)

const hasNextRecruits = computed(
  () => recruits.value.pages > 0
    ? recruits.value.page < recruits.value.pages
    : recruits.value.items.length >= recruits.value.page_size
)
const hasNextWithdrawals = computed(
  () => withdrawals.value.pages > 0
    ? withdrawals.value.page < withdrawals.value.pages
    : withdrawals.value.items.length >= withdrawals.value.page_size
)

watch(
  () => props.agent?.user_id,
  () => {
    activeTab.value = 'overview'
    recruits.value = emptyPage(20)
    withdrawals.value = emptyPage(10)
    recruitsLoaded.value = false
    withdrawalsLoaded.value = false
    recruitsError.value = ''
    withdrawalsError.value = ''
  }
)

watch(activeTab, (tab) => {
  if (!props.show || !props.agent) return
  if (tab === 'recruits' && !recruitsLoaded.value) void loadRecruits(1)
  if (tab === 'withdrawals' && !withdrawalsLoaded.value) void loadWithdrawals(1)
})

function emptyPage<T>(pageSize: number): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: pageSize, pages: 0 }
}

async function loadRecruits(page: number): Promise<void> {
  if (!props.agent || recruitsLoading.value) return
  recruitsLoading.value = true
  recruitsError.value = ''
  try {
    recruits.value = await resellerAPI.listAdminAgentRecruits(props.agent.user_id, page, 20)
    recruitsLoaded.value = true
  } catch (error) {
    recruitsError.value = extractApiErrorMessage(error, '下线数据加载失败')
  } finally {
    recruitsLoading.value = false
  }
}

async function loadWithdrawals(page: number): Promise<void> {
  if (!props.agent || withdrawalsLoading.value) return
  withdrawalsLoading.value = true
  withdrawalsError.value = ''
  try {
    withdrawals.value = await resellerAPI.listAdminWithdrawals({
      userId: props.agent.user_id,
      page,
      pageSize: 10
    })
    withdrawalsLoaded.value = true
  } catch (error) {
    withdrawalsError.value = extractApiErrorMessage(error, '提现记录加载失败')
  } finally {
    withdrawalsLoading.value = false
  }
}

function statusLabel(status: ResellerStatus): string {
  if (status === 'disabled') return '已停用'
  if (status === 'revoked') return '已撤销'
  return '启用中'
}

function withdrawalStatusLabel(status: WithdrawStatus): string {
  const labels: Record<WithdrawStatus, string> = {
    pending: '待审批',
    approved: '已批准',
    rejected: '已拒绝',
    cancelled: '已取消'
  }
  return labels[status] || status
}

function withdrawalStatusClass(status: WithdrawStatus): string {
  return status === 'approved' ? 'active' : status === 'pending' ? 'pending' : 'muted'
}

function roleLabel(role: AgentSummary['role'] | string): string {
  return role === 'agent_manager' ? 'Agent Manager' : 'Agent'
}

function rebateLabel(agent: AgentSummary): string {
  if (agent.rebate_mode === 'disabled') return '已关闭（0%）'
  const rate = agent.effective_rebate_rate_percent
  if (agent.rebate_mode === 'global') {
    return rate == null ? '跟随全局' : `跟随全局（${formatRate(rate)}）`
  }
  return rate == null ? '自定义（未设置）' : `自定义（${formatRate(rate)}）`
}

function formatRate(value: number): string {
  return `${value.toFixed(2).replace(/\.?0+$/, '')}%`
}

function formatMoney(value: number | string): string {
  const amount = Number(value)
  return Number.isFinite(amount) ? amount.toFixed(2) : '0.00'
}

function copyRecruitId(value: number | string): void {
  void copyToClipboard(String(value), '用户ID已复制')
}

function formatDate(value?: string | null): string {
  return value ? formatDateTime(value) : '--'
}
</script>

<style scoped>
.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding-bottom: 1rem;
}

.detail-tabs {
  display: flex;
  gap: 0.25rem;
  overflow-x: auto;
  border-bottom: 1px solid var(--ssxz-border);
}

.detail-tab {
  flex: none;
  min-height: 2.5rem;
  border-bottom: 2px solid transparent;
  padding: 0 0.8rem;
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
  font-weight: 600;
}

.detail-tab:hover,
.detail-tab--active {
  border-bottom-color: var(--ssxz-text);
  color: var(--ssxz-text);
}

.section-title {
  margin-bottom: 0.75rem;
  color: var(--ssxz-text);
  font-size: 0.8rem;
  font-weight: 700;
}

.section-heading-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.detail-grid {
  display: grid;
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-border);
}

.detail-grid > div {
  min-width: 0;
  background: var(--ssxz-surface);
  padding: 0.85rem;
}

.detail-grid dt {
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
}

.detail-grid dd {
  margin-top: 0.35rem;
  overflow-wrap: anywhere;
  color: var(--ssxz-text);
  font-size: 0.875rem;
  font-weight: 500;
}

.detail-grid--metrics dd {
  font-size: 1.1rem;
}

.summary-grid {
  display: grid;
  gap: 0.65rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.summary-card {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  padding: 0.75rem;
}

.summary-card span {
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
}

.summary-card strong {
  color: var(--ssxz-text);
  font-size: 1rem;
}

.table-scroll {
  overflow-x: auto;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
}

.detail-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
  font-size: 0.78rem;
}

.detail-table th,
.detail-table td {
  border-bottom: 1px solid var(--ssxz-border);
  padding: 0.7rem 0.75rem;
  text-align: left;
  white-space: nowrap;
}

.detail-table th {
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-text-muted);
  font-size: 0.7rem;
  font-weight: 700;
}

.detail-table tbody tr:last-child td {
  border-bottom: 0;
}

.user-link {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  color: var(--ssxz-text);
  font-weight: 600;
}

.user-link:hover,
.subtle-link:hover {
  text-decoration: underline;
}

.user-link small {
  color: var(--ssxz-text-muted);
  font-size: 0.68rem;
  font-weight: 400;
}

.role-badge,
.status-badge {
  display: inline-flex;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  font-size: 0.7rem;
  font-weight: 600;
}

.role-badge {
  color: var(--ssxz-text-secondary);
}

.status-badge--active {
  border-color: color-mix(in srgb, var(--ssxz-success) 35%, var(--ssxz-border));
  color: var(--ssxz-success);
}

.status-badge--pending {
  border-color: color-mix(in srgb, var(--ssxz-warning) 35%, var(--ssxz-border));
  color: var(--ssxz-warning);
}

.status-badge--disabled,
.status-badge--muted,
.status-badge--revoked {
  color: var(--ssxz-text-muted);
}

.pagination-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  color: var(--ssxz-text-muted);
  font-size: 0.75rem;
}

.pagination-actions {
  display: flex;
  gap: 0.5rem;
}

.empty-state {
  padding: 2.5rem 1rem;
  text-align: center;
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.subtle-link {
  color: var(--ssxz-text-secondary);
  font-size: 0.75rem;
}

.note-cell {
  max-width: 18rem;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (min-width: 768px) {
  .detail-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .detail-grid--metrics {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .summary-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
