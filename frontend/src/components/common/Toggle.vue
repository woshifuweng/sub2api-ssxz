<template>
  <button
    type="button"
    class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full border p-0.5 transition-all duration-200 ease-out focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-offset-dark-800"
    :class="
      modelValue
        ? 'border-primary-600 bg-primary-600 dark:border-primary-500 dark:bg-primary-500'
        : 'border-gray-300 bg-gray-200 dark:border-dark-500 dark:bg-dark-600'
    "
    role="switch"
    :aria-checked="modelValue"
    :aria-label="ariaLabel"
    :data-state="modelValue ? 'on' : 'off'"
    :disabled="disabled"
    @click="toggle"
  >
    <span
      class="pointer-events-none inline-flex h-[18px] w-[18px] transform items-center justify-center rounded-full bg-white shadow-sm ring-1 ring-black/5 transition duration-200 ease-out"
      :class="modelValue ? 'translate-x-5' : 'translate-x-0'"
    >
      <Check
        v-if="modelValue"
        class="h-3 w-3 text-primary-700"
        :stroke-width="2.5"
        aria-hidden="true"
      />
      <X
        v-else
        class="h-3 w-3 text-gray-500 dark:text-gray-600"
        :stroke-width="2.25"
        aria-hidden="true"
      />
    </span>
  </button>
</template>

<script setup lang="ts">
import { Check, X } from '@lucide/vue'

const props = defineProps<{
  modelValue: boolean
  disabled?: boolean
  ariaLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>
