<template>
  <AppSectionShell
    title="图片创作"
    subtitle="图片工作台与无限画布，按你的创作方式选择。"
    eyebrow="创作工具"
    icon="photo"
  >
    <ImageToolsWorkspace />
  </AppSectionShell>
</template>

<script setup lang="ts">
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import ImageToolsWorkspace from '@/components/user/ImageToolsWorkspace.vue'
</script>
