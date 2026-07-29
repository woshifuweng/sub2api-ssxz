<template>
  <AppSectionShell
    title="代理管理"
    subtitle="管理直属代理并查看团队提现申请"
    eyebrow="RESELLER MANAGER"
    icon="badge"
  >
    <div class="manager-page">
      <div v-if="loading" class="card p-10 text-center text-sm text-[var(--ssxz-text-muted)]">
        正在加载代理团队...
      </div>

      <template v-else-if="dashboard">
        <section class="grid gap-4 sm:grid-cols-3" aria-label="团队数据概览">
          <article class="card manager-stat"><span>直属代理</span><strong>{{ dashboard.total_agents }}</strong></article>
          <article class="card manager-stat"><span>团队用户</span><strong>{{ dashboard.total_recruits }}</strong></article>
          <article class="card manager-stat"><span>待处理提现</span><strong>{{ dashboard.pending_withdrawals }}</strong><small>仅管理员可审核</small></article>
        </section>

        <form class="card p-5" @submit.prevent="grantAgent">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-end">
            <div class="min-w-0 flex-1">
              <Input v-model="grantForm.userId" type="number" label="用户 ID" placeholder="输入要授权的用户 ID" :disabled="granting" required />
            </div>
            <div class="min-w-0 flex-[2]">
              <Input v-model="grantForm.notes" label="备注（可选）" placeholder="例如：华东区域代理" :disabled="granting" />
            </div>
            <LiquidButton type="submit" size="default" :disabled="granting">
              <Icon name="userPlus" size="sm" />
              <span>{{ granting ? '正在授权' : '添加直属代理' }}</span>
            </LiquidButton>
          </div>
          <p class="mt-3 text-xs text-[var(--ssxz-text-muted)]">
            只能授予普通代理角色；不能修改其他经理名下的代理。
          </p>
        </form>

        <section class="card overflow-hidden">
          <header class="manager-section-header">
            <div><h2>直属代理</h2><p>共 {{ agents.total }} 人，仅显示当前经理授权的账号。</p></div>
            <div class="flex items-center gap-2">
              <input v-model.trim="search" class="input h-9 w-52" type="search" placeholder="搜索邮箱或昵称" @keyup.enter="loadAgents" />
              <LiquidButton type="button" variant="outline" size="sm" @click="loadAgents"><Icon name="search" size="sm" /><span>搜索</span></LiquidButton>
            </div>
          </header>
          <div class="overflow-x-auto">
            <table class="manager-table min-w-[780px]">
              <thead><tr><th>代理</th><th>推广码</th><th>直属用户</th><th>可用返利</th><th>授权时间</th><th class="text-right">操作</th></tr></thead>
              <tbody>
                <tr v-for="item in agents.items" :key="item.user_id">
                  <td><strong>{{ item.username || `用户 ${item.user_id}` }}</strong><small>{{ item.email }}</small></td>
                  <td>{{ item.aff_code || '-' }}</td>
                  <td>{{ item.recruit_count }}</td>
                  <td>{{ formatCurrency(item.aff_quota) }}</td>
                  <td>{{ formatDateTime(item.granted_at) }}</td>
                  <td class="text-right">
                    <LiquidButton type="button" variant="destructive" size="sm" :disabled="revokingId === item.user_id" @click="revokeAgent(item.user_id)">
                      <Icon name="ban" size="sm" /><span>{{ revokingId === item.user_id ? '处理中' : '撤销' }}</span>
                    </LiquidButton>
                  </td>
                </tr>
                <tr v-if="agents.items.length === 0"><td colspan="6" class="py-10 text-center text-[var(--ssxz-text-muted)]">暂无直属代理</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="card overflow-hidden">
          <header class="manager-section-header">
            <div><h2>团队提现申请</h2><p>经理仅可查看，审批由平台管理员完成。</p></div>
            <div class="flex items-center gap-2">
              <select v-model="withdrawStatus" class="input h-9 w-32" @change="loadWithdrawals">
                <option value="">全部状态</option><option value="pending">待审核</option><option value="approved">已通过</option><option value="rejected">已拒绝</option>
              </select>
              <LiquidButton type="button" variant="outline" size="sm" @click="loadAll"><Icon name="refresh" size="sm" /><span>刷新</span></LiquidButton>
            </div>
          </header>
          <div class="overflow-x-auto">
            <table class="manager-table min-w-[760px]">
              <thead><tr><th>代理</th><th>提交时间</th><th>金额</th><th>方式</th><th>状态</th><th>备注</th></tr></thead>
              <tbody>
                <tr v-for="item in withdrawals.items" :key="item.id">
                  <td>{{ item.user_email || `用户 ${item.user_id}` }}</td>
                  <td>{{ formatDateTime(item.requested_at) }}</td>
                  <td class="font-medium text-[var(--ssxz-text)]">{{ formatCurrency(item.amount) }}</td>
                  <td>{{ methodLabel(item.method) }}</td>
                  <td><span :class="['manager-status', `manager-status--${item.status}`]">{{ statusLabel(item.status) }}</span></td>
                  <td>{{ item.note || '-' }}</td>
                </tr>
                <tr v-if="withdrawals.items.length === 0"><td colspan="6" class="py-10 text-center text-[var(--ssxz-text-muted)]">暂无提现申请</td></tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import Input from '@/components/common/Input.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import resellerAPI, { type AgentSummary, type ManagerDashboard, type WithdrawRequest, type WithdrawStatus } from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import { useAppStore } from '@/stores/app'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const loading = ref(true)
const granting = ref(false)
const revokingId = ref<number | null>(null)
const dashboard = ref<ManagerDashboard | null>(null)
const agents = ref<PaginatedResponse<AgentSummary>>(emptyPage())
const withdrawals = ref<PaginatedResponse<WithdrawRequest>>(emptyPage())
const search = ref('')
const withdrawStatus = ref('')
const grantForm = reactive({ userId: '', notes: '' })

function emptyPage<T>(): PaginatedResponse<T> { return { items: [], total: 0, page: 1, page_size: 20, pages: 0 } }
function methodLabel(method: string): string { return { alipay: '支付宝', wechat: '微信', bank: '银行卡', manual: '其他方式' }[method] || method }
function statusLabel(status: WithdrawStatus): string { return { pending: '待审核', approved: '已通过', rejected: '已拒绝' }[status] }

async function loadAgents(): Promise<void> {
  try { agents.value = await resellerAPI.listManagedAgents(1, 20, search.value) }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '直属代理加载失败')) }
}

async function loadWithdrawals(): Promise<void> {
  try { withdrawals.value = await resellerAPI.listManagedWithdrawals(1, 20, withdrawStatus.value) }
  catch (error) { appStore.showError(extractApiErrorMessage(error, '提现申请加载失败')) }
}

async function loadAll(): Promise<void> {
  loading.value = true
  try {
    const [dashboardData, agentData, withdrawalData] = await Promise.all([
      resellerAPI.getManagerDashboard(),
      resellerAPI.listManagedAgents(1, 20, search.value),
      resellerAPI.listManagedWithdrawals(1, 20, withdrawStatus.value)
    ])
    dashboard.value = dashboardData
    agents.value = agentData
    withdrawals.value = withdrawalData
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '代理团队加载失败'))
  } finally { loading.value = false }
}

async function grantAgent(): Promise<void> {
  const userId = Number(grantForm.userId)
  if (!Number.isInteger(userId) || userId <= 0) { appStore.showWarning('请输入有效的用户 ID'); return }
  granting.value = true
  try {
    await resellerAPI.grantManagedAgent(userId, grantForm.notes.trim())
    grantForm.userId = ''
    grantForm.notes = ''
    appStore.showSuccess('直属代理已添加')
    await loadAll()
  } catch (error) { appStore.showError(extractApiErrorMessage(error, '代理授权失败')) }
  finally { granting.value = false }
}

async function revokeAgent(userId: number): Promise<void> {
  if (!window.confirm('确认撤销该直属代理的角色？')) return
  revokingId.value = userId
  try {
    await resellerAPI.revokeManagedAgent(userId)
    appStore.showSuccess('代理角色已撤销')
    await loadAll()
  } catch (error) { appStore.showError(extractApiErrorMessage(error, '撤销代理失败')) }
  finally { revokingId.value = null }
}

onMounted(() => void loadAll())
</script>

<style scoped>
.manager-page { display: grid; gap: 1.25rem; }
.manager-stat { display: flex; min-height: 7.5rem; flex-direction: column; gap: .5rem; padding: 1.25rem; }
.manager-stat span, .manager-stat small { color: var(--ssxz-text-muted); font-size: .8rem; }
.manager-stat strong { color: var(--ssxz-text); font-size: 1.75rem; }
.manager-stat small { margin-top: auto; }
.manager-section-header { display: flex; min-height: 4.75rem; align-items: center; justify-content: space-between; gap: 1rem; padding: 1rem 1.25rem; border-bottom: 1px solid var(--ssxz-border); }
.manager-section-header h2 { color: var(--ssxz-text); font-size: 1rem; font-weight: 600; }
.manager-section-header p { margin-top: .25rem; color: var(--ssxz-text-muted); font-size: .8rem; }
.manager-table { width: 100%; font-size: .82rem; text-align: left; }
.manager-table th { padding: .75rem 1rem; color: var(--ssxz-text-muted); font-weight: 500; background: var(--ssxz-surface-sunken); }
.manager-table td { padding: .85rem 1rem; color: var(--ssxz-text-secondary); border-top: 1px solid var(--ssxz-border); }
.manager-table td strong, .manager-table td small { display: block; }
.manager-table td small { margin-top: .2rem; color: var(--ssxz-text-muted); }
.manager-status { display: inline-flex; align-items: center; min-height: 1.5rem; padding: 0 .55rem; border: 1px solid var(--ssxz-border); border-radius: 999px; color: var(--ssxz-text-muted); font-size: .72rem; }
.manager-status--pending { border-color: var(--ssxz-warning-border, #854d0e); color: var(--ssxz-warning-text, #f59e0b); }
.manager-status--approved { border-color: var(--ssxz-success-border, #166534); color: var(--ssxz-success-text, #22c55e); }
.manager-status--rejected { border-color: var(--ssxz-danger-border, #991b1b); color: var(--ssxz-danger-text, #ef4444); }
@media (max-width: 767px) { .manager-section-header { align-items: flex-start; flex-direction: column; } }
</style>
