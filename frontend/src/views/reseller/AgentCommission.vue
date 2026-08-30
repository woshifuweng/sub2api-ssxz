<template>
  <AppSectionShell
    title="佣金明细"
    subtitle="查看下线消费带来的逐笔佣金记录"
    eyebrow="RESELLER"
    icon="document"
  >
    <div class="commission-page">
      <section class="card commission-summary">
        <div>
          <span>筛选期间佣金</span>
          <strong>{{ formatCurrency(commission.total_commission_usd) }}</strong>
        </div>
        <div class="commission-total">
          <span>记录总数</span>
          <strong>{{ commission.total }}</strong>
        </div>
      </section>

      <section class="card overflow-hidden">
        <header class="commission-header">
          <div>
            <h2>佣金记录</h2>
            <p>来源账号仅展示脱敏邮箱。</p>
          </div>
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="loading"
            @click="loadPage(1)"
          >
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            <span>刷新</span>
          </LiquidButton>
        </header>

        <div class="commission-filters" role="group" aria-label="时间范围">
          <LiquidButton
            v-for="preset in presets"
            :key="preset.value"
            type="button"
            size="sm"
            :variant="selectedPreset === preset.value ? 'default' : 'outline'"
            :disabled="loading"
            @click="selectPreset(preset.value)"
          >
            <span>{{ preset.label }}</span>
          </LiquidButton>
          <label v-if="selectedPreset === 'custom'" class="date-field">
            <span>开始</span>
            <input v-model="customStart" type="date" class="input h-9" @change="loadPage(1)" />
          </label>
          <label v-if="selectedPreset === 'custom'" class="date-field">
            <span>结束</span>
            <input v-model="customEnd" type="date" class="input h-9" @change="loadPage(1)" />
          </label>
        </div>

        <div v-if="loadError" class="commission-empty">{{ loadError }}</div>
        <div v-else-if="loading" class="commission-empty">正在加载佣金记录...</div>
        <div v-else class="overflow-x-auto">
          <table class="commission-table min-w-[760px]">
            <thead>
              <tr>
                <th>时间</th>
                <th>来源账号</th>
                <th>对方消费</th>
                <th>我的佣金</th>
                <th>佣金率</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in commission.items" :key="`${item.time}-${item.source_user_masked_email}-${item.commission_usd}`">
                <td>{{ formatDateTime(item.time) }}</td>
                <td>{{ maskEmail(item.source_user_masked_email) }}</td>
                <td>{{ formatCurrency(item.source_consumption_usd) }}</td>
                <td class="font-medium text-[var(--ssxz-text)]">{{ formatCurrency(item.commission_usd) }}</td>
                <td>{{ formatRate(item.commission_rate) }}</td>
              </tr>
              <tr v-if="commission.items.length === 0">
                <td colspan="5" class="py-10 text-center text-[var(--ssxz-text-muted)]">暂无佣金记录</td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="commission.total > 0"
          :page="commission.page"
          :page-size="commission.page_size"
          :total="commission.total"
          :show-page-size-selector="false"
          @update:page="loadPage"
        />
      </section>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Pagination from '@/components/common/Pagination.vue'
import resellerAPI, { type CommissionResponse } from '@/api/reseller'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'

type Preset = 'week' | 'month' | 'last_month' | 'custom'

const presets: Array<{ value: Preset; label: string }> = [
  { value: 'week', label: '本周' },
  { value: 'month', label: '本月' },
  { value: 'last_month', label: '上月' },
  { value: 'custom', label: '自定义' }
]

const selectedPreset = ref<Preset>('month')
const customStart = ref('')
const customEnd = ref('')
const loading = ref(true)
const loadError = ref('')
const commission = ref<CommissionResponse>(emptyResponse())

function emptyResponse(): CommissionResponse {
  return { items: [], total: 0, total_commission_usd: 0, page: 1, page_size: 50 }
}

function toDateInput(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function getRange(): { start: string; end: string } {
  if (selectedPreset.value === 'custom') {
    return { start: customStart.value, end: customEnd.value }
  }

  const now = new Date()
  const end = toDateInput(now)
  const start = new Date(now)
  if (selectedPreset.value === 'week') {
    start.setDate(now.getDate() - 6)
  } else if (selectedPreset.value === 'last_month') {
    start.setMonth(now.getMonth() - 1, 1)
    const lastDay = new Date(now.getFullYear(), now.getMonth(), 0)
    return { start: toDateInput(start), end: toDateInput(lastDay) }
  } else {
    start.setDate(1)
  }
  return { start: toDateInput(start), end }
}

function selectPreset(preset: Preset): void {
  selectedPreset.value = preset
  if (preset !== 'custom') {
    void loadPage(1)
  }
}

async function loadPage(page = 1): Promise<void> {
  loading.value = true
  loadError.value = ''
  const range = getRange()
  try {
    commission.value = await resellerAPI.listCommission(page, commission.value.page_size, range.start, range.end)
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, '佣金记录加载失败')
  } finally {
    loading.value = false
  }
}

function maskEmail(email: string | undefined): string {
  if (!email) return '--'
  if (email.includes('*')) return email
  const [local, domain] = email.split('@')
  if (!domain) return '***'
  return `${local?.slice(0, 1) || '*'}***@${domain}`
}

function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`
}

onMounted(() => void loadPage(1))
</script>

<style scoped>
.commission-page { display: grid; gap: 1.25rem; }
.commission-summary { display: flex; gap: 3rem; padding: 1.25rem 1.5rem; }
.commission-summary div { display: grid; gap: 0.4rem; }
.commission-summary span, .commission-header p, .date-field span { color: var(--ssxz-text-muted); font-size: 0.8rem; }
.commission-summary strong { color: var(--ssxz-text); font-size: 1.75rem; }
.commission-total strong { font-size: 1.5rem; }
.commission-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; border-bottom: 1px solid var(--ssxz-border); padding: 1rem 1.25rem; }
.commission-header h2 { color: var(--ssxz-text); font-size: 1rem; font-weight: 600; }
.commission-header p { margin-top: 0.25rem; }
.commission-filters { display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; padding: 1rem 1.25rem; }
.date-field { display: flex; align-items: center; gap: 0.4rem; }
.commission-table { width: 100%; font-size: 0.82rem; text-align: left; }
.commission-table th { background: var(--ssxz-surface-sunken); padding: 0.75rem 1rem; color: var(--ssxz-text-muted); font-weight: 500; }
.commission-table td { border-top: 1px solid var(--ssxz-border); padding: 0.85rem 1rem; color: var(--ssxz-text-secondary); }
.commission-empty { padding: 2.5rem; color: var(--ssxz-text-muted); text-align: center; }
@media (max-width: 639px) { .commission-summary { flex-direction: column; gap: 1rem; } .commission-header { align-items: stretch; flex-direction: column; } }
</style>
