<template>
  <BrandLogo variant="animated" size="100%" :theme="theme" />
</template>

<script setup lang="ts">
import BrandLogo from '@/components/common/BrandLogo.vue'

defineProps<{
  theme: 'light' | 'dark'
}>()
</script>
