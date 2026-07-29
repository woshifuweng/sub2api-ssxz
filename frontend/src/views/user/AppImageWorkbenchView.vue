<template>
  <div class="workbench-shell">
    <iframe
      v-if="iframeSrc"
      :src="iframeSrc"
      class="workbench-frame"
      allow="clipboard-write"
      referrerpolicy="no-referrer"
    />
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
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
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
/* cancel ssxz-admin-content padding + max-width so iframe is edge-to-edge */
.workbench-shell {
  margin: calc(-1 * var(--ssxz-space-page-y, 24px)) calc(-1 * var(--ssxz-space-page-x, 24px));
  height: calc(100vh - var(--ssxz-header-height, 56px));
  max-width: none;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.workbench-frame {
  flex: 1;
  border: none;
  width: 100%;
  height: 100%;
  display: block;
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
</style>
