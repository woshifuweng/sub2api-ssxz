<template>
  <header class="mb-6 flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
    <div>
      <p class="text-xs font-semibold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
        {{ t('nav.resellerAdmin') }}
      </p>
      <h1 class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ title }}</h1>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  title: string
  description: string
}>()

const { t } = useI18n()
</script>
