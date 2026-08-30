<template>
  <component
    :is="component"
    :to="as === 'RouterLink' ? to : undefined"
    :type="as === 'RouterLink' ? undefined : type"
    :form="as === 'RouterLink' ? undefined : form"
    :disabled="as === 'RouterLink' ? undefined : disabled"
    :aria-disabled="disabled || undefined"
    class="btn"
    :class="buttonClass"
    @click="handleClick"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = withDefaults(defineProps<{
  as?: 'button' | 'RouterLink'
  to?: string
  type?: 'button' | 'submit' | 'reset'
  form?: string
  variant?: 'default' | 'outline' | 'destructive'
  size?: 'sm' | 'default'
  disabled?: boolean
}>(), {
  as: 'button',
  type: 'button',
  variant: 'default',
  size: 'default',
  disabled: false
})

const component = computed(() => props.as === 'RouterLink' ? RouterLink : 'button')
const buttonClass = computed(() => [
  props.variant === 'outline'
    ? 'btn-secondary'
    : props.variant === 'destructive'
      ? 'btn-danger'
      : 'btn-primary',
  props.size === 'sm' ? 'btn-sm' : 'btn-md',
  { 'pointer-events-none opacity-50': props.as === 'RouterLink' && props.disabled }
])

function handleClick(event: MouseEvent): void {
  if (props.disabled) event.preventDefault()
}
</script>
