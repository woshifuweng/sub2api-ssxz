<template>
  <button
    :type="type"
    :disabled="disabled"
    :class="['f0-button', `f0-button--${variant}`, `f0-button--${size}`]"
  >
    <slot name="leading" />
    <slot />
    <slot name="trailing" />
  </button>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive'
    size?: 'sm' | 'default' | 'lg' | 'icon'
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
  }>(),
  {
    variant: 'default',
    size: 'default',
    type: 'button',
    disabled: false
  }
)
</script>
