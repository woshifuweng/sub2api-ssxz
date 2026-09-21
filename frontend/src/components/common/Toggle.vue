<template>
  <span class="inline-flex min-w-0 items-center gap-2">
    <button
      v-bind="$attrs"
      type="button"
      @click="toggle"
      class="relative inline-flex h-7 w-12 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-dark-800"
      :class="[modelValue ? 'bg-blue-500' : 'bg-gray-500 dark:bg-dark-500']"
      role="switch"
      :aria-checked="modelValue"
      :aria-label="label || undefined"
      :aria-describedby="descriptionId || undefined"
    >
      <span
        class="pointer-events-none inline-block h-6 w-6 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
        :class="[modelValue ? 'translate-x-5' : 'translate-x-0']"
      />
    </button>
    <span
      v-if="showState"
      aria-hidden="true"
      class="whitespace-nowrap text-xs font-medium text-gray-600 dark:text-gray-300"
    >
      {{ modelValue ? '开' : '关' }}
    </span>
  </span>
</template>

<script setup lang="ts">
defineOptions({ inheritAttrs: false })

const props = defineProps<{
  modelValue: boolean
  label?: string
  descriptionId?: string
  showState?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  emit('update:modelValue', !props.modelValue)
}
</script>
