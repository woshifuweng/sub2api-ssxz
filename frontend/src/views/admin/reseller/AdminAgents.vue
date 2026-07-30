<template>
  <AppLayout>
    <AdminPageHeader
      title="Reseller 管理"
      description="查看已开通的 Agent 及其推广数据"
    >
      <template #actions>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import resellerAPI, { type AgentSummary } from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const route = useRoute()
const appStore = useAppStore()
const loading = ref(true)
const search = ref('')
const agents = ref<PaginatedResponse<AgentSummary>>(emptyPage())

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
  { key: 'granted_at', label: '加入时间' }
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
