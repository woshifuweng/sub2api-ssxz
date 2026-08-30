<template>
  <span :class="['f0-badge', `f0-badge--${variant}`]">
    <slot />
  </span>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive'
  }>(),
  {
    variant: 'default'
  }
)
</script>
