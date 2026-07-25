<template>
  <BaseDialog
    :show="show"
    :title="t('admin.subscriptions.bulkAssign.title')"
    width="wide"
    @close="handleClose"
  >
    <form id="bulk-assign-subscriptions-form" class="space-y-5" @submit.prevent="openConfirmation">
      <div>
        <label class="input-label">{{ t('admin.subscriptions.bulkAssign.users') }}</label>
        <div class="relative">
          <Icon
            name="search"
            size="sm"
            class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
          />
          <input
            v-model="searchKeyword"
            data-testid="bulk-user-search"
            type="search"
            class="input pl-9"
            :placeholder="t('admin.subscriptions.bulkAssign.searchPlaceholder')"
            @input="scheduleSearch"
          />
        </div>

        <div
          v-if="searchKeyword.trim()"
          class="mt-2 max-h-48 overflow-y-auto rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800"
        >
          <div v-if="searchLoading" class="px-4 py-3 text-sm text-gray-500 dark:text-dark-300">
            {{ t('common.loading') }}
          </div>
          <div
            v-else-if="searchResults.length === 0"
            class="px-4 py-3 text-sm text-gray-500 dark:text-dark-300"
          >
            {{ t('common.noOptionsFound') }}
          </div>
          <button
            v-for="user in searchResults"
            :key="user.id"
            :data-testid="`bulk-user-${user.id}`"
            type="button"
            class="flex w-full items-center gap-3 border-b border-gray-100 px-4 py-3 text-left last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700"
            @click="toggleUser(user)"
          >
            <span
              :class="[
                'flex h-5 w-5 flex-none items-center justify-center rounded border',
                isSelected(user.id)
                  ? 'border-primary-500 bg-primary-500 text-white'
                  : 'border-gray-300 text-transparent dark:border-dark-500'
              ]"
            >
              <Icon name="check" size="xs" :stroke-width="2.5" />
            </span>
            <span class="min-w-0 flex-1 truncate text-sm font-medium text-gray-900 dark:text-white">
              {{ user.email }}
            </span>
            <span class="text-xs text-gray-400 dark:text-dark-400">#{{ user.id }}</span>
          </button>
        </div>

        <div class="mt-3 rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
          <div class="flex items-center justify-between gap-3">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ t('admin.subscriptions.bulkAssign.selectedCount', { count: selectedUsers.length }) }}
            </span>
            <button
              v-if="selectedUsers.length"
              type="button"
              class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
              @click="selectedUsers = []"
            >
              {{ t('admin.subscriptions.bulkAssign.clearSelection') }}
            </button>
          </div>
          <div v-if="selectedUsers.length" class="mt-2 flex max-h-24 flex-wrap gap-2 overflow-y-auto">
            <button
              v-for="user in selectedUsers"
              :key="user.id"
              type="button"
              class="inline-flex max-w-full items-center gap-1.5 rounded-md bg-white px-2.5 py-1.5 text-xs text-gray-700 shadow-sm ring-1 ring-gray-200 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-600"
              :title="t('admin.subscriptions.bulkAssign.removeUser')"
              @click="removeUser(user.id)"
            >
              <span class="truncate">{{ user.email }}</span>
              <Icon name="x" size="xs" />
            </button>
          </div>
          <p v-else class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.subscriptions.bulkAssign.noUsersSelected') }}
          </p>
        </div>
      </div>

      <div class="grid gap-5 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.group') }}</label>
          <Select
            v-model="groupId"
            :options="groupOptions"
            :placeholder="t('admin.subscriptions.selectGroup')"
            searchable
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
              />
              <span v-else class="text-gray-400">{{ t('admin.subscriptions.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
          <p class="input-hint">{{ t('admin.subscriptions.groupHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.validityDays') }}</label>
          <input v-model.number="validityDays" type="number" min="1" class="input" />
          <p class="input-hint">{{ t('admin.subscriptions.validityHint') }}</p>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="bulk-assign-subscriptions-form"
          data-testid="bulk-assign-continue"
          class="btn btn-primary"
          :disabled="!canContinue || submitting"
        >
          {{ t('admin.subscriptions.bulkAssign.continue') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <ConfirmDialog
    :show="showConfirmation"
    :title="t('admin.subscriptions.bulkAssign.confirmTitle')"
    :message="confirmationMessage"
    :confirm-text="t('admin.subscriptions.bulkAssign.confirmAction')"
    :cancel-text="t('common.cancel')"
    @confirm="submit"
    @cancel="showConfirmation = false"
  />
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { SimpleUser } from '@/api/admin/usage'
import type {
  BulkAssignSubscriptionResult,
  Group,
  GroupPlatform,
  SubscriptionType
} from '@/types'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  show: boolean
  groups: Group[]
}

interface Emits {
  (event: 'close'): void
  (event: 'assigned', result: BulkAssignSubscriptionResult): void
}

interface GroupOption {
  value: number
  label: string
  description: string | null
  platform: GroupPlatform
  subscriptionType: SubscriptionType
  rate: number
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { t } = useI18n()
const appStore = useAppStore()

const searchKeyword = ref('')
const searchResults = ref<SimpleUser[]>([])
const searchLoading = ref(false)
const selectedUsers = ref<SimpleUser[]>([])
const groupId = ref<number | null>(null)
const validityDays = ref(30)
const showConfirmation = ref(false)
const submitting = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null
let searchSequence = 0

const groupOptions = computed(() =>
  props.groups
    .filter((group) => group.status === 'active' && group.subscription_type === 'subscription')
    .map((group) => ({
      value: group.id,
      label: group.name,
      description: group.description,
      platform: group.platform,
      subscriptionType: group.subscription_type,
      rate: group.rate_multiplier
    }))
)

const canContinue = computed(
  () => selectedUsers.value.length > 0 && groupId.value !== null && validityDays.value >= 1
)

const selectedGroupName = computed(
  () => groupOptions.value.find((option) => option.value === groupId.value)?.label ?? ''
)

const confirmationMessage = computed(() =>
  t('admin.subscriptions.bulkAssign.confirmMessage', {
    count: selectedUsers.value.length,
    group: selectedGroupName.value,
    days: validityDays.value
  })
)

const isSelected = (userId: number) => selectedUsers.value.some((user) => user.id === userId)

const toggleUser = (user: SimpleUser) => {
  if (isSelected(user.id)) {
    removeUser(user.id)
    return
  }
  selectedUsers.value = [...selectedUsers.value, user]
}

const removeUser = (userId: number) => {
  selectedUsers.value = selectedUsers.value.filter((user) => user.id !== userId)
}

const scheduleSearch = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(searchUsers, 300)
}

const searchUsers = async () => {
  const keyword = searchKeyword.value.trim()
  const sequence = ++searchSequence
  if (!keyword) {
    searchResults.value = []
    return
  }

  searchLoading.value = true
  try {
    const results = await adminAPI.usage.searchUsers(keyword)
    if (sequence === searchSequence) searchResults.value = results
  } catch (error) {
    if (sequence === searchSequence) searchResults.value = []
    console.error('Failed to search users for bulk subscription assignment:', error)
  } finally {
    if (sequence === searchSequence) searchLoading.value = false
  }
}

const openConfirmation = () => {
  if (!canContinue.value) return
  showConfirmation.value = true
}

const reset = () => {
  searchSequence += 1
  searchKeyword.value = ''
  searchResults.value = []
  searchLoading.value = false
  selectedUsers.value = []
  groupId.value = null
  validityDays.value = 30
  showConfirmation.value = false
}

const handleClose = () => {
  if (submitting.value) return
  reset()
  emit('close')
}

const submit = async () => {
  if (!canContinue.value || submitting.value || groupId.value === null) return
  showConfirmation.value = false
  submitting.value = true
  try {
    const result = await adminAPI.subscriptions.bulkAssign({
      user_ids: selectedUsers.value.map((user) => user.id),
      group_id: groupId.value,
      validity_days: validityDays.value
    })
    emit('assigned', result)

    if (result.failed_count > 0) {
      appStore.showError(
        t('admin.subscriptions.bulkAssign.partialSuccess', {
          success: result.success_count,
          failed: result.failed_count
        })
      )
      selectedUsers.value = selectedUsers.value.filter(
        (user) => result.statuses[String(user.id)] === 'failed'
      )
      return
    }

    appStore.showSuccess(
      t('admin.subscriptions.bulkAssign.success', {
        created: result.created_count,
        reused: result.reused_count
      })
    )
    reset()
    emit('close')
  } catch (error: any) {
    appStore.showError(
      error?.response?.data?.detail || t('admin.subscriptions.bulkAssign.failed')
    )
  } finally {
    submitting.value = false
  }
}

watch(
  () => props.show,
  (show) => {
    if (!show && !submitting.value) reset()
  }
)

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>
