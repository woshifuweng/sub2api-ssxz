<template>
  <AppSectionShell
    title="代理工作台"
    subtitle="查看推广收益、直属用户和提现记录"
    eyebrow="RESELLER"
    icon="users"
  >
    <div class="reseller-page">
      <div v-if="loading" class="card p-10 text-center text-sm text-[var(--ssxz-text-muted)]">
        正在加载代理数据...
      </div>

      <template v-else-if="dashboard">
        <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="代理数据概览">
          <article class="card reseller-stat">
            <span>可申请提现</span>
            <strong>{{ formatCurrency(withdrawableAmount) }}</strong>
            <small>已申请待处理 {{ formatCurrency(dashboard.pending_withdraw) }}</small>
          </article>
          <article class="card reseller-stat">
            <span>累计返利</span>
            <strong>{{ formatCurrency(dashboard.aff_history_quota) }}</strong>
            <small>冻结中 {{ formatCurrency(dashboard.aff_frozen_quota) }}</small>
          </article>
          <article class="card reseller-stat">
            <span>邀请用户</span>
            <strong>{{ dashboard.recruit_count }}</strong>
            <small>当前返利比例 {{ formatPercent(dashboard.rebate_rate) }}</small>
          </article>
          <article class="card reseller-stat">
            <span>推广码</span>
            <strong class="break-all text-lg">{{ dashboard.aff_code || '尚未生成' }}</strong>
            <LiquidButton
              v-if="dashboard.aff_code"
              type="button"
              variant="outline"
              size="sm"
              @click="copyInviteLink"
            >
              <Icon name="copy" size="sm" />
              <span>复制邀请链接</span>
            </LiquidButton>
          </article>
        </section>

        <section class="grid gap-5 xl:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
          <form class="card p-5" @submit.prevent="submitWithdrawal">
            <div class="mb-5">
              <h2 class="text-base font-semibold text-[var(--ssxz-text)]">申请提现</h2>
              <p class="mt-1 text-sm text-[var(--ssxz-text-muted)]">
                提交后由平台审核，待处理金额不会重复占用。
              </p>
            </div>

            <div class="space-y-4">
              <Input
                v-model="withdrawForm.amount"
                type="number"
                label="提现金额"
                placeholder="输入金额"
                :disabled="submitting"
                required
              />
              <label class="block">
                <span class="input-label mb-1.5 block">收款方式</span>
                <select v-model="withdrawForm.method" class="input w-full" :disabled="submitting">
                  <option value="alipay">支付宝</option>
                  <option value="wechat">微信</option>
                  <option value="bank">银行卡</option>
                  <option value="manual">其他方式</option>
                </select>
              </label>
              <Input
                v-model="withdrawForm.account"
                label="收款账号"
                placeholder="填写账号或银行卡号"
                :disabled="submitting"
                required
              />
              <LiquidButton
                type="submit"
                class="w-full"
                size="default"
                :disabled="submitting || withdrawableAmount <= 0"
              >
                <Icon v-if="submitting" name="refresh" size="sm" class="animate-spin" />
                <span>{{ submitting ? '正在提交' : '提交申请' }}</span>
              </LiquidButton>
            </div>
          </form>

          <section class="card overflow-hidden">
            <header class="reseller-section-header">
              <div>
                <h2>提现记录</h2>
                <p>仅展示当前账号提交的申请。</p>
              </div>
              <LiquidButton type="button" variant="outline" size="sm" @click="loadAll">
                <Icon name="refresh" size="sm" />
                <span>刷新</span>
              </LiquidButton>
            </header>
            <div class="overflow-x-auto">
              <table class="reseller-table min-w-[640px]">
                <thead>
                  <tr><th>时间</th><th>金额</th><th>方式</th><th>状态</th><th>备注</th></tr>
                </thead>
                <tbody>
                  <tr v-for="item in withdrawals.items" :key="item.id">
                    <td>{{ formatDateTime(item.requested_at) }}</td>
                    <td class="font-medium text-[var(--ssxz-text)]">{{ formatCurrency(item.amount) }}</td>
                    <td>{{ methodLabel(item.method) }}</td>
                    <td><span :class="['reseller-status', `reseller-status--${item.status}`]">{{ statusLabel(item.status) }}</span></td>
                    <td>{{ item.note || '-' }}</td>
                  </tr>
                  <tr v-if="withdrawals.items.length === 0">
                    <td colspan="5" class="py-10 text-center text-[var(--ssxz-text-muted)]">暂无提现记录</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </section>

        <section class="card overflow-hidden">
          <header class="reseller-section-header">
            <div>
              <h2>直属用户</h2>
              <p>邮箱已脱敏，仅用于核对推广效果。</p>
            </div>
            <span class="text-sm text-[var(--ssxz-text-muted)]">共 {{ recruits.total }} 人</span>
          </header>
          <div class="overflow-x-auto">
            <table class="reseller-table min-w-[680px]">
              <thead>
                <tr><th>用户</th><th>加入时间</th><th>累计返利</th><th>近 30 天状态</th></tr>
              </thead>
              <tbody>
                <tr v-for="item in recruits.items" :key="item.user_id">
                  <td><strong>{{ item.username || '未设置昵称' }}</strong><small>{{ item.email }}</small></td>
                  <td>{{ item.joined_at ? formatDateTime(item.joined_at) : '-' }}</td>
                  <td>{{ formatCurrency(item.total_rebate) }}</td>
                  <td><span :class="['reseller-status', item.is_active ? 'reseller-status--approved' : '']">{{ item.is_active ? '活跃' : '暂无调用' }}</span></td>
                </tr>
                <tr v-if="recruits.items.length === 0">
                  <td colspan="4" class="py-10 text-center text-[var(--ssxz-text-muted)]">暂无直属用户</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import Input from '@/components/common/Input.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import resellerAPI, {
  type AgentDashboard,
  type RecruitRecord,
  type WithdrawRequest,
  type WithdrawStatus
} from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import { useAppStore } from '@/stores/app'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const loading = ref(true)
const submitting = ref(false)
const dashboard = ref<AgentDashboard | null>(null)
const recruits = ref<PaginatedResponse<RecruitRecord>>(emptyPage())
const withdrawals = ref<PaginatedResponse<WithdrawRequest>>(emptyPage())
const withdrawForm = reactive({ amount: '', method: 'alipay' as 'alipay' | 'wechat' | 'bank' | 'manual', account: '' })

const withdrawableAmount = computed(() => Math.max(0, (dashboard.value?.aff_quota ?? 0) - (dashboard.value?.pending_withdraw ?? 0)))

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
}

function formatPercent(value: number): string {
  return `${Number(value || 0).toFixed(2).replace(/\.00$/, '')}%`
}

function methodLabel(method: string): string {
  return { alipay: '支付宝', wechat: '微信', bank: '银行卡', manual: '其他方式' }[method] || method
}

function statusLabel(status: WithdrawStatus): string {
  return { pending: '待审核', approved: '已通过', rejected: '已拒绝' }[status]
}

async function loadAll(): Promise<void> {
  loading.value = true
  try {
    const [dashboardData, recruitData, withdrawalData] = await Promise.all([
      resellerAPI.getAgentDashboard(),
      resellerAPI.listRecruits(),
      resellerAPI.listMyWithdrawals()
    ])
    dashboard.value = dashboardData
    recruits.value = recruitData
    withdrawals.value = withdrawalData
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '代理数据加载失败'))
  } finally {
    loading.value = false
  }
}

async function submitWithdrawal(): Promise<void> {
  const amount = Number(withdrawForm.amount)
  const account = withdrawForm.account.trim()
  if (!Number.isFinite(amount) || amount <= 0 || amount > withdrawableAmount.value) {
    appStore.showWarning('请输入不超过可提现额度的有效金额')
    return
  }
  if (!account) {
    appStore.showWarning('请填写收款账号')
    return
  }

  submitting.value = true
  try {
    await resellerAPI.requestWithdraw({ amount, method: withdrawForm.method, account_info: { account } })
    withdrawForm.amount = ''
    withdrawForm.account = ''
    appStore.showSuccess('提现申请已提交')
    await loadAll()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '提现申请提交失败'))
  } finally {
    submitting.value = false
  }
}

async function copyInviteLink(): Promise<void> {
  if (!dashboard.value?.aff_code) return
  const link = `${window.location.origin}/register?aff=${encodeURIComponent(dashboard.value.aff_code)}`
  try {
    await navigator.clipboard.writeText(link)
    appStore.showSuccess('邀请链接已复制')
  } catch {
    appStore.showError('复制失败，请手动复制推广码')
  }
}

onMounted(() => void loadAll())
</script>

<style scoped>
.reseller-page { display: grid; gap: 1.25rem; }
.reseller-stat { display: flex; min-height: 9.5rem; flex-direction: column; gap: .55rem; padding: 1.25rem; }
.reseller-stat > span, .reseller-stat > small { color: var(--ssxz-text-muted); font-size: .8rem; }
.reseller-stat > strong { color: var(--ssxz-text); font-size: 1.55rem; line-height: 1.2; }
.reseller-stat > :last-child { margin-top: auto; }
.reseller-section-header { display: flex; min-height: 4.75rem; align-items: center; justify-content: space-between; gap: 1rem; padding: 1rem 1.25rem; border-bottom: 1px solid var(--ssxz-border); }
.reseller-section-header h2 { color: var(--ssxz-text); font-size: 1rem; font-weight: 600; }
.reseller-section-header p { margin-top: .25rem; color: var(--ssxz-text-muted); font-size: .8rem; }
.reseller-table { width: 100%; font-size: .82rem; text-align: left; }
.reseller-table th { padding: .75rem 1rem; color: var(--ssxz-text-muted); font-weight: 500; background: var(--ssxz-surface-sunken); }
.reseller-table td { padding: .85rem 1rem; color: var(--ssxz-text-secondary); border-top: 1px solid var(--ssxz-border); }
.reseller-table td strong, .reseller-table td small { display: block; }
.reseller-table td small { margin-top: .2rem; color: var(--ssxz-text-muted); }
.reseller-status { display: inline-flex; align-items: center; min-height: 1.5rem; padding: 0 .55rem; border: 1px solid var(--ssxz-border); border-radius: 999px; color: var(--ssxz-text-muted); font-size: .72rem; }
.reseller-status--pending { border-color: var(--ssxz-warning-border, #854d0e); color: var(--ssxz-warning-text, #f59e0b); }
.reseller-status--approved { border-color: var(--ssxz-success-border, #166534); color: var(--ssxz-success-text, #22c55e); }
.reseller-status--rejected { border-color: var(--ssxz-danger-border, #991b1b); color: var(--ssxz-danger-text, #ef4444); }
</style>
