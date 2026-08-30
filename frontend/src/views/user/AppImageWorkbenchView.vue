<template>
  <AppSectionShell
    title="图片工作台"
    subtitle="正在进入独立图片工作台，额度与计费仍使用当前平台账户。"
    eyebrow="创作工具"
    icon="photo"
  >
    <div class="workbench-redirect">
      <svg class="workbench-spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v3m0 12v3M4.22 4.22l2.12 2.12m11.32 11.32 2.12 2.12M3 12h3m12 0h3M4.22 19.78l2.12-2.12M17.66 6.34l2.12-2.12" />
      </svg>
      <p>正在打开图片工作台…</p>
      <a :href="IMAGE_WORKBENCH_URL" class="workbench-link">没有自动跳转？点击继续 →</a>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import { IMAGE_WORKBENCH_URL } from '@/constants/imageWorkbench'

onMounted(() => {
  window.location.replace(IMAGE_WORKBENCH_URL)
})
</script>

<style scoped>
.workbench-redirect {
  min-height: 24rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card, 10px);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  color: var(--ssxz-text-muted);
  font-size: 0.9rem;
}

.workbench-spinner {
  width: 2rem;
  height: 2rem;
  opacity: 0.4;
  animation: spin 1s linear infinite;
}

.workbench-link {
  color: var(--ssxz-accent, #6366f1);
  font-size: 0.85rem;
  text-decoration: none;
}

.workbench-link:hover {
  text-decoration: underline;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
