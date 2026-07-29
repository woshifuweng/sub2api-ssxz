<template>
  <div class="flex items-center gap-1">
    <LiquidButton
      variant="plain"
      size="sm"
      @click="$emit('run', row)"
      :disabled="running"
      class="h-8 gap-1.5 rounded-md px-2.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
    >
      <Icon name="refresh" size="sm" :class="running ? 'animate-spin' : ''" />
      <span class="text-xs">{{ t('admin.channelMonitor.runNow') }}</span>
    </LiquidButton>
    <LiquidButton
      variant="plain"
      size="sm"
      @click="$emit('edit', row)"
      class="h-8 gap-1.5 rounded-md px-2.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
    >
      <Icon name="edit" size="sm" />
      <span class="text-xs">{{ t('common.edit') }}</span>
    </LiquidButton>
    <LiquidButton
      variant="plain"
      size="sm"
      @click="$emit('delete', row)"
      class="h-8 gap-1.5 rounded-md px-2.5 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
    >
      <Icon name="trash" size="sm" />
      <span class="text-xs">{{ t('common.delete') }}</span>
    </LiquidButton>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ChannelMonitor } from '@/api/admin/channelMonitor'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  row: ChannelMonitor
  running: boolean
}>()

defineEmits<{
  (e: 'run', row: ChannelMonitor): void
  (e: 'edit', row: ChannelMonitor): void
  (e: 'delete', row: ChannelMonitor): void
}>()

const { t } = useI18n()
</script>
