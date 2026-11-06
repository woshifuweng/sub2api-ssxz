<template>
  <PlatformIcon :platform="provider as GroupPlatform" :size="platformSize" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Provider } from '@/api/admin/channelMonitor'
import type { GroupPlatform } from '@/types'
import PlatformIcon from '@/components/common/PlatformIcon.vue'

const props = withDefaults(defineProps<{
  provider: Provider | string
  size?: number
}>(), {
  size: 20
})

const platformSize = computed<'xs' | 'sm' | 'md' | 'lg'>(() => {
  if (props.size <= 12) return 'xs'
  if (props.size <= 14) return 'sm'
  if (props.size <= 18) return 'md'
  return 'lg'
})
</script>
