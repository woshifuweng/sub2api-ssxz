<template>
  <AppLayout>
    <AdminPageHeader
      title="兑换审批"
      description="审核 Agent 转入账户余额的申请"
    >
      <template #actions>
        <LiquidButton
          type="button"
          variant="outline"
          size="sm"
          :disabled="loading"
          @click="loadRequests"
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
          <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
            <div class="flex gap-2" role="group" aria-label="申请状态">
              <LiquidButton
                v-for="tab in statusTabs"
                :key="tab.value || 'all'"
                type="button"
                :variant="statusFilter === tab.value ? 'default' : 'outline'"
                size="sm"
                :disabled="loading"
                @click="setStatusFilter(tab.value)"
              >
                <span>{{ tab.label }}</span>
              </LiquidButton>
            </div>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              共 {{ requests.total }} 条
            </span>
          </div>
        </template>

        <template #table>
          <DataTable :columns="columns" :data="requests.items" :loading="loading">
            <template #cell-applicant="{ row }">
              <div class="min-w-0">
                <div class="truncate font-medium text-gray-900 dark:text-white">
                  {{ row.username || `用户 #${row.user_id}` }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.user_email || '--' }}
                </div>
              </div>
            </template>

            <template #cell-amount="{ value }">
              <span class="font-medium text-gray-900 dark:text-white">
                {{ formatCurrency(value) }}
              </span>
            </template>

            <template #cell-status="{ row }">
              <div class="flex flex-col items-start gap-1">
                <WithdrawalStatusBadge :status="row.status" />
                <span
                  v-if="row.status === 'rejected' && row.note"
                  class="max-w-48 truncate text-xs text-red-600 dark:text-red-400"
                  :title="row.note"
                >
                  {{ row.note }}
                </span>
              </div>
            </template>

            <template #cell-requested_at="{ value }">
              {{ value ? formatDateTime(value) : '--' }}
            </template>

            <template #cell-reviewed_at="{ value }">
              {{ value ? formatDateTime(value) : '--' }}
            </template>

            <template #cell-actions="{ row }">
              <div v-if="row.status === 'pending'" class="flex flex-wrap items-center gap-2">
                <LiquidButton
                  type="button"
                  size="sm"
                  :disabled="reviewingId === row.id"
                  @click="approveRequest(row)"
                >
                  <span>批准</span>
                </LiquidButton>
                <LiquidButton
                  type="button"
                  variant="destructive"
                  size="sm"
                  :disabled="reviewingId === row.id"
                  @click="openRejectDialog(row)"
                >
                  <span>拒绝</span>
                </LiquidButton>
              </div>
              <span v-else>--</span>
            </template>

            <template #empty>
              <div class="py-10 text-center">
                <p class="font-medium text-gray-900 dark:text-white">暂无申请记录</p>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ statusFilter === 'pending' ? '当前没有待审批申请。' : '当前没有历史申请。' }}
                </p>
              </div>
            </template>
          </DataTable>
        </template>

        <template #pagination>
          <Pagination
            v-if="requests.total > 0"
            :page="requests.page"
            :page-size="requests.page_size"
            :total="requests.total"
            @update:page="changePage"
            @update:page-size="changePageSize"
          />
        </template>
      </TablePageLayout>
    </div>

    <BaseDialog
      :show="!!rejectTarget"
      title="拒绝兑换申请"
      width="narrow"
      @close="closeRejectDialog"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-300">
          请填写拒绝原因。该原因会展示在用户的兑换记录中。
        </p>
        <label class="block">
          <span class="input-label mb-1.5 block">拒绝原因</span>
          <textarea
            v-model="rejectReason"
            class="input min-h-28 w-full resize-y"
            maxlength="500"
            placeholder="请输入明确原因"
            :disabled="reviewingId !== null"
          />
        </label>
      </div>

      <template #footer>
        <div class="flex justify-end gap-2">
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="reviewingId !== null"
            @click="closeRejectDialog"
          >
            <span>取消</span>
          </LiquidButton>
          <LiquidButton
            type="button"
            variant="destructive"
            size="sm"
            :disabled="reviewingId !== null || !rejectReason.trim()"
            @click="rejectRequest"
          >
            <span>{{ reviewingId !== null ? '正在提交' : '确认拒绝' }}</span>
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>
    <TotpStepUpDialog :controller="stepUp" />
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import resellerAPI, { type WithdrawRequest, type WithdrawStatus } from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import WithdrawalStatusBadge from '@/components/reseller/WithdrawalStatusBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { isStepUpCancelled, useStepUp } from '@/composables/useStepUp'

type StatusFilter = Extract<WithdrawStatus, 'pending' | 'approved' | 'rejected' | 'cancelled'> | ''

const route = useRoute()
const appStore = useAppStore()
const stepUp = useStepUp()
const loading = ref(true)
const statusFilter = ref<StatusFilter>('pending')
const reviewingId = ref<number | null>(null)
const rejectTarget = ref<WithdrawRequest | null>(null)
const rejectReason = ref('')
const requests = ref<PaginatedResponse<WithdrawRequest>>(emptyPage())

const statusTabs: Array<{ value: StatusFilter; label: string }> = [
  { value: '', label: '全部' },
  { value: 'pending', label: '待审批' },
  { value: 'approved', label: '已批准' },
  { value: 'rejected', label: '已拒绝' },
  { value: 'cancelled', label: '已取消' }
]

const sectionLinks = [
  { to: '/admin/reseller/agents', label: 'Agent 列表' },
  { to: '/admin/reseller/withdrawals', label: '兑换审批' }
]

const columns: Column[] = [
  { key: 'applicant', label: '申请人' },
  { key: 'amount', label: '申请金额' },
  { key: 'status', label: '状态' },
  { key: 'requested_at', label: '申请时间' },
  { key: 'reviewed_at', label: '审批时间' },
  { key: 'actions', label: '操作' }
]

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
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

async function loadRequests(page = requests.value.page || 1): Promise<void> {
  loading.value = true
  try {
    requests.value = await resellerAPI.listAdminWithdrawals({
      page,
      pageSize: requests.value.page_size,
      status: statusFilter.value
    })
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '兑换申请加载失败'))
  } finally {
    loading.value = false
  }
}

function setStatusFilter(status: StatusFilter): void {
  if (statusFilter.value === status) return
  statusFilter.value = status
  void loadRequests(1)
}

async function approveRequest(request: WithdrawRequest): Promise<void> {
  reviewingId.value = request.id
  try {
    await stepUp.run(() => resellerAPI.reviewWithdrawal(request.id, { action: 'approve' }))
    appStore.showSuccess('兑换申请已批准')
    await loadRequests(requests.value.page)
  } catch (error) {
    if (isStepUpCancelled(error)) return
    appStore.showError(extractApiErrorMessage(error, '批准失败'))
  } finally {
    reviewingId.value = null
  }
}

function openRejectDialog(request: WithdrawRequest): void {
  rejectTarget.value = request
  rejectReason.value = ''
}

function closeRejectDialog(): void {
  if (reviewingId.value !== null) return
  rejectTarget.value = null
  rejectReason.value = ''
}

async function rejectRequest(): Promise<void> {
  const target = rejectTarget.value
  const reason = rejectReason.value.trim()
  if (!target || !reason) {
    appStore.showWarning('请填写拒绝原因')
    return
  }

  reviewingId.value = target.id
  try {
    await stepUp.run(() => resellerAPI.reviewWithdrawal(target.id, { action: 'reject', reason }))
    appStore.showSuccess('兑换申请已拒绝')
    rejectTarget.value = null
    rejectReason.value = ''
    await loadRequests(requests.value.page)
  } catch (error) {
    if (isStepUpCancelled(error)) return
    appStore.showError(extractApiErrorMessage(error, '拒绝失败'))
  } finally {
    reviewingId.value = null
  }
}

function changePage(page: number): void {
  void loadRequests(page)
}

function changePageSize(pageSize: number): void {
  requests.value.page_size = pageSize
  void loadRequests(1)
}

onMounted(() => void loadRequests(1))
</script>
