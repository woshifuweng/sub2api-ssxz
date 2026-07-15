<template>
  <BaseDialog :show="show" :title="t('admin.users.userApiKeys')" width="wide" @close="handleClose">
    <div v-if="user" class="space-y-4">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
          <span class="text-lg font-medium text-primary-700 dark:text-primary-300">{{ user.email.charAt(0).toUpperCase() }}</span>
        </div>
        <div><p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p><p class="text-sm text-gray-500 dark:text-dark-400">{{ user.username }}</p></div>
      </div>
      <div v-if="loading" class="flex justify-center py-8"><svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg></div>
      <div v-else-if="apiKeys.length === 0" class="py-8 text-center"><p class="text-sm text-gray-500">{{ t('admin.users.noApiKeys') }}</p></div>
      <div v-else class="space-y-3">
        <div
          class="flex flex-wrap items-center gap-2 rounded-xl border border-blue-100 bg-blue-50/70 p-3 text-xs text-blue-900 dark:border-blue-900/40 dark:bg-blue-950/30 dark:text-blue-100"
          data-testid="api-key-readiness-summary"
        >
          <span class="font-semibold">客户 Key 可用性</span>
          <span>可交付 {{ readinessTotals.ready }} 个</span>
          <span>需处理 {{ readinessTotals.blocked }} 个</span>
          <span>需留意 {{ readinessTotals.warning }} 个</span>
        </div>
        <div ref="scrollContainerRef" class="max-h-96 space-y-3 overflow-y-auto" @scroll="closeGroupSelector">
        <div v-for="key in apiKeys" :key="key.id" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
          <div class="flex items-start justify-between">
            <div class="min-w-0 flex-1">
              <div class="mb-1 flex flex-wrap items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ key.name }}</span>
                <span :class="['badge text-xs', key.status === 'active' ? 'badge-success' : 'badge-danger']">{{ key.status }}</span>
                <span
                  :class="[
                    'rounded-full px-2 py-0.5 text-xs font-semibold',
                    keyReadiness(key).level === 'ready'
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                      : keyReadiness(key).level === 'warning'
                        ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'
                        : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
                  ]"
                  data-testid="api-key-readiness-label"
                >
                  {{ keyReadiness(key).label }}
                </span>
              </div>
              <p class="truncate font-mono text-sm text-gray-500">{{ key.key.substring(0, 20) }}...{{ key.key.substring(key.key.length - 8) }}</p>
            </div>
          </div>
          <div
            class="mt-3 rounded-lg border border-gray-100 bg-gray-50 p-3 text-xs dark:border-dark-700 dark:bg-dark-900/50"
            data-testid="api-key-readiness-row"
          >
            <p class="mb-2 font-semibold text-gray-700 dark:text-dark-100">运营判断</p>
            <div class="grid gap-2 sm:grid-cols-2">
              <div>
                <span class="text-gray-400">账户余额：</span>
                <span
                  :class="(user?.balance || 0) > 0 ? 'text-green-600 dark:text-green-300' : 'text-red-600 dark:text-red-300'"
                  :title="formatCurrencyTitle(user?.balance || 0)"
                >
                  {{ (user?.balance || 0) > 0 ? formatCurrency(user?.balance || 0) : '余额不足' }}
                </span>
              </div>
              <div>
                <span class="text-gray-400">最近使用：</span>
                <span class="text-gray-700 dark:text-dark-100">{{ key.last_used_at ? formatDateTime(key.last_used_at) : '暂无记录' }}</span>
              </div>
              <div>
                <span class="text-gray-400">Key 额度：</span>
                <span class="text-gray-700 dark:text-dark-100" :title="formatQuotaTitle(key)">{{ formatQuotaLine(key) }}</span>
              </div>
              <div>
                <span class="text-gray-400">限速：</span>
                <span class="text-gray-700 dark:text-dark-100">{{ formatRateLimitLine(key) }}</span>
              </div>
              <div>
                <span class="text-gray-400">模型限制：</span>
                <span class="text-gray-700 dark:text-dark-100">{{ key.allowed_models?.length ? `${key.allowed_models.length} 个模型` : '不额外限制' }}</span>
              </div>
              <div>
                <span class="text-gray-400">IP 限制：</span>
                <span class="text-gray-700 dark:text-dark-100">{{ formatIPRestrictionLine(key) }}</span>
              </div>
            </div>
            <ul
              v-if="keyReadiness(key).notes.length > 0"
              class="mt-2 space-y-1 text-gray-600 dark:text-dark-200"
              data-testid="api-key-readiness-notes"
            >
              <li v-for="note in keyReadiness(key).notes" :key="note">• {{ note }}</li>
            </ul>
          </div>
          <div class="mt-3 flex flex-wrap gap-4 text-xs text-gray-500">
            <div class="flex items-center gap-1">
              <span>{{ t('admin.users.group') }}:</span>
              <button
                :ref="(el) => setGroupButtonRef(key.id, el)"
                @click="openGroupSelector(key)"
                class="-mx-1 -my-0.5 flex cursor-pointer items-center gap-1 rounded-md px-1 py-0.5 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700"
                :disabled="updatingKeyIds.has(key.id)"
              >
                <GroupBadge
                  v-if="key.group_id && key.group"
                  :name="key.group.name"
                  :platform="key.group.platform"
                  :subscription-type="key.group.subscription_type"
                  :rate-multiplier="key.group.rate_multiplier"
                />
                <span v-else class="text-gray-400 italic">{{ t('admin.users.none') }}</span>
                <svg v-if="updatingKeyIds.has(key.id)" class="h-3 w-3 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                <svg v-else class="h-3 w-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9" /></svg>
              </button>
            </div>
            <div class="flex items-center gap-1"><span>{{ t('admin.users.columns.created') }}: {{ formatDateTime(key.created_at) }}</span></div>
          </div>
        </div>
        </div>
      </div>
    </div>
  </BaseDialog>

  <!-- Group Selector Dropdown -->
  <Teleport to="body">
    <div
      v-if="groupSelectorKeyId !== null && dropdownPosition"
      ref="dropdownRef"
      class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-64 overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 duration-200 dark:bg-dark-800 dark:ring-white/10"
      :style="{ top: dropdownPosition.top + 'px', left: dropdownPosition.left + 'px' }"
    >
      <div class="max-h-64 overflow-y-auto p-1.5">
        <!-- Unbind option -->
        <button
          @click="changeGroup(selectedKeyForGroup!, null)"
          :class="[
            'flex w-full items-center rounded-lg px-3 py-2 text-sm transition-colors',
            !selectedKeyForGroup?.group_id
              ? 'bg-primary-50 dark:bg-primary-900/20'
              : 'hover:bg-gray-100 dark:hover:bg-dark-700'
          ]"
        >
          <span class="text-gray-500 italic">{{ t('admin.users.none') }}</span>
          <svg
            v-if="!selectedKeyForGroup?.group_id"
            class="ml-auto h-4 w-4 shrink-0 text-primary-600 dark:text-primary-400"
            fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"
          ><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
        </button>
        <!-- Group options -->
        <button
          v-for="group in allGroups"
          :key="group.id"
          @click="changeGroup(selectedKeyForGroup!, group.id)"
          :class="[
            'flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm transition-colors',
            selectedKeyForGroup?.group_id === group.id
              ? 'bg-primary-50 dark:bg-primary-900/20'
              : 'hover:bg-gray-100 dark:hover:bg-dark-700'
          ]"
        >
          <GroupOptionItem
            :name="group.name"
            :platform="group.platform"
            :subscription-type="group.subscription_type"
            :rate-multiplier="group.rate_multiplier"
            :description="group.description"
            :selected="selectedKeyForGroup?.group_id === group.id"
          />
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { formatCurrency, formatCurrencyTitle, formatDateTime } from '@/utils/format'
import type { AdminUser, AdminGroup, ApiKey } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits(['close'])
const { t } = useI18n()
const appStore = useAppStore()

const apiKeys = ref<ApiKey[]>([])
const allGroups = ref<AdminGroup[]>([])
const loading = ref(false)
const updatingKeyIds = ref(new Set<number>())
const groupSelectorKeyId = ref<number | null>(null)
const dropdownPosition = ref<{ top: number; left: number } | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const scrollContainerRef = ref<HTMLElement | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())

type ReadinessLevel = 'ready' | 'warning' | 'blocked'

interface KeyReadiness {
  level: ReadinessLevel
  label: string
  notes: string[]
}

const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const readinessTotals = computed(() => {
  return apiKeys.value.reduce(
    (acc, key) => {
      const level = keyReadiness(key).level
      if (level === 'ready') acc.ready += 1
      if (level === 'warning') acc.warning += 1
      if (level === 'blocked') acc.blocked += 1
      return acc
    },
    { ready: 0, warning: 0, blocked: 0 }
  )
})

const hasGroupBinding = (key: ApiKey) => {
  return Boolean(
    key.group_id ||
    (key.group_ids && key.group_ids.length > 0) ||
    (key.groups && key.groups.length > 0)
  )
}

const isExpired = (value: string | null) => {
  return Boolean(value && new Date(value).getTime() <= Date.now())
}

const isExpiringSoon = (value: string | null) => {
  if (!value) return false
  const expiresAt = new Date(value).getTime()
  const sevenDays = 7 * 24 * 60 * 60 * 1000
  return expiresAt > Date.now() && expiresAt - Date.now() <= sevenDays
}

const isQuotaExhausted = (key: ApiKey) => key.quota > 0 && key.quota_used >= key.quota
const isQuotaNearLimit = (key: ApiKey) => key.quota > 0 && key.quota_used >= key.quota * 0.8 && !isQuotaExhausted(key)

const rateLimitWindows = (key: ApiKey) => [
  { label: '5小时', limit: key.rate_limit_5h, usage: key.usage_5h },
  { label: '1天', limit: key.rate_limit_1d, usage: key.usage_1d },
  { label: '7天', limit: key.rate_limit_7d, usage: key.usage_7d }
]

const exhaustedRateLimitWindows = (key: ApiKey) => {
  return rateLimitWindows(key).filter((window) => window.limit > 0 && window.usage >= window.limit)
}

const nearRateLimitWindows = (key: ApiKey) => {
  return rateLimitWindows(key).filter((window) => window.limit > 0 && window.usage >= window.limit * 0.8 && window.usage < window.limit)
}

const keyReadiness = (key: ApiKey): KeyReadiness => {
  const blockers: string[] = []
  const warnings: string[] = []

  if ((props.user?.balance || 0) <= 0) blockers.push('账户余额不足，客户调用会被拒绝')
  if (key.status !== 'active') blockers.push(`Key 状态为 ${key.status}`)
  if (!hasGroupBinding(key)) blockers.push('未绑定分组，先确认客户应使用哪个模型分组')
  if (isExpired(key.expires_at)) blockers.push('Key 已过期')
  if (isQuotaExhausted(key)) blockers.push('Key 额度已用完')

  const exhaustedWindows = exhaustedRateLimitWindows(key)
  if (exhaustedWindows.length > 0) {
    blockers.push(`${exhaustedWindows.map((window) => window.label).join('、')} 限额已用完`)
  }

  if (isExpiringSoon(key.expires_at)) warnings.push('Key 即将过期')
  if (isQuotaNearLimit(key)) warnings.push('Key 额度接近上限')

  const nearWindows = nearRateLimitWindows(key)
  if (nearWindows.length > 0) {
    warnings.push(`${nearWindows.map((window) => window.label).join('、')} 限额接近上限`)
  }
  if (key.allowed_models?.length) warnings.push('已限制可用模型，客户需选择匹配模型')
  if ((key.ip_whitelist?.length || 0) > 0 || (key.ip_blacklist?.length || 0) > 0) {
    warnings.push('已配置 IP 限制，客户换网络可能无法调用')
  }
  if (!key.last_used_at) warnings.push('暂无调用记录，交付前建议做一次 smoke test')

  if (blockers.length > 0) {
    return { level: 'blocked', label: '需处理', notes: [...blockers, ...warnings] }
  }
  if (warnings.length > 0) {
    return { level: 'warning', label: '需留意', notes: warnings }
  }
  return { level: 'ready', label: '可交付', notes: ['状态、余额、分组、额度看起来可用'] }
}

const formatQuotaLine = (key: ApiKey) => {
  if (key.quota <= 0) return '不额外限制'
  return `${formatCurrency(key.quota_used || 0)} / ${formatCurrency(key.quota)}`
}

const formatQuotaTitle = (key: ApiKey) => {
  if (key.quota <= 0) return undefined
  return `已用 ${formatCurrencyTitle(key.quota_used || 0)}；额度 ${formatCurrencyTitle(key.quota)}`
}

const formatRateLimitLine = (key: ApiKey) => {
  const active = rateLimitWindows(key).filter((window) => window.limit > 0)
  if (active.length === 0) return '不额外限制'
  return active.map((window) => `${window.label} $${(window.usage || 0).toFixed(2)}/$${window.limit.toFixed(2)}`).join('，')
}

const formatIPRestrictionLine = (key: ApiKey) => {
  const whitelistCount = key.ip_whitelist?.length || 0
  const blacklistCount = key.ip_blacklist?.length || 0
  if (whitelistCount === 0 && blacklistCount === 0) return '未设置'
  return `白名单 ${whitelistCount} / 黑名单 ${blacklistCount}`
}

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

watch(
  () => props.show,
  (v) => {
    if (v && props.user) {
      load()
      loadGroups()
    } else {
      closeGroupSelector()
    }
  },
  { immediate: true }
)

async function load() {
  if (!props.user) return
  loading.value = true
  groupButtonRefs.value.clear()
  try {
    const res = await adminAPI.users.getUserApiKeys(props.user.id)
    apiKeys.value = res.items || []
  } catch (error) {
    console.error('Failed to load API keys:', error)
  } finally {
    loading.value = false
  }
}

async function loadGroups() {
  try {
    const groups = await adminAPI.groups.getAll()
    allGroups.value = groups
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const DROPDOWN_HEIGHT = 272 // max-h-64 = 16rem = 256px + padding
const DROPDOWN_GAP = 4

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    closeGroupSelector()
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const spaceBelow = window.innerHeight - rect.bottom
      const openUpward = spaceBelow < DROPDOWN_HEIGHT && rect.top > spaceBelow
      dropdownPosition.value = {
        top: openUpward ? rect.top - DROPDOWN_HEIGHT - DROPDOWN_GAP : rect.bottom + DROPDOWN_GAP,
        left: rect.left
      }
    }
    groupSelectorKeyId.value = key.id
  }
}

const closeGroupSelector = () => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  closeGroupSelector()
  if (key.group_id === newGroupId || (!key.group_id && newGroupId === null)) return

  updatingKeyIds.value.add(key.id)
  try {
    const result = await adminAPI.apiKeys.updateApiKeyGroup(key.id, newGroupId)
    // Update local data
    const idx = apiKeys.value.findIndex((k) => k.id === key.id)
    if (idx !== -1) {
      apiKeys.value[idx] = result.api_key
    }
    if (result.auto_granted_group_access && result.granted_group_name) {
      appStore.showSuccess(t('admin.users.groupChangedWithGrant', { group: result.granted_group_name }))
    } else {
      appStore.showSuccess(t('admin.users.groupChangedSuccess'))
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.users.groupChangeFailed'))
  } finally {
    updatingKeyIds.value.delete(key.id)
  }
}

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && groupSelectorKeyId.value !== null) {
    event.stopPropagation()
    closeGroupSelector()
  }
}

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (dropdownRef.value && !dropdownRef.value.contains(target)) {
    // Check if the click is on one of the group trigger buttons
    for (const el of groupButtonRefs.value.values()) {
      if (el.contains(target)) return
    }
    closeGroupSelector()
  }
}

const handleClose = () => {
  closeGroupSelector()
  emit('close')
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleKeyDown, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleKeyDown, true)
})
</script>
