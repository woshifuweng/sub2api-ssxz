<template>
  <AppSectionShell
    title="我的返利"
    subtitle="查看可兑换余额、累计佣金与推广进展"
    eyebrow="RESELLER"
    icon="users"
  >
    <div class="reseller-page">
      <div
        v-if="loading"
        class="card p-10 text-center text-sm text-[var(--ssxz-text-muted)]"
      >
        正在加载返利数据...
      </div>

      <div v-else-if="loadError" class="card reseller-empty">
        <Icon name="exclamationTriangle" size="lg" />
        <strong>{{ loadError }}</strong>
        <LiquidButton type="button" variant="outline" size="sm" @click="loadPage">
          <Icon name="refresh" size="sm" />
          <span>重新加载</span>
        </LiquidButton>
      </div>

      <template v-else-if="dashboard">
        <section class="reseller-stats" aria-label="返利数据概览">
          <article class="card reseller-stat">
            <span>可兑换余额</span>
            <strong>{{ formatNumber(availableBalance) }} 额度</strong>
            <small>待审核 {{ formatNumber(dashboard.pending_withdraw) }} 额度</small>
          </article>
          <article class="card reseller-stat">
            <span>累计佣金</span>
            <strong>{{ cumulativeCommission }}<span v-if="cumulativeCommission !== '--'"> 额度</span></strong>
            <small>尚未成熟 {{ formatNumber(dashboard.aff_frozen_quota) }} 额度</small>
          </article>
          <article class="card reseller-stat">
            <span>招募人数</span>
            <strong>{{ dashboard.recruit_count }}</strong>
            <small>当前比例 {{ formatPercent(dashboard.rebate_rate) }}</small>
          </article>
          <article class="card reseller-stat">
            <span>推广码</span>
            <strong class="break-all text-lg">{{ dashboard.aff_code || '--' }}</strong>
            <small>用于生成专属邀请链接</small>
          </article>
        </section>

        <section class="card reseller-panel">
          <header class="reseller-panel__header">
            <div>
              <h2>邀请链接</h2>
              <p>分享此链接，新用户注册后会自动归入你的推广关系。</p>
            </div>
          </header>
          <div class="reseller-invite-row">
            <code>{{ inviteLink || '推广码尚未生成' }}</code>
            <LiquidButton
              type="button"
              variant="outline"
              size="sm"
              :disabled="!inviteLink"
              @click="copyInviteLink"
            >
              <Icon name="copy" size="sm" />
              <span>一键复制</span>
            </LiquidButton>
          </div>
        </section>

        <section class="card reseller-panel overflow-hidden">
          <header class="reseller-panel__header">
            <div>
              <h2>最近兑换记录</h2>
              <p>展示最近 5 条申请及处理结果。</p>
            </div>
            <LiquidButton
              as="RouterLink"
              to="/app/reseller/withdrawals"
              variant="outline"
              size="sm"
            >
              <span>查看全部</span>
              <Icon name="arrowRight" size="sm" />
            </LiquidButton>
          </header>

          <div class="overflow-x-auto">
            <table class="reseller-table min-w-[560px]">
              <thead>
                <tr>
                  <th>申请时间</th>
                  <th>额度</th>
                  <th>状态</th>
                  <th>说明</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in recentRequests" :key="item.id">
                  <td>{{ formatRelativeTime(item.requested_at) }}</td>
                  <td class="font-medium text-[var(--ssxz-text)]">
                    {{ formatNumber(item.amount) }} 额度
                  </td>
                  <td><WithdrawalStatusBadge :status="item.status" /></td>
                  <td>{{ item.status === 'rejected' ? item.note || '未提供原因' : '--' }}</td>
                </tr>
                <tr v-if="recentRequests.length === 0">
                  <td colspan="4" class="py-10 text-center text-[var(--ssxz-text-muted)]">
                    暂无兑换记录
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <div class="reseller-footer-action">
          <div>
            <strong>将返利转入账户余额</strong>
            <span>提交后由管理员审核，通过后自动增加站内余额。</span>
          </div>
          <LiquidButton as="RouterLink" to="/app/reseller/withdrawals" size="default">
            <Icon name="swap" size="sm" />
            <span>兑换余额</span>
          </LiquidButton>
        </div>
      </template>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import WithdrawalStatusBadge from '@/components/reseller/WithdrawalStatusBadge.vue'
import resellerAPI, { type WithdrawRequest } from '@/api/reseller'
import { useResellerStore } from '@/stores/reseller'
import { useClipboard } from '@/composables/useClipboard'
import { formatNumber, formatRelativeTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const resellerStore = useResellerStore()
const { copyToClipboard } = useClipboard()
const loading = ref(true)
const loadError = ref('')
const recentRequests = ref<WithdrawRequest[]>([])

const dashboard = computed(() => resellerStore.dashboard)
const availableBalance = computed(() => Math.max(
  0,
  (dashboard.value?.aff_quota ?? 0) - (dashboard.value?.pending_withdraw ?? 0)
))
const cumulativeCommission = computed(() => (
  typeof dashboard.value?.commission_earned === 'number'
    ? formatNumber(dashboard.value.commission_earned)
    : '--'
))
const inviteLink = computed(() => {
  const code = dashboard.value?.aff_code
  if (!code) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(code)}`
})

function formatPercent(value: number | undefined): string {
  if (typeof value !== 'number') return '--'
  const percent = value <= 1 ? value * 100 : value
  return `${percent.toFixed(2).replace(/\.00$/, '')}%`
}

async function loadPage(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const [, history] = await Promise.all([
      resellerStore.fetchDashboard(),
      resellerAPI.listMyWithdrawals(1, 5)
    ])
    recentRequests.value = Array.isArray(history.items) ? history.items : []
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, '返利数据加载失败')
  } finally {
    loading.value = false
  }
}

async function copyInviteLink(): Promise<void> {
  await copyToClipboard(inviteLink.value, '邀请链接已复制')
}

onMounted(() => void loadPage())
</script>

<style scoped>
.reseller-page {
  display: grid;
  gap: 1.25rem;
}

.reseller-stats {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.reseller-stat {
  display: flex;
  min-height: 8.75rem;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1.25rem;
}

.reseller-stat span,
.reseller-stat small {
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.reseller-stat strong {
  color: var(--ssxz-text);
  font-size: 1.55rem;
  line-height: 1.2;
}

.reseller-stat small {
  margin-top: auto;
}

.reseller-panel__header {
  display: flex;
  min-height: 4.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 1rem 1.25rem;
}

.reseller-panel__header h2 {
  color: var(--ssxz-text);
  font-size: 1rem;
  font-weight: 600;
}

.reseller-panel__header p {
  margin-top: 0.25rem;
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.reseller-invite-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.25rem;
}

.reseller-invite-row code {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-surface-sunken);
  padding: 0.75rem;
  color: var(--ssxz-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reseller-table {
  width: 100%;
  font-size: 0.82rem;
  text-align: left;
}

.reseller-table th {
  background: var(--ssxz-surface-sunken);
  padding: 0.75rem 1rem;
  color: var(--ssxz-text-muted);
  font-weight: 500;
}

.reseller-table td {
  border-top: 1px solid var(--ssxz-border);
  padding: 0.85rem 1rem;
  color: var(--ssxz-text-secondary);
}

.reseller-footer-action {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid var(--ssxz-border);
  padding-top: 1.25rem;
}

.reseller-footer-action div {
  display: grid;
  gap: 0.2rem;
}

.reseller-footer-action strong {
  color: var(--ssxz-text);
  font-size: 0.92rem;
}

.reseller-footer-action span {
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.reseller-empty {
  display: grid;
  justify-items: center;
  gap: 1rem;
  padding: 2.5rem;
  color: var(--ssxz-text-muted);
  text-align: center;
}

@media (max-width: 1023px) {
  .reseller-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 639px) {
  .reseller-stats {
    grid-template-columns: minmax(0, 1fr);
  }

  .reseller-panel__header,
  .reseller-invite-row,
  .reseller-footer-action {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
