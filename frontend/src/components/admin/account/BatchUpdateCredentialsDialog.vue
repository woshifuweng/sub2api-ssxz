<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.batchCredentials.title')"
    width="normal"
    @close="handleClose"
  >
    <form id="batch-update-credentials-form" class="space-y-5" @submit.prevent="openConfirmation">
      <div class="rounded-lg bg-primary-50 px-4 py-3 text-sm text-primary-900 dark:bg-primary-900/20 dark:text-primary-100">
        {{ t('admin.accounts.batchCredentials.selectedCount', { count: accountIds.length }) }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.batchCredentials.field') }}</label>
        <Select v-model="field" :options="fieldOptions" />
        <p class="input-hint">{{ t('admin.accounts.batchCredentials.fieldHint') }}</p>
      </div>

      <div v-if="field === 'intercept_warmup_requests'">
        <label class="input-label">{{ t('admin.accounts.batchCredentials.value') }}</label>
        <Select v-model="booleanValue" :options="booleanOptions" />
      </div>

      <div v-else class="space-y-3">
        <div>
          <label class="input-label">{{ t('admin.accounts.batchCredentials.value') }}</label>
          <input
            v-model="stringValue"
            data-testid="batch-credential-value"
            type="password"
            autocomplete="new-password"
            class="input"
            :disabled="clearValue"
            :placeholder="t('admin.accounts.batchCredentials.valuePlaceholder')"
          />
          <p class="input-hint">{{ t('admin.accounts.batchCredentials.valuePrivacyHint') }}</p>
        </div>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            v-model="clearValue"
            data-testid="batch-credential-clear"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span>{{ t('admin.accounts.batchCredentials.clearValue') }}</span>
        </label>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="batch-update-credentials-form"
          data-testid="batch-credentials-continue"
          class="btn btn-primary"
          :disabled="!canContinue || submitting"
        >
          {{ t('admin.accounts.batchCredentials.continue') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <ConfirmDialog
    :show="showConfirmation"
    :title="t('admin.accounts.batchCredentials.confirmTitle')"
    :message="confirmationMessage"
    :confirm-text="t('admin.accounts.batchCredentials.confirmAction')"
    :cancel-text="t('common.cancel')"
    @confirm="submit"
    @cancel="showConfirmation = false"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { BatchCredentialField } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'

interface Props {
  show: boolean
  accountIds: number[]
}

interface Emits {
  (event: 'close'): void
  (event: 'updated'): void
  (event: 'completed'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { t } = useI18n()
const appStore = useAppStore()

const field = ref<BatchCredentialField>('account_uuid')
const stringValue = ref('')
const booleanValue = ref(true)
const clearValue = ref(false)
const showConfirmation = ref(false)
const submitting = ref(false)

const fieldOptions = computed(() => [
  { value: 'account_uuid', label: t('admin.accounts.batchCredentials.fields.accountUuid') },
  { value: 'org_uuid', label: t('admin.accounts.batchCredentials.fields.orgUuid') },
  {
    value: 'intercept_warmup_requests',
    label: t('admin.accounts.batchCredentials.fields.interceptWarmup')
  }
])

const booleanOptions = computed(() => [
  { value: true, label: t('common.enabled') },
  { value: false, label: t('common.disabled') }
])

const fieldLabel = computed(
  () => fieldOptions.value.find((option) => option.value === field.value)?.label ?? ''
)

const canContinue = computed(() => {
  if (props.accountIds.length === 0) return false
  if (field.value === 'intercept_warmup_requests') return true
  return clearValue.value || stringValue.value.trim().length > 0
})

const confirmationMessage = computed(() =>
  t('admin.accounts.batchCredentials.confirmMessage', {
    count: props.accountIds.length,
    field: fieldLabel.value
  })
)

const resetValue = () => {
  stringValue.value = ''
  booleanValue.value = true
  clearValue.value = false
  showConfirmation.value = false
}

const reset = () => {
  field.value = 'account_uuid'
  resetValue()
}

const handleClose = () => {
  if (submitting.value) return
  reset()
  emit('close')
}

const openConfirmation = () => {
  if (!canContinue.value) return
  showConfirmation.value = true
}

const submit = async () => {
  if (!canContinue.value || submitting.value) return
  showConfirmation.value = false
  submitting.value = true

  const value = field.value === 'intercept_warmup_requests'
    ? booleanValue.value
    : clearValue.value
      ? null
      : stringValue.value.trim()

  try {
    const result = await adminAPI.accounts.batchUpdateCredentials({
      account_ids: [...props.accountIds],
      field: field.value,
      value
    })

    stringValue.value = ''
    emit('updated')
    if (result.failed > 0) {
      appStore.showError(
        t('admin.accounts.batchCredentials.partialSuccess', {
          success: result.success,
          failed: result.failed
        })
      )
      return
    }

    appStore.showSuccess(
      t('admin.accounts.batchCredentials.success', { count: result.success })
    )
    reset()
    emit('completed')
  } catch (error) {
    appStore.showError(t('admin.accounts.batchCredentials.failed'))
    console.error('Failed to batch update account credential fields:', error)
  } finally {
    submitting.value = false
  }
}

watch(field, resetValue)
watch(
  () => props.show,
  (show) => {
    if (!show && !submitting.value) reset()
  }
)
</script>
