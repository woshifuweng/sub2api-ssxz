<template>
  <AppSectionShell
    title="图片工作台"
    subtitle="在站内直接生成与改图，额度与计费同账户记录一致。"
    eyebrow="创作工具"
    icon="photo"
  >
    <div class="workbench-workbench">
      <iframe
        v-if="iframeSrc"
        :src="iframeSrc"
        class="workbench-embed-frame"
        title="图片工作台"
        allow="clipboard-write"
        referrerpolicy="no-referrer"
      ></iframe>
      <div v-else-if="loading" class="workbench-placeholder">
        <svg class="workbench-spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v3m0 12v3M4.22 4.22l2.12 2.12m11.32 11.32 2.12 2.12M3 12h3m12 0h3M4.22 19.78l2.12-2.12M17.66 6.34l2.12-2.12" />
        </svg>
        <span>正在加载图片工作台…</span>
      </div>
      <div v-else class="workbench-placeholder">
        <svg class="workbench-empty-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
        </svg>
        <p>需要先创建 API Key 才能使用图片工作台</p>
        <RouterLink to="/app/keys" class="workbench-key-link">前往创建 API Key →</RouterLink>
      </div>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import { list as listKeys, reveal as revealKey } from '@/api/keys'

const iframeSrc = ref<string>('')
const loading = ref(true)

onMounted(async () => {
  try {
    // fetch first active key and pass via URL param so workbench is pre-authorised
    const res = await listKeys(1, 20, { status: '1' })
    const keys = res.items ?? []
    const first = keys[0]
    if (first?.id) {
      const { key } = await revealKey(first.id)
      iframeSrc.value = `/image/?apiKey=${encodeURIComponent(key)}`
    } else {
      // no active key — open without pre-fill, user can set manually
      iframeSrc.value = '/image/'
    }
  } catch {
    iframeSrc.value = '/image/'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
/* Same card treatment as the recharge page (AppPurchaseView) so the workbench
   reads as part of the platform instead of a separate site. Taller than the
   shop embed because this is a hands-on tool, not a product list. */
.workbench-workbench {
  width: 100%;
  height: calc(100vh - 13rem);
  min-height: 34rem;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card, 10px);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
  display: flex;
  flex-direction: column;
}

.workbench-embed-frame {
  display: block;
  flex: 1;
  width: 100%;
  border: none;
}

.workbench-placeholder {
  flex: 1;
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

.workbench-empty-icon {
  width: 3rem;
  height: 3rem;
  opacity: 0.3;
}

.workbench-key-link {
  color: var(--ssxz-accent, #6366f1);
  font-size: 0.85rem;
  text-decoration: none;
}
.workbench-key-link:hover { text-decoration: underline; }

@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 760px) {
  .workbench-workbench {
    height: calc(100vh - 11rem);
    min-height: 30rem;
  }
}
</style>
