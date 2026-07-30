<template>
  <BaseDialog
    :show="show"
    title="编辑 Agent"
    width="wide"
    :close-on-escape="!submitting"
    @close="close"
  >
    <form class="space-y-6" @submit.prevent="submit">
      <div class="agent-context">
        <div class="min-w-0">
          <p class="truncate font-medium text-[var(--ssxz-text)]">
            {{ agent?.username || `用户 ${agent?.user_id ?? ''}` }}
          </p>
          <p class="truncate text-sm text-[var(--ssxz-text-muted)]">
            {{ agent?.email || '--' }}
          </p>
        </div>
        <span :class="['status-badge', `status-badge--${agent?.status || 'active'}`]">
          {{ statusLabel(agent?.status) }}
        </span>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <label class="block">
          <span class="input-label mb-1.5 block">角色</span>
          <select v-model="role" class="input w-full" :disabled="submitting">
            <option value="agent">Agent</option>
            <option value="agent_manager">Agent Manager</option>
          </select>
          <span class="field-hint">降级 Manager 前必须先转移其直属 Agent。</span>
        </label>

        <label class="block">
          <span class="input-label mb-1.5 block">上级 Manager</span>
          <select v-model="managerId" class="input w-full" :disabled="submitting">
            <option value="">不设置上级</option>
            <option
              v-for="manager in availableManagers"
              :key="manager.user_id"
              :value="String(manager.user_id)"
            >
              {{ manager.username || manager.email }}（{{ manager.email }}）
            </option>
          </select>
          <span class="field-hint">只能选择启用中的 Agent Manager。</span>
        </label>
      </div>

      <label class="block">
        <span class="input-label mb-1.5 block">备注</span>
        <textarea
          v-model.trim="notes"
          class="input min-h-24 w-full resize-y"
          maxlength="500"
          placeholder="记录渠道来源、负责人或合作约定"
          :disabled="submitting"
        />
      </label>

      <fieldset class="space-y-3">
        <legend class="input-label">用户返利策略</legend>
        <div class="grid gap-3 md:grid-cols-3">
          <label
            v-for="option in rebateOptions"
            :key="option.value"
            :class="['policy-option', { 'policy-option--selected': rebateMode === option.value }]"
          >
            <input
              v-model="rebateMode"
              class="sr-only"
              type="radio"
              name="rebate-mode"
              :value="option.value"
              :disabled="submitting"
            />
            <strong>{{ option.label }}</strong>
            <span>{{ option.description }}</span>
          </label>
        </div>
        <label v-if="rebateMode === 'custom'" class="block max-w-xs">
          <span class="input-label mb-1.5 block">自定义返利比例（%）</span>
          <input
            v-model.number="customRate"
            class="input w-full"
            type="number"
            min="0"
            max="100"
            step="0.01"
            :disabled="submitting"
          />
        </label>
      </fieldset>

      <label class="block">
        <span class="input-label mb-1.5 block">变更原因（可选）</span>
        <input
          v-model.trim="reason"
          class="input w-full"
          type="text"
          maxlength="300"
          placeholder="用于本次敏感配置变更的审计说明"
          :disabled="submitting"
        />
      </label>
    </form>

    <template #footer>
      <div class="flex flex-wrap justify-end gap-2">
        <LiquidButton
          type="button"
          variant="outline"
          size="sm"
          :disabled="submitting"
          @click="close"
        >
          <span>取消</span>
        </LiquidButton>
        <LiquidButton
          type="button"
          size="sm"
          :disabled="submitting || !agent"
          data-testid="save-agent"
          @click="submit"
        >
          <span>{{ submitting ? '正在保存' : '保存变更' }}</span>
        </LiquidButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import resellerAPI, {
  type AgentDetail,
  type AgentSummary,
  type RebateMode,
  type ResellerRole,
  type ResellerStatus
} from '@/api/reseller'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface Props {
  show: boolean
  agent: AgentDetail | null
  managers: AgentSummary[]
}

interface Emits {
  (event: 'close'): void
  (event: 'saved', agent: AgentDetail): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const appStore = useAppStore()

const role = ref<ResellerRole>('agent')
const managerId = ref('')
const notes = ref('')
const rebateMode = ref<RebateMode>('global')
const customRate = ref<number>(0)
const reason = ref('')
const submitting = ref(false)

const rebateOptions: Array<{ value: RebateMode; label: string; description: string }> = [
  { value: 'global', label: '跟随全局', description: '使用系统当前统一返利比例' },
  { value: 'disabled', label: '关闭返利', description: '该 Agent 招募用户不产生返利' },
  { value: 'custom', label: '自定义', description: '为该 Agent 单独设置返利比例' }
]

const availableManagers = computed(() =>
  props.managers.filter(
    (manager) => manager.user_id !== props.agent?.user_id && manager.status === 'active'
  )
)

watch(
  () => [props.show, props.agent] as const,
  ([show, agent]) => {
    if (!show || !agent) return
    role.value = agent.role
    managerId.value = agent.manager_id ? String(agent.manager_id) : ''
    notes.value = agent.notes || ''
    rebateMode.value = agent.rebate_mode || 'global'
    customRate.value = agent.effective_rebate_rate_percent ?? 0
    reason.value = ''
  },
  { immediate: true }
)

function statusLabel(status?: ResellerStatus): string {
  if (status === 'disabled') return '已停用'
  if (status === 'revoked') return '已撤销'
  return '启用中'
}

function close(): void {
  if (submitting.value) return
  emit('close')
}

async function submit(): Promise<void> {
  const agent = props.agent
  if (!agent || submitting.value) return
  if (
    rebateMode.value === 'custom' &&
    (!Number.isFinite(customRate.value) || customRate.value < 0 || customRate.value > 100)
  ) {
    appStore.showWarning('自定义返利比例必须在 0% 到 100% 之间')
    return
  }

  submitting.value = true
  try {
    const updated = await resellerAPI.updateAdminAgent(agent.user_id, {
      role: role.value,
      manager_id: managerId.value ? Number(managerId.value) : null,
      notes: notes.value,
      rebate_policy: {
        mode: rebateMode.value,
        ...(rebateMode.value === 'custom' ? { rate_percent: customRate.value } : {})
      },
      reason: reason.value || undefined
    })
    appStore.showSuccess('Agent 配置已更新')
    emit('saved', updated)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 配置保存失败'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.agent-context {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding-bottom: 1rem;
}

.field-hint {
  display: block;
  margin-top: 0.4rem;
  color: var(--ssxz-text-muted);
  font-size: 0.75rem;
}

.policy-option {
  display: flex;
  min-height: 5rem;
  cursor: pointer;
  flex-direction: column;
  gap: 0.35rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  padding: 0.85rem;
  transition:
    border-color 150ms ease,
    background-color 150ms ease;
}

.policy-option:hover,
.policy-option--selected {
  border-color: var(--ssxz-text-subtle);
  background: var(--ssxz-surface-raised);
}

.policy-option strong {
  color: var(--ssxz-text);
  font-size: 0.875rem;
}

.policy-option span {
  color: var(--ssxz-text-muted);
  font-size: 0.75rem;
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
</style>
