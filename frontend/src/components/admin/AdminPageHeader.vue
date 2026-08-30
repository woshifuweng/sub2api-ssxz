<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <header
    class="mb-6 flex min-w-0 flex-col gap-4 border-b border-gray-200 pb-6 sm:flex-row sm:items-start sm:justify-between dark:border-dark-700"
  >
    <div class="min-w-0">
      <h1 class="m-0 text-xl font-bold leading-snug text-gray-900 dark:text-white">
        {{ title }}
      </h1>
      <p v-if="description" class="mt-1 text-sm leading-relaxed text-gray-500 dark:text-dark-400">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
