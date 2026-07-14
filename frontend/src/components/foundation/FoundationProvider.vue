<template>
  <div class="f0-foundation" :class="{ 'f0-dark': theme === 'dark' }" :data-theme="theme">
    <slot />
  </div>
</template>

<script setup lang="ts">
import '@/foundation/foundation.css'

withDefaults(
  defineProps<{
    theme?: 'light' | 'dark'
  }>(),
  {
    theme: 'light'
  }
)
</script>
