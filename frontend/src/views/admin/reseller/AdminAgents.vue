<template>
  <AppLayout>
    <AdminPageHeader
      title="Reseller 管理"
      description="查看已开通的 Agent 及其推广数据"
    >
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
          <div class="card p-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <input
                v-model.trim="search"
                class="input min-w-0 flex-1 sm:max-w-sm"
                type="search"
                placeholder="搜索邮箱"
                @keyup.enter="searchAgents"
              />
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
                  :disabled="loading || !search"
                  @click="resetSearch"
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
                <div class="truncate font-medium text-gray-900 dark:text-white">
                  {{ row.username || `用户 ${row.user_id}` }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ row.email || '--' }}
                </div>
              </div>
            </template>

            <template #cell-role="{ row }">
              <span class="role-badge">{{ roleLabel(row.role) }}</span>
            </template>

            <template #cell-manager>--</template>
            <template #cell-recruit_count="{ value }">{{ value ?? '--' }}</template>
            <template #cell-total_commission>--</template>
            <template #cell-commission_rate="{ value }">{{ formatRate(value) }}</template>
            <template #cell-granted_at="{ value }">
              {{ value ? formatDateTime(value) : '--' }}
            </template>
            <template #cell-actions="{ row }">
              <LiquidButton
                type="button"
                variant="destructive"
                size="sm"
                :disabled="revoking"
                :data-testid="`revoke-agent-${row.user_id}`"
                @click="revokeTarget = row"
              >
                <span>撤销</span>
              </LiquidButton>
            </template>

            <template #empty>
              <div class="py-10 text-center">
                <p class="font-medium text-gray-900 dark:text-white">暂无 Agent 数据</p>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ search ? '没有匹配当前搜索条件的用户。' : '当前没有已开通的 Agent。' }}
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

    <BaseDialog
      :show="!!revokeTarget"
      title="撤销 Reseller 角色"
      width="narrow"
      :close-on-escape="!revoking"
      @close="closeRevokeDialog"
    >
      <div class="space-y-3">
        <p class="text-sm text-[var(--ssxz-text-secondary)]">
          确认撤销
          <strong class="text-[var(--ssxz-text)]">
            {{ revokeTarget?.email || `用户 ${revokeTarget?.user_id ?? ''}` }}
          </strong>
          的 {{ revokeTarget ? roleLabel(revokeTarget.role) : '' }} 角色？
        </p>
        <p class="text-xs text-[var(--ssxz-text-muted)]">
          此操作会移除 Reseller 管理入口，但不会删除用户账号。
        </p>
      </div>
      <template #footer>
        <div class="flex flex-wrap justify-end gap-2">
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="revoking"
            @click="closeRevokeDialog"
          >
            <span>取消</span>
          </LiquidButton>
          <LiquidButton
            type="button"
            variant="destructive"
            size="sm"
            :disabled="revoking"
            data-testid="confirm-revoke-agent"
            @click="confirmRevoke"
          >
            <span>{{ revoking ? '正在撤销' : '确认撤销' }}</span>
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import resellerAPI, { type AgentSummary } from '@/api/reseller'
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
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const route = useRoute()
const appStore = useAppStore()
const loading = ref(true)
const revoking = ref(false)
const search = ref('')
const agents = ref<PaginatedResponse<AgentSummary>>(emptyPage())
const grantDialogOpen = ref(false)
const revokeTarget = ref<AgentSummary | null>(null)

const sectionLinks = [
  { to: '/admin/reseller/agents', label: 'Agent 列表' },
  { to: '/admin/reseller/withdrawals', label: '兑换审批' }
]

const columns: Column[] = [
  { key: 'user', label: '用户' },
  { key: 'role', label: '角色' },
  { key: 'manager', label: '上级 Manager' },
  { key: 'recruit_count', label: '招募人数' },
  { key: 'total_commission', label: '累计佣金' },
  { key: 'commission_rate', label: '佣金比例' },
  { key: 'granted_at', label: '加入时间' },
  { key: 'actions', label: '操作', class: 'text-right' }
]

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
}

function roleLabel(role: AgentSummary['role']): string {
  return role === 'agent_manager' ? 'Agent Manager' : 'Agent'
}

function formatRate(value: unknown): string {
  if (typeof value !== 'number') return '--'
  return `${(value * 100).toFixed(2).replace(/\.00$/, '')}%`
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
    agents.value = await resellerAPI.listAdminAgents(page, agents.value.page_size, search.value)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 列表加载失败'))
  } finally {
    loading.value = false
  }
}

function searchAgents(): void {
  void loadAgents(1)
}

function resetSearch(): void {
  search.value = ''
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
  void loadAgents(1)
}

function closeRevokeDialog(): void {
  if (revoking.value) return
  revokeTarget.value = null
}

async function confirmRevoke(): Promise<void> {
  const target = revokeTarget.value
  if (!target || revoking.value) return

  revoking.value = true
  try {
    await resellerAPI.revokeAdminRole(target.user_id)
    appStore.showSuccess(`${target.email || `用户 ${target.user_id}`} 的 Reseller 角色已撤销`)
    revokeTarget.value = null
    const nextPage =
      agents.value.items.length === 1 && agents.value.page > 1
        ? agents.value.page - 1
        : agents.value.page
    await loadAgents(nextPage)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Reseller 角色撤销失败'))
  } finally {
    revoking.value = false
  }
}

onMounted(() => void loadAgents(1))
</script>

<style scoped>
.role-badge {
  display: inline-flex;
  min-height: 1.5rem;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  padding: 0 0.55rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.72rem;
  font-weight: 600;
}
</style>
