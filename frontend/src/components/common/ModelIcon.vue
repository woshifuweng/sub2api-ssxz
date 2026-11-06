<template>
  <LobeModelIcon
    :model="model"
    :size="iconSize"
    type="color"
    class="model-icon"
    :aria-label="label || undefined"
    :aria-hidden="label ? undefined : true"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ModelIcon as LobeModelIcon } from '@lobehub/icons-vue'

const props = withDefaults(defineProps<{
  model: string
  size?: string | number
  label?: string
}>(), {
  size: '18px',
  label: ''
})

const iconSize = computed(() => {
  if (typeof props.size === 'number') return props.size
  const parsed = Number.parseFloat(props.size)
  return Number.isFinite(parsed) ? parsed : 18
})
</script>

<style scoped>
.model-icon {
  display: block;
  flex-shrink: 0;
}
</style>
