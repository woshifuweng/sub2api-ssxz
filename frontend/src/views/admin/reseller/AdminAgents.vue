<template>
  <AppLayout>
    <AdminPageHeader title="Reseller 管理" description="管理 Agent 合作关系、返利策略与生命周期">
      <template #actions>
        <LiquidButton
          type="button"
          size="sm"
          data-testid="open-grant-dialog"
          @click="grantDialogOpen = true"
        >
          <Icon name="userPlus" size="sm" />
          <span>添加 Agent</span>
        </LiquidButton>
        <LiquidButton
          type="button"
          variant="outline"
          size="sm"
          :disabled="loading"
          title="刷新"
          @click="loadAgents"
        >
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          <span>刷新</span>
        </LiquidButton>
      </template>
    </AdminPageHeader>

    <div class="space-y-5">
      <nav class="flex flex-wrap gap-2" aria-label="Reseller 管理导航">
        <RouterLink
          v-for="item in sectionLinks"
          :key="item.to"
          :to="item.to"
          :class="sectionLinkClass(item.to)"
        >
          {{ item.label }}
        </RouterLink>
      </nav>

      <TablePageLayout>
        <template #filters>
          <div class="filter-panel">
            <div class="status-tabs" role="tablist" aria-label="Agent 状态">
              <button
                v-for="tab in statusTabs"
                :key="tab.value"
                type="button"
                :class="['status-tab', { 'status-tab--active': statusFilter === tab.value }]"
                role="tab"
                :aria-selected="statusFilter === tab.value"
                @click="setStatusFilter(tab.value)"
              >
                {{ tab.label }}
              </button>
            </div>

            <div class="filter-grid">
              <input
                v-model.trim="search"
                class="input min-w-0"
                type="search"
                placeholder="搜索邮箱"
                @keyup.enter="searchAgents"
              />
              <select v-model="roleFilter" class="input" @change="searchAgents">
                <option value="">全部角色</option>
                <option value="agent">Agent</option>
                <option value="agent_manager">Agent Manager</option>
              </select>
              <select v-model="managerFilter" class="input" @change="searchAgents">
                <option value="">全部上级</option>
                <option
                  v-for="manager in managers"
                  :key="manager.user_id"
                  :value="String(manager.user_id)"
                >
                  {{ manager.username || manager.email }}
                </option>
              </select>
              <div class="flex gap-2">
                <LiquidButton
                  type="button"
                  variant="outline"
                  size="sm"
                  :disabled="loading"
                  @click="searchAgents"
                >
                  <Icon name="search" size="sm" />
                  <span>搜索</span>
                </LiquidButton>
                <LiquidButton
                  type="button"
                  variant="outline"
                  size="sm"
                  :disabled="loading || !hasFilters"
                  @click="resetFilters"
                >
                  <span>重置</span>
                </LiquidButton>
              </div>
            </div>
          </div>
        </template>

        <template #table>
          <DataTable :columns="columns" :data="agents.items" :loading="loading">
            <template #cell-user="{ row }">
              <div class="min-w-0">
                <button
                  type="button"
                  class="user-link"
                  :title="`查看 ${row.email || `用户 ${row.user_id}`} 详情`"
                  @click="openDetail(row)"
                >
                  {{ row.username || `用户 ${row.user_id}` }}
                </button>
                <div class="truncate text-xs text-[var(--ssxz-text-muted)]">
                  {{ row.email || '--' }}
                </div>
              </div>
            </template>

            <template #cell-status="{ row }">
              <span :class="['status-badge', `status-badge--${row.status}`]">
                {{ statusLabel(row.status) }}
              </span>
            </template>

            <template #cell-role="{ row }">
              <span class="role-badge">{{ roleLabel(row.role) }}</span>
            </template>

            <template #cell-manager_email="{ value }">{{ value || '--' }}</template>
            <template #cell-recruit_count="{ value }">{{ value ?? 0 }}</template>
            <template #cell-rebate="{ row }">{{ rebateLabel(row) }}</template>
            <template #cell-commission_balance="{ value }">${{ formatMoney(value) }}</template>
            <template #cell-commission_total="{ value }">${{ formatMoney(value) }}</template>
            <template #cell-updated_at="{ value }">{{ value ? formatDateTime(value) : '--' }}</template>

            <template #cell-actions="{ row }">
              <div class="action-group">
                <LiquidButton
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="查看详情"
                  :data-testid="`view-agent-${row.user_id}`"
                  @click="openDetail(row)"
                >
                  <Icon name="eye" size="sm" />
                </LiquidButton>

                <LiquidButton
                  v-if="row.status !== 'revoked'"
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="编辑"
                  :disabled="busyAgentId === row.user_id"
                  :data-testid="`edit-agent-${row.user_id}`"
                  @click="openEdit(row)"
                >
                  <Icon name="edit" size="sm" />
                </LiquidButton>

                <LiquidButton
                  v-if="row.status === 'active'"
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="停用"
                  :disabled="busyAgentId === row.user_id"
                  :data-testid="`disable-agent-${row.user_id}`"
                  @click="disableTarget = row"
                >
                  <Icon name="ban" size="sm" />
                </LiquidButton>

                <LiquidButton
                  v-else-if="row.status === 'disabled'"
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="恢复启用"
                  :disabled="busyAgentId === row.user_id"
                  :data-testid="`enable-agent-${row.user_id}`"
                  @click="enableAgent(row)"
                >
                  <Icon name="play" size="sm" />
                </LiquidButton>

                <LiquidButton
                  v-else
                  type="button"
                  variant="outline"
                  size="sm"
                  :disabled="busyAgentId === row.user_id"
                  :data-testid="`reauthorize-agent-${row.user_id}`"
                  @click="reauthorizeTarget = row"
                >
                  <span>重新授权</span>
                </LiquidButton>

                <details v-if="row.status !== 'revoked'" class="action-menu">
                  <summary title="更多操作" aria-label="更多操作">
                    <Icon name="more" size="sm" />
                  </summary>
                  <div class="action-menu__panel">
                    <button
                      type="button"
                      :data-testid="`revoke-agent-${row.user_id}`"
                      @click="revokeTarget = row"
                    >
                      最终撤销角色
                    </button>
                  </div>
                </details>
              </div>
            </template>

            <template #empty>
              <div class="py-10 text-center">
                <p class="font-medium text-[var(--ssxz-text)]">暂无 Agent 数据</p>
                <p class="mt-1 text-sm text-[var(--ssxz-text-muted)]">
                  {{ hasFilters ? '没有匹配当前筛选条件的记录。' : '当前没有已授权的 Agent。' }}
                </p>
              </div>
            </template>
          </DataTable>
        </template>

        <template #pagination>
          <Pagination
            v-if="agents.total > 0"
            :page="agents.page"
            :page-size="agents.page_size"
            :total="agents.total"
            @update:page="changePage"
            @update:page-size="changePageSize"
          />
        </template>
      </TablePageLayout>
    </div>

    <AdminAgentGrantDialog
      :show="grantDialogOpen"
      @close="grantDialogOpen = false"
      @granted="handleGranted"
    />

    <AdminAgentEditDrawer
      :show="!!editAgent"
      :agent="editAgent"
      :managers="managers"
      @close="editAgent = null"
      @saved="handleAgentChanged"
    />

    <AdminAgentDetailDrawer
      :show="detailOpen"
      :agent="detailAgent"
      :loading="detailLoading"
      @close="closeDetail"
    />

    <BaseDialog
      :show="!!disableTarget"
      title="停用 Agent"
      width="narrow"
      :close-on-escape="!actionSubmitting"
      @close="closeDisableDialog"
    >
      <div class="space-y-4">
        <p class="text-sm text-[var(--ssxz-text-secondary)]">
          停用后，该 Agent 不再产生新返利，但历史数据和合作配置会保留。
        </p>
        <label class="block">
          <span class="input-label mb-1.5 block">停用原因</span>
          <textarea
            v-model.trim="disableReason"
            class="input min-h-24 w-full resize-y"
            maxlength="300"
            placeholder="请填写停用原因"
            :disabled="actionSubmitting"
          />
        </label>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="actionSubmitting"
            @click="closeDisableDialog"
          >
            <span>取消</span>
          </LiquidButton>
          <LiquidButton
            type="button"
            variant="destructive"
            size="sm"
            :disabled="actionSubmitting || !disableReason"
            data-testid="confirm-disable-agent"
            @click="confirmDisable"
          >
            <span>{{ actionSubmitting ? '正在停用' : '确认停用' }}</span>
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="!!revokeTarget"
      title="最终撤销 Reseller 角色"
      width="narrow"
      :close-on-escape="!actionSubmitting"
      @close="closeRevokeDialog"
    >
      <div class="space-y-3">
        <p class="text-sm text-[var(--ssxz-text-secondary)]">
          确认最终撤销
          <strong class="text-[var(--ssxz-text)]">
            {{ revokeTarget?.email || `用户 ${revokeTarget?.user_id ?? ''}` }}
          </strong>
          的 {{ revokeTarget ? roleLabel(revokeTarget.role) : '' }} 角色？
        </p>
        <p class="text-xs text-[var(--ssxz-danger)]">
          这不是临时停用。若该账号仍有直属 Agent 或待处理兑换，系统会拒绝操作。
        </p>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="actionSubmitting"
            @click="closeRevokeDialog"
          >
            <span>取消</span>
          </LiquidButton>
          <LiquidButton
            type="button"
            variant="destructive"
            size="sm"
            :disabled="actionSubmitting"
            data-testid="confirm-revoke-agent"
            @click="confirmRevoke"
          >
            <span>{{ actionSubmitting ? '正在撤销' : '确认最终撤销' }}</span>
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="!!reauthorizeTarget"
      title="重新授权 Reseller 角色"
      width="narrow"
      :close-on-escape="!actionSubmitting"
      @close="closeReauthorizeDialog"
    >
      <p class="text-sm text-[var(--ssxz-text-secondary)]">
        将按原角色重新授权
        <strong class="text-[var(--ssxz-text)]">{{ reauthorizeTarget?.email }}</strong>
        。重新授权后状态恢复为启用，上级关系需在编辑中重新设置。
      </p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="actionSubmitting"
            @click="closeReauthorizeDialog"
          >
            <span>取消</span>
          </LiquidButton>
          <LiquidButton
            type="button"
            size="sm"
            :disabled="actionSubmitting"
            data-testid="confirm-reauthorize-agent"
            @click="confirmReauthorize"
          >
            <span>{{ actionSubmitting ? '正在授权' : '确认重新授权' }}</span>
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import resellerAPI, {
  type AgentDetail,
  type AgentSummary,
  type ResellerRole,
  type ResellerStatus
} from '@/api/reseller'
import type { AdminUser, PaginatedResponse } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Icon from '@/components/icons/Icon.vue'
import AdminAgentGrantDialog from './AdminAgentGrantDialog.vue'
import AdminAgentEditDrawer from './AdminAgentEditDrawer.vue'
import AdminAgentDetailDrawer from './AdminAgentDetailDrawer.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

type StatusTabValue = '' | ResellerStatus

const route = useRoute()
const appStore = useAppStore()
const loading = ref(true)
const search = ref('')
const statusFilter = ref<StatusTabValue>('')
const roleFilter = ref<'' | ResellerRole>('')
const managerFilter = ref('')
const agents = ref<PaginatedResponse<AgentSummary>>(emptyPage())
const managers = ref<AgentSummary[]>([])
const grantDialogOpen = ref(false)
const editAgent = ref<AgentDetail | null>(null)
const detailOpen = ref(false)
const detailAgent = ref<AgentDetail | null>(null)
const detailLoading = ref(false)
const disableTarget = ref<AgentSummary | null>(null)
const disableReason = ref('')
const revokeTarget = ref<AgentSummary | null>(null)
const reauthorizeTarget = ref<AgentSummary | null>(null)
const actionSubmitting = ref(false)
const busyAgentId = ref<number | null>(null)

const sectionLinks = [
  { to: '/admin/reseller/agents', label: 'Agent 列表' },
  { to: '/admin/reseller/withdrawals', label: '兑换审批' }
]

const statusTabs: Array<{ value: StatusTabValue; label: string }> = [
  { value: '', label: '当前合作' },
  { value: 'active', label: '启用中' },
  { value: 'disabled', label: '已停用' },
  { value: 'revoked', label: '已撤销' }
]

const columns: Column[] = [
  { key: 'user', label: '用户' },
  { key: 'status', label: '状态' },
  { key: 'role', label: '角色' },
  { key: 'manager_email', label: '上级 Manager' },
  { key: 'recruit_count', label: '招募' },
  { key: 'rebate', label: '返利策略' },
  { key: 'commission_balance', label: '可兑换佣金' },
  { key: 'commission_total', label: '累计佣金' },
  { key: 'updated_at', label: '最近更新' },
  { key: 'actions', label: '操作', class: 'text-right' }
]

const hasFilters = computed(
  () => Boolean(search.value || statusFilter.value || roleFilter.value || managerFilter.value)
)

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
}

function roleLabel(role: AgentSummary['role']): string {
  return role === 'agent_manager' ? 'Agent Manager' : 'Agent'
}

function statusLabel(status: ResellerStatus): string {
  if (status === 'disabled') return '已停用'
  if (status === 'revoked') return '已撤销'
  return '启用中'
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

function formatMoney(value: unknown): string {
  const amount = Number(value)
  return Number.isFinite(amount) ? amount.toFixed(2) : '0.00'
}

function sectionLinkClass(path: string): string[] {
  const active = route.path === path
  return [
    'inline-flex h-9 items-center rounded-md border px-3 text-sm font-medium transition-colors',
    active
      ? 'border-gray-900 bg-gray-900 text-white dark:border-white dark:bg-white dark:text-gray-950'
      : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:hover:bg-dark-800'
  ]
}

async function loadAgents(page = agents.value.page || 1): Promise<void> {
  loading.value = true
  try {
    agents.value = await resellerAPI.listAdminAgents(page, agents.value.page_size, {
      search: search.value || undefined,
      status: statusFilter.value,
      role: roleFilter.value,
      manager_id: managerFilter.value ? Number(managerFilter.value) : undefined
    })
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 列表加载失败'))
  } finally {
    loading.value = false
  }
}

async function loadManagers(): Promise<void> {
  try {
    const response = await resellerAPI.listAdminAgents(1, 100, {
      status: 'active',
      role: 'agent_manager'
    })
    managers.value = response.items
  } catch {
    managers.value = []
  }
}

function setStatusFilter(status: StatusTabValue): void {
  statusFilter.value = status
  void loadAgents(1)
}

function searchAgents(): void {
  void loadAgents(1)
}

function resetFilters(): void {
  search.value = ''
  statusFilter.value = ''
  roleFilter.value = ''
  managerFilter.value = ''
  void loadAgents(1)
}

function changePage(page: number): void {
  void loadAgents(page)
}

function changePageSize(pageSize: number): void {
  agents.value.page_size = pageSize
  void loadAgents(1)
}

function handleGranted(user: AdminUser): void {
  grantDialogOpen.value = false
  search.value = user.email
  void Promise.all([loadAgents(1), loadManagers()])
}

async function openDetail(agent: AgentSummary): Promise<void> {
  detailOpen.value = true
  detailLoading.value = true
  detailAgent.value = null
  try {
    detailAgent.value = await resellerAPI.getAdminAgent(agent.user_id)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 详情加载失败'))
    detailOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

function closeDetail(): void {
  detailOpen.value = false
  detailAgent.value = null
}

async function openEdit(agent: AgentSummary): Promise<void> {
  busyAgentId.value = agent.user_id
  try {
    editAgent.value = await resellerAPI.getAdminAgent(agent.user_id)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 配置加载失败'))
  } finally {
    busyAgentId.value = null
  }
}

function handleAgentChanged(agent: AgentDetail): void {
  editAgent.value = null
  detailAgent.value = agent
  void Promise.all([loadAgents(agents.value.page), loadManagers()])
}

function closeDisableDialog(): void {
  if (actionSubmitting.value) return
  disableTarget.value = null
  disableReason.value = ''
}

async function confirmDisable(): Promise<void> {
  const target = disableTarget.value
  if (!target || actionSubmitting.value || !disableReason.value) return
  actionSubmitting.value = true
  busyAgentId.value = target.user_id
  try {
    await resellerAPI.disableAdminAgent(target.user_id, disableReason.value)
    appStore.showSuccess(`${target.email} 已停用`)
    disableTarget.value = null
    disableReason.value = ''
    await loadAgents(agents.value.page)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 停用失败'))
  } finally {
    actionSubmitting.value = false
    busyAgentId.value = null
  }
}

async function enableAgent(target: AgentSummary): Promise<void> {
  if (busyAgentId.value) return
  busyAgentId.value = target.user_id
  try {
    await resellerAPI.enableAdminAgent(target.user_id)
    appStore.showSuccess(`${target.email} 已恢复启用`)
    await loadAgents(agents.value.page)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 恢复失败'))
  } finally {
    busyAgentId.value = null
  }
}

function closeRevokeDialog(): void {
  if (actionSubmitting.value) return
  revokeTarget.value = null
}

async function confirmRevoke(): Promise<void> {
  const target = revokeTarget.value
  if (!target || actionSubmitting.value) return
  actionSubmitting.value = true
  busyAgentId.value = target.user_id
  try {
    await resellerAPI.revokeAdminRole(target.user_id)
    appStore.showSuccess(`${target.email || `用户 ${target.user_id}`} 的 Reseller 角色已最终撤销`)
    revokeTarget.value = null
    await Promise.all([loadAgents(agents.value.page), loadManagers()])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Reseller 角色撤销失败'))
  } finally {
    actionSubmitting.value = false
    busyAgentId.value = null
  }
}

function closeReauthorizeDialog(): void {
  if (actionSubmitting.value) return
  reauthorizeTarget.value = null
}

async function confirmReauthorize(): Promise<void> {
  const target = reauthorizeTarget.value
  if (!target || actionSubmitting.value) return
  actionSubmitting.value = true
  busyAgentId.value = target.user_id
  try {
    await resellerAPI.grantAdminRole(target.user_id, {
      role: target.role,
      notes: target.notes || undefined
    })
    appStore.showSuccess(`${target.email} 已重新授权`)
    reauthorizeTarget.value = null
    statusFilter.value = 'active'
    await Promise.all([loadAgents(1), loadManagers()])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Reseller 角色重新授权失败'))
  } finally {
    actionSubmitting.value = false
    busyAgentId.value = null
  }
}

onMounted(() => {
  void Promise.all([loadAgents(1), loadManagers()])
})
</script>

<style scoped>
.filter-panel {
  display: grid;
  gap: 1rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  padding: 1rem;
}

.status-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
  border-bottom: 1px solid var(--ssxz-border);
}

.status-tab {
  min-height: 2.25rem;
  border-bottom: 2px solid transparent;
  padding: 0 0.75rem;
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
  font-weight: 600;
}

.status-tab:hover {
  color: var(--ssxz-text);
}

.status-tab--active {
  border-bottom-color: var(--ssxz-text);
  color: var(--ssxz-text);
}

.filter-grid {
  display: grid;
  gap: 0.75rem;
}

.user-link {
  display: block;
  max-width: 13rem;
  overflow: hidden;
  color: var(--ssxz-text);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-link:hover {
  text-decoration: underline;
}

.role-badge,
.status-badge {
  display: inline-flex;
  min-height: 1.5rem;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  padding: 0 0.55rem;
  font-size: 0.72rem;
  font-weight: 600;
}

.role-badge {
  color: var(--ssxz-text-secondary);
}

.status-badge--active {
  border-color: color-mix(in srgb, var(--ssxz-success) 35%, var(--ssxz-border));
  color: var(--ssxz-success);
}

.status-badge--disabled {
  border-color: color-mix(in srgb, var(--ssxz-warning) 35%, var(--ssxz-border));
  color: var(--ssxz-warning);
}

.status-badge--revoked {
  color: var(--ssxz-text-muted);
}

.action-group {
  display: flex;
  min-width: max-content;
  align-items: center;
  justify-content: flex-end;
  gap: 0.25rem;
}

.action-menu {
  position: relative;
}

.action-menu summary {
  display: inline-flex;
  width: 2.25rem;
  height: 2.25rem;
  cursor: pointer;
  list-style: none;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: var(--ssxz-text-secondary);
}

.action-menu summary::-webkit-details-marker {
  display: none;
}

.action-menu summary:hover {
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-text);
}

.action-menu__panel {
  position: absolute;
  z-index: 20;
  top: calc(100% + 0.35rem);
  right: 0;
  min-width: 9.5rem;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-lg);
}

.action-menu__panel button {
  width: 100%;
  padding: 0.7rem 0.85rem;
  color: var(--ssxz-danger);
  font-size: 0.78rem;
  text-align: left;
}

.action-menu__panel button:hover {
  background: color-mix(in srgb, var(--ssxz-danger) 10%, transparent);
}

@media (min-width: 768px) {
  .filter-grid {
    grid-template-columns: minmax(14rem, 1fr) minmax(9rem, 0.35fr) minmax(10rem, 0.45fr) auto;
    align-items: center;
  }
}
</style>
