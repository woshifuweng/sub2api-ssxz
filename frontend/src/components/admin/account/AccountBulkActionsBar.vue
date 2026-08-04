<template>
  <div
    v-if="selectedIds.length > 0"
    class="mb-4 flex flex-col gap-3 rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20 xl:flex-row xl:items-center xl:justify-between"
  >
    <div class="flex flex-wrap items-center gap-2">
      <span v-if="allResultsSelected" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selectedAll', { count: selectedIds.length }) }}
      </span>
      <span v-else-if="selectedIds.length > 0" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-else class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkEdit.title') }}
      </span>
      <template v-if="selectedIds.length > 0">
        <button
          @click="$emit('select-page')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
        </button>
      </template>
      <template v-if="!allResultsSelected && totalResults > selectedIds.length">
        <span v-if="selectedIds.length > 0" class="text-gray-300 dark:text-primary-800">•</span>
        <button
          :disabled="selectingAll"
          @click="$emit('select-all-results')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 disabled:cursor-not-allowed disabled:opacity-60 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{
            selectingAll
              ? t('admin.accounts.bulkActions.selectingAll')
              : t('admin.accounts.bulkActions.selectAllResults', { count: totalResults })
          }}
        </button>
      </template>
      <template v-if="selectedIds.length > 0">
        <span class="text-gray-300 dark:text-primary-800">•</span>
        <button
          @click="$emit('clear')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </template>
    </div>
    <div class="flex flex-wrap gap-2">
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('batch-test')">
        {{ t('admin.accounts.bulkActions.batchTest') }}
      </button>
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('batch-credentials')">
        <Icon name="key" size="sm" class="mr-1.5" />
        {{ t('admin.accounts.bulkActions.updateCredentials') }}
      </button>
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('reset-status')">
        {{ t('admin.accounts.bulkActions.resetStatus') }}
      </button>
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('refresh-token')">
        {{ t('admin.accounts.bulkActions.refreshToken') }}
      </button>
      <button type="button" class="btn btn-success btn-sm" @click="emit('toggle-schedulable', true)">
        {{ t('admin.accounts.bulkActions.enableScheduling') }}
      </button>
      <button type="button" class="btn btn-warning btn-sm" @click="emit('toggle-schedulable', false)">
        {{ t('admin.accounts.bulkActions.disableScheduling') }}
      </button>
      <button type="button" class="btn btn-primary btn-sm" @click="emit('edit')">
        {{ t('admin.accounts.bulkActions.edit') }}
      </button>
      <button type="button" class="btn btn-danger btn-sm" @click="emit('delete')">
        {{ t('admin.accounts.bulkActions.delete') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  selectedIds: number[]
  totalResults: number
  selectingAll: boolean
  allResultsSelected: boolean
}>()

defineEmits([
  'delete',
  'edit-selected',
  'edit-filtered',
  'clear',
  'select-page',
  'select-all-results',
  'toggle-schedulable',
  'reset-status',
  'refresh-token',
  'probe-upstream-billing'
])

const { t } = useI18n()
</script>
