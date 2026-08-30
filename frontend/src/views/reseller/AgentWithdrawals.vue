<template>
  <AppSectionShell
    title="兑换记录"
    subtitle="将返利转入账户余额，并查看处理进度"
    eyebrow="RESELLER"
    icon="swap"
  >
    <div class="withdrawal-page">
      <section class="card conversion-panel">
        <div class="conversion-copy">
          <span>当前可兑换</span>
          <strong>{{ formatNumber(availableBalance) }} 额度</strong>
          <small>提现方式：余额转入</small>
          <p>申请审核通过后，金额会自动转入当前账户余额。</p>
        </div>

        <div class="conversion-form">
          <p>申请余额转入需要经过审核，金额将转入当前账户余额。</p>
          <LiquidButton
            type="button"
            :disabled="submitting || availableBalance < 5"
            @click="withdrawModalOpen = true"
          >
            <Icon name="swap" size="sm" />
            <span>申请提现</span>
          </LiquidButton>
        </div>

      </section>

      <section class="card overflow-hidden">
        <header class="withdrawal-header">
          <div>
            <h2>历史记录</h2>
            <p>待审核记录可以由本人撤销，完成后不可更改。</p>
          </div>
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="loading"
            @click="loadPage"
          >
            <Icon name="refresh" size="sm" />
            <span>刷新</span>
          </LiquidButton>
        </header>

        <div v-if="loading" class="withdrawal-empty">正在加载记录...</div>
        <div v-else-if="loadError" class="withdrawal-empty">{{ loadError }}</div>
        <div v-else class="overflow-x-auto">
          <table class="withdrawal-table min-w-[700px]">
            <thead>
              <tr>
                <th>额度</th>
                <th>状态</th>
                <th>申请时间</th>
                <th>说明</th>
                <th class="text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in requests.items" :key="item.id">
                <td class="font-medium text-[var(--ssxz-text)]">
                  {{ formatNumber(item.amount) }} 额度
                </td>
                <td><WithdrawalStatusBadge :status="item.status" /></td>
                <td>{{ formatRelativeTime(item.requested_at) }}</td>
                <td>{{ item.status === 'rejected' ? item.note || '未提供原因' : '--' }}</td>
                <td class="text-right">
                  <LiquidButton
                    v-if="item.status === 'pending'"
                    type="button"
                    variant="outline"
                    size="sm"
                    @click="cancelTarget = item"
                  >
                    <span>撤销</span>
                  </LiquidButton>
                  <span v-else>--</span>
                </td>
              </tr>
              <tr v-if="requests.items.length === 0">
                <td colspan="5" class="py-10 text-center text-[var(--ssxz-text-muted)]">
                  暂无兑换记录
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="requests.total > 0"
          :page="requests.page"
          :page-size="requests.page_size"
          :total="requests.total"
          :show-page-size-selector="false"
          @update:page="changePage"
        />
      </section>
    </div>

    <BaseDialog
      :show="withdrawModalOpen"
      title="申请提现"
      width="narrow"
      :close-on-click-outside="true"
      @close="withdrawModalOpen = false"
    >
      <form id="withdraw-form" class="withdraw-form" @submit.prevent="submitConversion">
        <label for="withdraw-available-balance">当前可用余额</label>
        <input
          id="withdraw-available-balance"
          class="input"
          :value="formatNumber(availableBalance)"
          readonly
        />
        <label for="withdraw-amount">提现金额</label>
        <input
          id="withdraw-amount"
          v-model="amount"
          class="input"
          type="number"
          inputmode="decimal"
          min="5"
          :max="availableBalance"
          step="0.01"
          placeholder="最低 5"
          :disabled="submitting"
          required
        />
        <small v-if="availableBalance < 5">当前可用余额不足最低提现金额 5。</small>
      </form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <LiquidButton type="button" variant="outline" size="sm" @click="withdrawModalOpen = false">
            取消
          </LiquidButton>
          <LiquidButton
            type="submit"
            form="withdraw-form"
            size="sm"
            :disabled="submitting || availableBalance < 5"
          >
            {{ submitting ? '提交中...' : '确认提现' }}
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="!!cancelTarget"
      title="撤销兑换申请"
      :message="cancelMessage"
      confirm-text="确认撤销"
      cancel-text="保留申请"
      :danger="true"
      @confirm="confirmCancel"
      @cancel="cancelTarget = null"
    />
    <TotpStepUpDialog :controller="stepUp" />
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Pagination from '@/components/common/Pagination.vue'
import WithdrawalStatusBadge from '@/components/reseller/WithdrawalStatusBadge.vue'
import resellerAPI, { type WithdrawRequest } from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import { useAppStore } from '@/stores/app'
import { useResellerStore } from '@/stores/reseller'
import { formatNumber, formatRelativeTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import { isStepUpCancelled, useStepUp } from '@/composables/useStepUp'

const appStore = useAppStore()
const resellerStore = useResellerStore()
const stepUp = useStepUp()
const amount = ref('')
const submitting = ref(false)
const pendingConversion = ref<{ amount: number; idempotencyKey: string } | null>(null)
const withdrawModalOpen = ref(false)
const loading = ref(true)
const loadError = ref('')
const cancelTarget = ref<WithdrawRequest | null>(null)
const requests = ref<PaginatedResponse<WithdrawRequest>>(emptyPage())

const availableBalance = computed(() => Math.max(
  0,
  (resellerStore.dashboard?.aff_quota ?? 0) - (resellerStore.dashboard?.pending_withdraw ?? 0)
))
const cancelMessage = computed(() => (
  cancelTarget.value
    ? `确认撤销 ${formatNumber(cancelTarget.value.amount)} 额度的兑换申请？`
    : ''
))

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
}

function getConversionIdempotencyKey(value: number): string {
  if (pendingConversion.value?.amount === value) {
    return pendingConversion.value.idempotencyKey
  }
  const requestID = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
  const idempotencyKey = `balance-conversion-${requestID}`
  pendingConversion.value = { amount: value, idempotencyKey }
  return idempotencyKey
}

async function loadPage(page = requests.value.page || 1): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const [, history] = await Promise.all([
      resellerStore.fetchDashboard(),
      resellerAPI.listMyWithdrawals(page, requests.value.page_size)
    ])
    requests.value = {
      ...history,
      items: Array.isArray(history.items) ? history.items : []
    }
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, '兑换记录加载失败')
  } finally {
    loading.value = false
  }
}

async function submitConversion(): Promise<void> {
  const value = Number(amount.value)
  if (!Number.isFinite(value) || value < 5 || value > availableBalance.value) {
    appStore.showWarning(`请输入 5 至 ${formatNumber(availableBalance.value)} 之间的额度`)
    return
  }

  submitting.value = true
  try {
    await stepUp.run(() => resellerAPI.requestBalanceConversion(value, {
      idempotencyKey: getConversionIdempotencyKey(value)
    }))
    pendingConversion.value = null
    amount.value = ''
    withdrawModalOpen.value = false
    appStore.showSuccess('兑换申请已提交')
    await loadPage(1)
  } catch (error) {
    if (isStepUpCancelled(error)) return
    appStore.showError(extractApiErrorMessage(error, '兑换申请提交失败'))
  } finally {
    submitting.value = false
  }
}

async function confirmCancel(): Promise<void> {
  const target = cancelTarget.value
  if (!target) return

  cancelTarget.value = null
  try {
    await resellerAPI.cancelWithdrawal(target.id)
    appStore.showSuccess('兑换申请已撤销')
    await loadPage(requests.value.page)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '撤销失败'))
  }
}

function changePage(page: number): void {
  void loadPage(page)
}

onMounted(() => void loadPage(1))
</script>

<style scoped>
.withdrawal-page {
  display: grid;
  gap: 1.25rem;
}

.conversion-panel {
  display: grid;
  gap: 2rem;
  grid-template-columns: minmax(0, 0.8fr) minmax(320px, 1.2fr);
  padding: 1.5rem;
}

.conversion-copy {
  display: grid;
  align-content: center;
  gap: 0.5rem;
}

.conversion-copy span,
.conversion-copy p,
.conversion-form small {
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.conversion-copy strong {
  color: var(--ssxz-text);
  font-size: 2rem;
  line-height: 1.2;
}

.conversion-form {
  display: grid;
  align-content: center;
  gap: 0.55rem;
}

.withdraw-form {
  display: grid;
  gap: 0.65rem;
}

.withdraw-form label {
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 600;
}

.withdraw-form small {
  color: var(--ssxz-text-muted);
  font-size: 0.78rem;
}

.conversion-form label {
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 600;
}

.conversion-form__row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.withdrawal-header {
  display: flex;
  min-height: 4.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 1rem 1.25rem;
}

.withdrawal-header h2 {
  color: var(--ssxz-text);
  font-size: 1rem;
  font-weight: 600;
}

.withdrawal-header p {
  margin-top: 0.25rem;
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.withdrawal-table {
  width: 100%;
  font-size: 0.82rem;
  text-align: left;
}

.withdrawal-table th {
  background: var(--ssxz-surface-sunken);
  padding: 0.75rem 1rem;
  color: var(--ssxz-text-muted);
  font-weight: 500;
}

.withdrawal-table td {
  border-top: 1px solid var(--ssxz-border);
  padding: 0.85rem 1rem;
  color: var(--ssxz-text-secondary);
}

.withdrawal-empty {
  padding: 2.5rem;
  color: var(--ssxz-text-muted);
  text-align: center;
}

@media (max-width: 767px) {
  .conversion-panel {
    grid-template-columns: minmax(0, 1fr);
  }

  .conversion-form__row,
  .withdrawal-header {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
