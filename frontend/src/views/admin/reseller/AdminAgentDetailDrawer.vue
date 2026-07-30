<template>
  <BaseDialog
    :show="show"
    title="Agent 详情"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="loading" class="py-12 text-center text-sm text-[var(--ssxz-text-muted)]">
      正在加载 Agent 详情
    </div>
    <div v-else-if="agent" class="space-y-6">
      <header class="detail-header">
        <div class="min-w-0">
          <h4 class="truncate text-base font-semibold text-[var(--ssxz-text)]">
            {{ agent.username || `用户 ${agent.user_id}` }}
          </h4>
          <p class="truncate text-sm text-[var(--ssxz-text-muted)]">{{ agent.email }}</p>
        </div>
        <span :class="['status-badge', `status-badge--${agent.status}`]">
          {{ statusLabel(agent.status) }}
        </span>
      </header>

      <section>
        <h5 class="section-title">合作配置</h5>
        <dl class="detail-grid">
          <div><dt>角色</dt><dd>{{ roleLabel(agent.role) }}</dd></div>
          <div><dt>上级 Manager</dt><dd>{{ agent.manager_email || '未设置' }}</dd></div>
          <div><dt>返利策略</dt><dd>{{ rebateLabel(agent) }}</dd></div>
          <div><dt>邀请码</dt><dd>{{ agent.aff_code || '--' }}</dd></div>
          <div class="md:col-span-2"><dt>备注</dt><dd>{{ agent.notes || '无' }}</dd></div>
        </dl>
      </section>

      <section>
        <h5 class="section-title">业务数据</h5>
        <dl class="detail-grid detail-grid--metrics">
          <div><dt>招募人数</dt><dd>{{ agent.recruit_count }}</dd></div>
          <div><dt>可兑换佣金</dt><dd>${{ formatMoney(agent.commission_balance) }}</dd></div>
          <div><dt>累计佣金</dt><dd>${{ formatMoney(agent.commission_total) }}</dd></div>
          <div><dt>待处理兑换</dt><dd>{{ agent.pending_redemption_count }}</dd></div>
        </dl>
      </section>

      <section>
        <h5 class="section-title">生命周期</h5>
        <dl class="detail-grid">
          <div><dt>授权时间</dt><dd>{{ formatDate(agent.granted_at) }}</dd></div>
          <div><dt>最近更新</dt><dd>{{ formatDate(agent.updated_at) }}</dd></div>
          <div v-if="agent.disabled_at">
            <dt>停用时间</dt><dd>{{ formatDate(agent.disabled_at) }}</dd>
          </div>
          <div v-if="agent.disabled_by_email">
            <dt>停用操作人</dt><dd>{{ agent.disabled_by_email }}</dd>
          </div>
          <div v-if="agent.disabled_reason" class="md:col-span-2">
            <dt>停用原因</dt><dd>{{ agent.disabled_reason }}</dd>
          </div>
          <div v-if="agent.revoked_at">
            <dt>最终撤销时间</dt><dd>{{ formatDate(agent.revoked_at) }}</dd>
          </div>
        </dl>
      </section>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <LiquidButton type="button" variant="outline" size="sm" @click="emit('close')">
          <span>关闭</span>
        </LiquidButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import type { AgentDetail, AgentSummary, ResellerStatus } from '@/api/reseller'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import { formatDateTime } from '@/utils/format'

defineProps<{
  show: boolean
  agent: AgentDetail | null
  loading: boolean
}>()

const emit = defineEmits<{ (event: 'close'): void }>()

function statusLabel(status: ResellerStatus): string {
  if (status === 'disabled') return '已停用'
  if (status === 'revoked') return '已撤销'
  return '启用中'
}

function roleLabel(role: AgentSummary['role']): string {
  return role === 'agent_manager' ? 'Agent Manager' : 'Agent'
}

function rebateLabel(agent: AgentSummary): string {
  if (agent.rebate_mode === 'disabled') return '已关闭（0%）'
  const rate = agent.effective_rebate_rate_percent
  if (agent.rebate_mode === 'global') {
    return rate == null ? '跟随全局' : `跟随全局（${formatRate(rate)}）`
  }
  return rate == null ? '自定义（未设置）' : `自定义（${formatRate(rate)}）`
}

function formatRate(value: number): string {
  return `${value.toFixed(2).replace(/\.?0+$/, '')}%`
}

function formatMoney(value: string): string {
  const amount = Number(value)
  return Number.isFinite(amount) ? amount.toFixed(2) : '0.00'
}

function formatDate(value: string): string {
  return value ? formatDateTime(value) : '--'
}
</script>

<style scoped>
.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding-bottom: 1rem;
}

.section-title {
  margin-bottom: 0.75rem;
  color: var(--ssxz-text);
  font-size: 0.8rem;
  font-weight: 700;
}

.detail-grid {
  display: grid;
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-border);
}

.detail-grid > div {
  min-width: 0;
  background: var(--ssxz-surface);
  padding: 0.85rem;
}

.detail-grid dt {
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
}

.detail-grid dd {
  margin-top: 0.35rem;
  overflow-wrap: anywhere;
  color: var(--ssxz-text);
  font-size: 0.875rem;
  font-weight: 500;
}

.detail-grid--metrics dd {
  font-size: 1.1rem;
}

.status-badge {
  display: inline-flex;
  flex: none;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  font-size: 0.72rem;
  font-weight: 600;
}

.status-badge--active {
  color: var(--ssxz-success);
}

.status-badge--disabled,
.status-badge--revoked {
  color: var(--ssxz-text-muted);
}

@media (min-width: 768px) {
  .detail-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .detail-grid--metrics {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
