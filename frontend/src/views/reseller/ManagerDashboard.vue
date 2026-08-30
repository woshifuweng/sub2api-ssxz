<template>
  <AppSectionShell
    title="管理 Agent"
    subtitle="查看直属 Agent 数据并维护角色"
    eyebrow="RESELLER MANAGER"
    icon="badge"
  >
    <div class="manager-page">
      <div
        v-if="loading"
        class="card p-10 text-center text-sm text-[var(--ssxz-text-muted)]"
      >
        正在加载 Agent 数据...
      </div>

      <template v-else>
        <section v-if="dashboard" class="manager-stats" aria-label="团队数据概览">
          <article class="card manager-stat">
            <span>直属 Agent</span>
            <strong>{{ dashboard.total_agents }}</strong>
          </article>
          <article class="card manager-stat">
            <span>团队招募</span>
            <strong>{{ dashboard.total_recruits }}</strong>
          </article>
          <article class="card manager-stat">
            <span>待审核申请</span>
            <strong>{{ dashboard.pending_withdrawals }}</strong>
          </article>
        </section>

        <form class="card manager-add-form" @submit.prevent="addAgent">
          <div>
            <h2>添加直属 Agent</h2>
            <p>输入用户 ID，为普通用户开通 Agent 角色。</p>
          </div>
          <Input
            v-model="newAgent.userId"
            type="number"
            label="用户 ID"
            placeholder="输入用户 ID"
            :disabled="saving"
            required
          />
          <Input
            v-model="newAgent.notes"
            label="备注（可选）"
            placeholder="例如：渠道来源或负责人"
            :disabled="saving"
          />
          <LiquidButton type="submit" size="default" :disabled="saving">
            <Icon name="userPlus" size="sm" />
            <span>{{ saving ? '正在保存' : '添加 Agent' }}</span>
          </LiquidButton>
        </form>

        <section class="card overflow-hidden">
          <header class="manager-header">
            <div>
              <h2>Agent 列表</h2>
              <p>共 {{ agents.total }} 人，仅展示当前账号管理的 Agent。</p>
            </div>
            <div class="manager-toolbar">
              <input
                v-model.trim="search"
                class="input h-9 w-52"
                type="search"
                placeholder="搜索邮箱或昵称"
                @keyup.enter="searchAgents"
              />
              <LiquidButton type="button" variant="outline" size="sm" @click="searchAgents">
                <Icon name="search" size="sm" />
                <span>搜索</span>
              </LiquidButton>
            </div>
          </header>

          <div v-if="loadError" class="manager-empty">{{ loadError }}</div>
          <div v-else class="overflow-x-auto">
            <table class="manager-table min-w-[840px]">
              <thead>
                <tr>
                  <th>用户</th>
                  <th>当前角色</th>
                  <th>招募数</th>
                  <th>累计额度</th>
                  <th>开通时间</th>
                  <th class="text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in agents.items" :key="item.user_id">
                  <td>
                    <strong>{{ item.username || `用户 ${item.user_id}` }}</strong>
                    <small>{{ maskEmail(item.email) }}</small>
                  </td>
                  <td>{{ item.role === 'agent_manager' ? '管理 Agent' : 'Agent' }}</td>
                  <td>{{ item.recruit_count ?? '--' }}</td>
                  <td>{{ item.commission_total ? `${item.commission_total} 额度` : '--' }}</td>
                  <td>{{ formatRelativeTime(item.granted_at) }}</td>
                  <td class="text-right">
                    <LiquidButton
                      type="button"
                      variant="outline"
                      size="sm"
                      @click="openRoleDialog(item)"
                    >
                      <span>设置角色</span>
                    </LiquidButton>
                  </td>
                </tr>
                <tr v-if="agents.items.length === 0">
                  <td colspan="6" class="py-10 text-center text-[var(--ssxz-text-muted)]">
                    暂无直属 Agent
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <Pagination
            v-if="agents.total > 0"
            :page="agents.page"
            :page-size="agents.page_size"
            :total="agents.total"
            :show-page-size-selector="false"
            @update:page="changePage"
          />
        </section>
      </template>
    </div>

    <BaseDialog
      :show="!!roleTarget"
      title="设置 Agent 角色"
      width="narrow"
      @close="closeRoleDialog"
    >
      <div class="space-y-4">
        <p class="text-sm text-[var(--ssxz-text-secondary)]">
          {{ roleTarget?.email || `用户 ${roleTarget?.user_id ?? ''}` }}
        </p>
        <label class="block">
          <span class="input-label mb-1.5 block">角色</span>
          <select v-model="selectedRole" class="input w-full" :disabled="saving">
            <option value="agent">Agent</option>
            <option value="none">无 Reseller 角色</option>
          </select>
        </label>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <LiquidButton type="button" variant="outline" size="sm" @click="closeRoleDialog">
            <span>取消</span>
          </LiquidButton>
          <LiquidButton
            type="button"
            :variant="selectedRole === 'none' ? 'destructive' : 'default'"
            size="sm"
            :disabled="saving"
            @click="saveRole"
          >
            <span>{{ saving ? '正在保存' : '确认设置' }}</span>
          </LiquidButton>
        </div>
      </template>
    </BaseDialog>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Input from '@/components/common/Input.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Pagination from '@/components/common/Pagination.vue'
import resellerAPI, {
  type AgentSummary,
  type ManagerDashboard,
  type ManagedAgentRole
} from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import { useAppStore } from '@/stores/app'
import { formatRelativeTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const dashboard = ref<ManagerDashboard | null>(null)
const agents = ref<PaginatedResponse<AgentSummary>>(emptyPage())
const search = ref('')
const newAgent = reactive({ userId: '', notes: '' })
const roleTarget = ref<AgentSummary | null>(null)
const selectedRole = ref<'agent' | 'none'>('agent')

function maskEmail(email: string | undefined): string {
  if (!email) return '--'
  const [local, domain] = email.split('@')
  if (!domain) return '***'
  if (local.length <= 2) return `${local[0] || '*'}***@${domain}`
  return `${local.slice(0, 2)}***@${domain}`
}

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
}

async function loadPage(page = agents.value.page || 1): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const [dashboardData, agentData] = await Promise.all([
      resellerAPI.getManagerDashboard(),
      resellerAPI.listManagedAgents(page, agents.value.page_size, search.value)
    ])
    dashboard.value = dashboardData
    agents.value = agentData
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, 'Agent 数据加载失败')
  } finally {
    loading.value = false
  }
}

async function addAgent(): Promise<void> {
  const userId = Number(newAgent.userId)
  if (!Number.isInteger(userId) || userId <= 0) {
    appStore.showWarning('请输入有效的用户 ID')
    return
  }

  saving.value = true
  try {
    await resellerAPI.setManagedAgentRole(userId, 'agent', newAgent.notes.trim())
    newAgent.userId = ''
    newAgent.notes = ''
    appStore.showSuccess('Agent 已添加')
    await loadPage(1)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 添加失败'))
  } finally {
    saving.value = false
  }
}

function openRoleDialog(agent: AgentSummary): void {
  roleTarget.value = agent
  selectedRole.value = 'agent'
}

function closeRoleDialog(): void {
  if (saving.value) return
  roleTarget.value = null
}

async function saveRole(): Promise<void> {
  const target = roleTarget.value
  if (!target) return

  saving.value = true
  const role: ManagedAgentRole = selectedRole.value === 'agent' ? 'agent' : null
  try {
    await resellerAPI.setManagedAgentRole(target.user_id, role)
    appStore.showSuccess(role === 'agent' ? 'Agent 角色已保留' : 'Agent 角色已移除')
    roleTarget.value = null
    await loadPage(agents.value.page)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '角色设置失败'))
  } finally {
    saving.value = false
  }
}

function searchAgents(): void {
  void loadPage(1)
}

function changePage(page: number): void {
  void loadPage(page)
}

onMounted(() => void loadPage(1))
</script>

<style scoped>
.manager-page {
  display: grid;
  gap: 1.25rem;
}

.manager-stats {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.manager-stat {
  display: flex;
  min-height: 7.5rem;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1.25rem;
}

.manager-stat span {
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.manager-stat strong {
  color: var(--ssxz-text);
  font-size: 1.75rem;
}

.manager-add-form {
  display: grid;
  align-items: end;
  gap: 1rem;
  grid-template-columns: minmax(220px, 0.8fr) minmax(180px, 0.7fr) minmax(260px, 1.5fr) auto;
  padding: 1.25rem;
}

.manager-add-form h2,
.manager-header h2 {
  color: var(--ssxz-text);
  font-size: 1rem;
  font-weight: 600;
}

.manager-add-form p,
.manager-header p {
  margin-top: 0.25rem;
  color: var(--ssxz-text-muted);
  font-size: 0.8rem;
}

.manager-header {
  display: flex;
  min-height: 4.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 1rem 1.25rem;
}

.manager-toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.manager-table {
  width: 100%;
  font-size: 0.82rem;
  text-align: left;
}

.manager-table th {
  background: var(--ssxz-surface-sunken);
  padding: 0.75rem 1rem;
  color: var(--ssxz-text-muted);
  font-weight: 500;
}

.manager-table td {
  border-top: 1px solid var(--ssxz-border);
  padding: 0.85rem 1rem;
  color: var(--ssxz-text-secondary);
}

.manager-table td strong,
.manager-table td small {
  display: block;
}

.manager-table td small {
  margin-top: 0.2rem;
  color: var(--ssxz-text-muted);
}

.manager-empty {
  padding: 2.5rem;
  color: var(--ssxz-text-muted);
  text-align: center;
}

@media (max-width: 1023px) {
  .manager-add-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 639px) {
  .manager-stats,
  .manager-add-form {
    grid-template-columns: minmax(0, 1fr);
  }

  .manager-header,
  .manager-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .manager-toolbar input {
    width: 100%;
  }
}
</style>
