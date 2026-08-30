<script setup lang="ts">
import { computed, useAttrs } from 'vue'

defineOptions({ inheritAttrs: false })

type ButtonVariant = 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'plain' | 'link'
type ButtonSize = 'default' | 'sm' | 'lg' | 'xl' | 'xxl' | 'icon'

const props = withDefaults(
  defineProps<{
    as?: string | object
    variant?: ButtonVariant
    size?: ButtonSize
  }>(),
  {
    as: 'button',
    variant: 'default',
    size: 'default',
  },
)

const attrs = useAttrs()

const variantClass: Record<ButtonVariant, string> = {
  default: 'btn-primary',
  destructive: 'btn-danger',
  outline: 'btn-secondary',
  secondary: 'btn-secondary',
  ghost: 'btn-ghost',
  plain: '',
  link: 'btn-ghost underline underline-offset-4',
}

const sizeClass: Record<ButtonSize, string> = {
  default: 'btn-md',
  sm: 'btn-sm',
  lg: 'btn-lg',
  xl: 'btn-lg px-6',
  xxl: 'btn-lg px-8',
  icon: 'btn-icon',
}

const buttonClass = computed(() => [
  props.variant === 'plain' ? '' : 'btn',
  variantClass[props.variant],
  props.variant === 'plain' && props.size !== 'icon' ? 'h-auto min-h-0 w-auto p-0' : sizeClass[props.size],
  attrs.class,
])
</script>

<template>
  <component :is="props.as" v-bind="attrs" :class="buttonClass">
    <slot />
  </component>
</template>
