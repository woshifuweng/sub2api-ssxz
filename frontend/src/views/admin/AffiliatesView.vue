<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h1 class="text-lg font-semibold text-gray-900 dark:text-white">推广返利管理</h1>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                管理用户推广码、专属返利比例和推广数据。统计来自已有邀请关系、订单和返利账本。
              </p>
            </div>
            <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadEntries">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              刷新
            </button>
          </div>
        </div>

        <div class="grid gap-6 p-6 lg:grid-cols-[minmax(0,1fr)_320px]">
          <div class="space-y-4">
            <div>
              <label class="input-label">查找推广用户</label>
              <div class="flex flex-col gap-2 sm:flex-row">
                <input
                  v-model.trim="lookupKeyword"
                  class="input"
                  placeholder="输入邮箱、用户名或用户 ID"
                  @keyup.enter="lookupUsers"
                />
                <button
                  class="btn btn-primary shrink-0"
                  :disabled="lookupLoading || !lookupKeyword"
                  @click="lookupUsers"
                >
                  搜索用户
                </button>
              </div>
            </div>

            <div
              v-if="lookupResults.length > 0"
              class="divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-700"
            >
              <button
                v-for="user in lookupResults"
                :key="user.id"
                class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-700"
                @click="selectLookupUser(user)"
              >
                <span>
                  <span class="block text-sm font-medium text-gray-900 dark:text-white">
                    {{ user.email || user.username || `用户 ${user.id}` }}
                  </span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    ID {{ user.id }}{{ user.username ? ` · ${user.username}` : '' }}
                  </span>
                </span>
                <span class="text-xs text-primary-600 dark:text-primary-400">选择</span>
              </button>
            </div>

            <div
              v-else-if="lookupSearched"
              class="rounded-lg border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
            >
              没有找到匹配用户。
            </div>
          </div>

          <form class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700" @submit.prevent="saveSelectedUser">
            <div>
              <div class="text-sm font-medium text-gray-900 dark:text-white">当前配置用户</div>
              <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                <template v-if="selectedUser">
                  {{ selectedUser.email || selectedUser.username || `用户 ${selectedUser.id}` }}
                </template>
                <template v-else>先从搜索结果或下方表格选择用户。</template>
              </div>
            </div>

            <div>
              <label class="input-label">推广码</label>
              <input
                v-model.trim="form.aff_code"
                class="input font-mono uppercase"
                placeholder="例如 SSXZ2026"
                :disabled="!selectedUser"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                留空不会修改推广码；后端会校验格式和唯一性。
              </p>
            </div>

            <div>
              <label class="input-label">专属返利比例（%）</label>
              <input
                v-model.number="form.aff_rebate_rate_percent"
                type="number"
                min="0"
                max="100"
                step="0.01"
                class="input"
                placeholder="留空使用默认比例"
                :disabled="!selectedUser || form.clear_rebate_rate"
              />
              <label class="mt-2 flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
                <input v-model="form.clear_rebate_rate" type="checkbox" class="rounded border-gray-300" :disabled="!selectedUser" />
                清除专属比例，改用系统默认比例
              </label>
            </div>

            <div class="flex flex-wrap gap-2">
              <button class="btn btn-primary" :disabled="saving || !selectedUser" type="submit">
                保存设置
              </button>
              <button
                class="btn btn-secondary text-red-600 dark:text-red-400"
                type="button"
                :disabled="saving || !selectedUser"
                @click="clearSelectedUser"
              >
                清除自定义
              </button>
            </div>
          </form>
        </div>
      </section>

      <TablePageLayout>
        <template #filters>
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <input
              v-model.trim="searchQuery"
              class="input sm:max-w-xs"
              placeholder="搜索自定义推广用户"
              @keyup.enter="handleSearch"
            />
            <div class="flex gap-2">
              <button class="btn btn-secondary" :disabled="loading" @click="handleSearch">搜索</button>
              <button class="btn btn-secondary" :disabled="loading" @click="resetSearch">重置</button>
            </div>
          </div>
        </template>

        <template #table>
          <DataTable :columns="columns" :data="entries" :loading="loading">
            <template #cell-user="{ row }">
              <div>
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ row.email || row.username || `用户 ${row.user_id}` }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">ID {{ row.user_id }}</div>
              </div>
            </template>

            <template #cell-aff_code="{ value, row }">
              <div class="flex flex-col">
                <code class="font-mono text-sm text-gray-900 dark:text-gray-100">{{ value }}</code>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.aff_code_custom ? '自定义' : '系统生成' }}
                </span>
              </div>
            </template>

            <template #cell-aff_rebate_rate_percent="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">
                {{ formatRate(value) }}
              </span>
            </template>

            <template #cell-aff_count="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ value }} 人</span>
            </template>

            <template #cell-invitee_recharge_total="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ formatQuota(value) }}</span>
            </template>

            <template #cell-accrued_rebate_total="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ formatQuota(value) }}</span>
            </template>

            <template #cell-aff_frozen_quota="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ formatQuota(value) }}</span>
            </template>

            <template #cell-aff_quota="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ formatQuota(value) }}</span>
            </template>

            <template #cell-transferred_rebate_total="{ value }">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ formatQuota(value) }}</span>
            </template>

            <template #cell-actions="{ row }">
              <div class="flex items-center gap-2">
                <button class="btn btn-secondary btn-sm" @click="selectEntry(row)">编辑</button>
                <button class="btn btn-secondary btn-sm text-red-600 dark:text-red-400" @click="clearEntry(row)">
                  清除
                </button>
              </div>
            </template>
          </DataTable>
        </template>

        <template #pagination>
          <Pagination
            v-if="pagination.total > 0"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </template>
      </TablePageLayout>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import type {
  AdminAffiliateEntry,
  AffiliateUserSummary
} from '@/api/admin/affiliate'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'

type SelectedAffiliateUser = AffiliateUserSummary | {
  id: number
  email: string
  username: string
}

const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const lookupLoading = ref(false)
const lookupSearched = ref(false)
const searchQuery = ref('')
const lookupKeyword = ref('')
const entries = ref<AdminAffiliateEntry[]>([])
const lookupResults = ref<AffiliateUserSummary[]>([])
const selectedUser = ref<SelectedAffiliateUser | null>(null)

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const form = reactive({
  aff_code: '',
  aff_rebate_rate_percent: null as number | null,
  clear_rebate_rate: false
})

const columns = computed<Column[]>(() => [
  { key: 'user', label: '用户' },
  { key: 'aff_code', label: '推广码' },
  { key: 'aff_count', label: '邀请人数' },
  { key: 'invitee_recharge_total', label: '充值额度' },
  { key: 'accrued_rebate_total', label: '累计返利' },
  { key: 'aff_frozen_quota', label: '冻结返利' },
  { key: 'aff_quota', label: '可结算' },
  { key: 'transferred_rebate_total', label: '已转余额' },
  { key: 'aff_rebate_rate_percent', label: '专属比例' },
  { key: 'actions', label: '操作' }
])

function formatRate(value: number | null | undefined) {
  return value == null ? '默认比例' : `${value.toFixed(2).replace(/\.00$/, '')}%`
}

function formatQuota(value: number | null | undefined) {
  return `${Number(value ?? 0).toFixed(2)} 额度`
}

function resetForm() {
  form.aff_code = ''
  form.aff_rebate_rate_percent = null
  form.clear_rebate_rate = false
}

function setPagination(response: { total: number; page: number; page_size: number; pages: number }) {
  pagination.total = response.total
  pagination.page = response.page
  pagination.page_size = response.page_size
  pagination.pages = response.pages
}

async function loadEntries() {
  loading.value = true
  try {
    const response = await adminAPI.affiliate.listUsers(
      pagination.page,
      pagination.page_size,
      searchQuery.value
    )
    entries.value = response.items
    setPagination(response)
  } catch (error: any) {
    appStore.showError(error?.message || '推广返利列表加载失败')
  } finally {
    loading.value = false
  }
}

async function lookupUsers() {
  if (!lookupKeyword.value) return
  lookupLoading.value = true
  lookupSearched.value = true
  try {
    lookupResults.value = await adminAPI.affiliate.lookupUsers(lookupKeyword.value)
  } catch (error: any) {
    appStore.showError(error?.message || '用户搜索失败')
  } finally {
    lookupLoading.value = false
  }
}

function selectLookupUser(user: AffiliateUserSummary) {
  selectedUser.value = user
  resetForm()
}

function selectEntry(row: AdminAffiliateEntry) {
  selectedUser.value = {
    id: row.user_id,
    email: row.email,
    username: row.username
  }
  form.aff_code = row.aff_code || ''
  form.aff_rebate_rate_percent = row.aff_rebate_rate_percent ?? null
  form.clear_rebate_rate = false
  lookupKeyword.value = row.email || row.username || String(row.user_id)
}

function buildPayload() {
  const payload: {
    aff_code?: string
    aff_rebate_rate_percent?: number | null
    clear_rebate_rate?: boolean
  } = {}
  if (form.aff_code) {
    payload.aff_code = form.aff_code.toUpperCase()
  }
  if (form.clear_rebate_rate) {
    payload.clear_rebate_rate = true
  } else if (form.aff_rebate_rate_percent != null) {
    payload.aff_rebate_rate_percent = Number(form.aff_rebate_rate_percent)
  }
  return payload
}

async function saveSelectedUser() {
  if (!selectedUser.value) return
  const payload = buildPayload()
  if (Object.keys(payload).length === 0) {
    appStore.showError('请填写推广码或专属返利比例')
    return
  }
  saving.value = true
  try {
    await adminAPI.affiliate.updateUserSettings(selectedUser.value.id, payload)
    appStore.showSuccess('推广返利设置已保存')
    await loadEntries()
  } catch (error: any) {
    appStore.showError(error?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function clearSelectedUser() {
  if (!selectedUser.value) return
  saving.value = true
  try {
    await adminAPI.affiliate.clearUserSettings(selectedUser.value.id)
    resetForm()
    appStore.showSuccess('推广返利自定义设置已清除')
    await loadEntries()
  } catch (error: any) {
    appStore.showError(error?.message || '清除失败')
  } finally {
    saving.value = false
  }
}

async function clearEntry(row: AdminAffiliateEntry) {
  saving.value = true
  try {
    await adminAPI.affiliate.clearUserSettings(row.user_id)
    appStore.showSuccess('推广返利自定义设置已清除')
    await loadEntries()
  } catch (error: any) {
    appStore.showError(error?.message || '清除失败')
  } finally {
    saving.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  loadEntries()
}

function resetSearch() {
  searchQuery.value = ''
  pagination.page = 1
  loadEntries()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadEntries()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadEntries()
}

onMounted(() => {
  loadEntries()
})
</script>
