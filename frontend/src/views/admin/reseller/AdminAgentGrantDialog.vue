<template>
  <BaseDialog
    :show="show"
    title="添加 Agent"
    width="normal"
    :close-on-escape="!submitting"
    @close="close"
  >
    <form class="space-y-5" @submit.prevent="grantRole">
      <section class="space-y-2">
        <label class="input-label block" for="agent-user-search">搜索用户邮箱</label>
        <div class="flex flex-col gap-2 sm:flex-row">
          <input
            id="agent-user-search"
            v-model.trim="query"
            class="input min-w-0 flex-1"
            type="search"
            autocomplete="off"
            placeholder="输入完整邮箱或邮箱关键词"
            :disabled="searching || submitting"
            @keyup.enter.prevent="searchUsers"
          />
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="searching || submitting || query.length < 3"
            data-testid="search-users"
            @click="searchUsers"
          >
            <Icon name="search" size="sm" />
            <span>{{ searching ? '搜索中' : '搜索用户' }}</span>
          </LiquidButton>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          至少输入 3 个字符。授权前请核对用户邮箱，避免选错账号。
        </p>
      </section>

      <section
        v-if="searchAttempted"
        class="max-h-56 overflow-y-auto rounded-md border border-gray-200 dark:border-dark-700"
        aria-label="用户搜索结果"
      >
        <button
          v-for="user in results"
          :key="user.id"
          type="button"
          class="user-result"
          :class="{ 'user-result--selected': selectedUserId === user.id }"
          :aria-pressed="selectedUserId === user.id"
          :disabled="submitting"
          :data-testid="`user-result-${user.id}`"
          @click="selectedUserId = user.id"
        >
          <span class="min-w-0">
            <strong>{{ user.username || `用户 ${user.id}` }}</strong>
            <small>{{ user.email }}</small>
          </span>
          <span class="user-result__status">可授权</span>
        </button>
        <div
          v-if="!searching && results.length === 0"
          class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400"
        >
          没有找到可授权的启用用户
        </div>
      </section>

      <section class="grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="input-label mb-1.5 block">角色</span>
          <select v-model="role" class="input w-full" :disabled="submitting">
            <option value="agent">Agent</option>
            <option value="agent_manager">Agent Manager</option>
          </select>
        </label>
        <label class="block">
          <span class="input-label mb-1.5 block">备注（可选）</span>
          <input
            v-model.trim="notes"
            class="input w-full"
            type="text"
            maxlength="200"
            placeholder="例如：渠道来源或负责人"
            :disabled="submitting"
          />
        </label>
      </section>
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
          :disabled="submitting || !selectedUser"
          data-testid="grant-role"
          @click="grantRole"
        >
          <Icon name="userPlus" size="sm" />
          <span>{{ submitting ? '正在授权' : '确认授权' }}</span>
        </LiquidButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import * as adminUsersAPI from '@/api/admin/users'
import resellerAPI, { type ResellerRole } from '@/api/reseller'
import type { AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface Props {
  show: boolean
}

interface Emits {
  (event: 'close'): void
  (event: 'granted', user: AdminUser): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const appStore = useAppStore()

const query = ref('')
const results = ref<AdminUser[]>([])
const selectedUserId = ref<number | null>(null)
const role = ref<ResellerRole>('agent')
const notes = ref('')
const searching = ref(false)
const submitting = ref(false)
const searchAttempted = ref(false)

const selectedUser = computed(
  () => results.value.find((user) => user.id === selectedUserId.value) ?? null
)

watch(
  () => props.show,
  (show) => {
    if (show) reset()
  }
)

function reset(): void {
  query.value = ''
  results.value = []
  selectedUserId.value = null
  role.value = 'agent'
  notes.value = ''
  searching.value = false
  submitting.value = false
  searchAttempted.value = false
}

function close(): void {
  if (submitting.value) return
  emit('close')
}

async function searchUsers(): Promise<void> {
  const keyword = query.value.trim()
  if (keyword.length < 3) {
    appStore.showWarning('请至少输入 3 个字符')
    return
  }

  searching.value = true
  searchAttempted.value = true
  selectedUserId.value = null
  try {
    const response = await adminUsersAPI.list(1, 10, {
      search: keyword,
      status: 'active'
    })
    results.value = response.items
  } catch (error) {
    results.value = []
    appStore.showError(extractApiErrorMessage(error, '用户搜索失败'))
  } finally {
    searching.value = false
  }
}

async function grantRole(): Promise<void> {
  const user = selectedUser.value
  if (!user || submitting.value) return

  submitting.value = true
  try {
    await resellerAPI.grantAdminRole(user.id, {
      role: role.value,
      notes: notes.value || undefined
    })
    const roleName = role.value === 'agent_manager' ? 'Agent Manager' : 'Agent'
    appStore.showSuccess(`${user.email} 已授权为 ${roleName}`)
    emit('granted', user)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, 'Agent 授权失败'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.user-result {
  display: flex;
  width: 100%;
  min-height: 3.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid light-dark(#e5e7eb, #374151);
  padding: 0.75rem 1rem;
  text-align: left;
  transition:
    background-color 150ms ease,
    color 150ms ease;
}

.user-result:last-child {
  border-bottom: 0;
}

.user-result:hover,
.user-result--selected {
  background: light-dark(#ffffff, #1f2937);
}

.user-result--selected {
  box-shadow: inset 3px 0 0 light-dark(#111827, #f9fafb);
}

.user-result strong,
.user-result small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-result strong {
  color: light-dark(#111827, #f9fafb);
  font-size: 0.85rem;
}

.user-result small,
.user-result__status {
  color: light-dark(#6b7280, #9ca3af);
  font-size: 0.72rem;
}

.user-result__status {
  flex: none;
}
</style>
