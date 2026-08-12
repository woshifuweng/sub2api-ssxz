<template>
  <BaseDialog
    :show="show"
    title="下线详情"
    width="wide"
    :close-on-click-outside="true"
    @close="emit('close')"
  >
    <div v-if="loadingDetail" class="drawer-empty">正在加载下线详情...</div>
    <div v-else-if="detail" class="recruit-detail">
      <header class="detail-header">
        <div class="min-w-0">
          <h3>{{ detail.email || '--' }}</h3>
          <p>{{ detail.username || `用户 ${detail.user_id}` }}</p>
        </div>
        <span :class="['status-badge', statusClass(detail)]">{{ statusLabel(detail) }}</span>
      </header>

      <dl class="detail-grid">
        <div><dt>masked_email</dt><dd>{{ detail.email || '--' }}</dd></div>
        <div><dt>status</dt><dd>{{ statusLabel(detail) }}</dd></div>
        <div><dt>reseller_role</dt><dd>{{ detail.reseller_role || '--' }}</dd></div>
        <div><dt>commission_rate</dt><dd>{{ formatRate(detail.commission_rate) }}</dd></div>
        <div><dt>created_at</dt><dd>{{ formatDate(detail.created_at || detail.joined_at) }}</dd></div>
      </dl>

      <div class="detail-tabs" role="tablist" aria-label="下线记录">
        <button
          type="button"
          :class="['detail-tab', { 'detail-tab--active': activeTab === 'logs' }]"
          role="tab"
          :aria-selected="activeTab === 'logs'"
          @click="selectTab('logs')"
        >
          消费记录
        </button>
        <button
          type="button"
          :class="['detail-tab', { 'detail-tab--active': activeTab === 'recharges' }]"
          role="tab"
          :aria-selected="activeTab === 'recharges'"
          @click="selectTab('recharges')"
        >
          充值记录
        </button>
      </div>

      <div v-if="loadError" class="drawer-empty">{{ loadError }}</div>
      <div v-else-if="loadingRecords" class="drawer-empty">正在加载记录...</div>
      <template v-else>
        <div v-if="activeTab === 'logs'" class="records-wrap">
          <table class="records-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>模型</th>
                <th>请求类型</th>
                <th>Token</th>
                <th class="text-right">实际消费</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in logs.items" :key="item.id">
                <td>{{ formatDate(item.created_at) }}</td>
                <td>{{ item.model || '--' }}</td>
                <td>{{ requestTypeLabel(item.request_type) }}</td>
                <td>{{ item.total_tokens.toLocaleString() }}</td>
                <td class="text-right">{{ formatMoney(item.actual_cost) }}</td>
              </tr>
              <tr v-if="logs.items.length === 0">
                <td colspan="5" class="empty-cell">暂无消费记录</td>
              </tr>
            </tbody>
          </table>
          <Pagination
            v-if="logs.total > 0"
            :page="logs.page"
            :page-size="logs.page_size"
            :total="logs.total"
            :show-page-size-selector="false"
            @update:page="loadLogs"
          />
        </div>

        <div v-else class="records-wrap">
          <table class="records-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>来源</th>
                <th class="text-right">到账金额</th>
                <th>备注</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in recharges.items" :key="item.id">
                <td>{{ formatDate(item.created_at) }}</td>
                <td>{{ item.event_type }}</td>
                <td class="text-right">{{ formatMoney(item.amount) }}</td>
                <td>{{ item.note || '--' }}</td>
              </tr>
              <tr v-if="recharges.items.length === 0">
                <td colspan="4" class="empty-cell">暂无充值记录</td>
              </tr>
            </tbody>
          </table>
          <Pagination
            v-if="recharges.total > 0"
            :page="recharges.page"
            :page-size="recharges.page_size"
            :total="recharges.total"
            :show-page-size-selector="false"
            @update:page="loadRecharges"
          />
        </div>
      </template>

      <footer v-if="detail.reseller_role" class="drawer-footer">
        <LiquidButton type="button" variant="outline" size="sm" @click="viewRecruitDownline">
          查看TA的下线
        </LiquidButton>
      </footer>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <LiquidButton type="button" variant="outline" size="sm" @click="emit('close')">
          关闭
        </LiquidButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Pagination from '@/components/common/Pagination.vue'
import resellerAPI, {
  type RecruitRecord,
  type RecruitRecharge,
  type RecruitUsageLog
} from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  show: boolean
  userId: number | null
  recruit: RecruitRecord | null
}>()

const emit = defineEmits<{ (event: 'close'): void }>()

type DetailTab = 'logs' | 'recharges'

const detail = ref<RecruitRecord | null>(props.recruit)
const selectedUserId = ref<number | null>(props.userId)
const activeTab = ref<DetailTab>('logs')
const loadingDetail = ref(false)
const loadingRecords = ref(false)
const loadError = ref('')
const logs = ref<PaginatedResponse<RecruitUsageLog>>(emptyPage())
const recharges = ref<PaginatedResponse<RecruitRecharge>>(emptyPage())

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
}

async function reload(userId: number, initialRecruit: RecruitRecord | null): Promise<void> {
  detail.value = initialRecruit
  loadingDetail.value = true
  loadError.value = ''
  try {
    detail.value = await resellerAPI.getRecruitDetail(userId)
    await loadActivePage(1)
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, '下线详情加载失败')
  } finally {
    loadingDetail.value = false
  }
}

async function loadActivePage(page: number): Promise<void> {
  const userId = selectedUserId.value
  if (!userId) return
  loadingRecords.value = true
  loadError.value = ''
  try {
    if (activeTab.value === 'logs') {
      logs.value = await resellerAPI.listRecruitLogs(userId, page, 20)
    } else {
      recharges.value = await resellerAPI.listRecruitRecharges(userId, page, 20)
    }
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, '记录加载失败')
  } finally {
    loadingRecords.value = false
  }
}

function selectTab(tab: DetailTab): void {
  if (activeTab.value === tab) return
  activeTab.value = tab
  void loadActivePage(1)
}

function loadLogs(page: number): void {
  void loadActivePage(page)
}

function loadRecharges(page: number): void {
  void loadActivePage(page)
}

function viewRecruitDownline(): void {
  if (!detail.value) return
  const userId = detail.value.user_id
  selectedUserId.value = userId
  // Keep this drawer mounted and reload the selected recruit as the next scope.
  void reload(userId, detail.value)
}

function statusLabel(recruit: RecruitRecord): string {
  if (recruit.status) return recruit.status
  return recruit.is_active ? 'active' : 'inactive'
}

function statusClass(recruit: RecruitRecord): string {
  return recruit.is_active || recruit.status === 'active' ? 'status-badge--active' : 'status-badge--muted'
}

function formatRate(value: number): string {
  return `${value.toFixed(2).replace(/\.00$/, '')}%`
}

function formatMoney(value: number): string {
  return `$${value.toFixed(4)}`
}

function formatDate(value?: string): string {
  return value ? formatDateTime(value) : '--'
}

function requestTypeLabel(value: number): string {
  const labels: Record<number, string> = {
    0: 'unknown',
    1: 'sync',
    2: 'stream',
    3: 'ws_v2',
    4: 'cyber',
    5: 'live'
  }
  return labels[value] || String(value)
}

watch(
  () => [props.show, props.userId] as const,
  ([show, userId]) => {
    if (show && userId) {
      selectedUserId.value = userId
      activeTab.value = 'logs'
      logs.value = emptyPage()
      recharges.value = emptyPage()
      void reload(userId, props.recruit)
    }
  }
)
</script>

<style scoped>
.recruit-detail { display: grid; gap: 1.25rem; }
.detail-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; border-bottom: 1px solid var(--ssxz-border); padding-bottom: 1rem; }
.detail-header h3 { overflow: hidden; color: var(--ssxz-text); font-size: 1rem; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.detail-header p { margin-top: 0.25rem; color: var(--ssxz-text-muted); font-size: 0.8rem; }
.detail-grid { display: grid; gap: 1px; overflow: hidden; border: 1px solid var(--ssxz-border); border-radius: var(--ssxz-radius-card); background: var(--ssxz-border); grid-template-columns: repeat(2, minmax(0, 1fr)); }
.detail-grid > div { background: var(--ssxz-surface); padding: 0.8rem; }
.detail-grid dt { color: var(--ssxz-text-muted); font-size: 0.72rem; }
.detail-grid dd { margin-top: 0.3rem; color: var(--ssxz-text); font-size: 0.85rem; font-weight: 500; }
.status-badge { display: inline-flex; flex: none; border-radius: 999px; padding: 0.2rem 0.55rem; font-size: 0.72rem; font-weight: 600; }
.status-badge--active { background: color-mix(in srgb, var(--ssxz-success) 12%, transparent); color: var(--ssxz-success); }
.status-badge--muted { background: color-mix(in srgb, var(--ssxz-text-muted) 12%, transparent); color: var(--ssxz-text-muted); }
.detail-tabs { display: flex; gap: 0.25rem; border-bottom: 1px solid var(--ssxz-border); }
.detail-tab { border-bottom: 2px solid transparent; padding: 0.65rem 0.9rem; color: var(--ssxz-text-muted); font-size: 0.82rem; }
.detail-tab--active { border-color: var(--ssxz-accent); color: var(--ssxz-text); }
.records-wrap { overflow-x: auto; }
.records-table { width: 100%; min-width: 620px; border-collapse: collapse; font-size: 0.8rem; }
.records-table th, .records-table td { border-bottom: 1px solid var(--ssxz-border); padding: 0.7rem 0.8rem; text-align: left; }
.records-table th { color: var(--ssxz-text-muted); font-weight: 500; }
.records-table td { color: var(--ssxz-text-secondary); }
.empty-cell, .drawer-empty { padding: 2rem; color: var(--ssxz-text-muted); text-align: center; }
.drawer-footer { display: flex; justify-content: flex-start; border-top: 1px solid var(--ssxz-border); padding-top: 1rem; }
@media (max-width: 640px) { .detail-grid { grid-template-columns: minmax(0, 1fr); } .detail-header { align-items: flex-start; flex-direction: column; } }
</style>
