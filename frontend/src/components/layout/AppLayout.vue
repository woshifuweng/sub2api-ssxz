<template>
  <div class="ssxz-admin-shell min-h-screen">
    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="ssxz-admin-main relative min-h-screen transition-all duration-200"
      :class="{ 'is-collapsed': sidebarCollapsed }"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="ssxz-admin-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<style scoped>
.ssxz-admin-shell {
  background: var(--ssxz-bg, #070b14);
  color: var(--ssxz-text, #f1f5f9);
}

.ssxz-admin-main {
  position: relative;
}

.ssxz-admin-content {
  margin-inline: auto;
  max-width: var(--ssxz-content-max, 1360px);
  padding: var(--ssxz-space-page-y, 24px) var(--ssxz-space-page-x, 24px);
}

@media (min-width: 1024px) {
  .ssxz-admin-main {
    margin-left: var(--ssxz-sidebar-width, 248px);
  }

  .ssxz-admin-main.is-collapsed {
    margin-left: var(--ssxz-sidebar-collapsed-width, 72px);
  }
}

@media (max-width: 767px) {
  .ssxz-admin-content {
    padding: 16px;
  }
}

.ssxz-admin-shell :deep(.bg-white),
.ssxz-admin-shell :deep(.dark\:bg-dark-800),
.ssxz-admin-shell :deep(.dark\:bg-dark-900),
.ssxz-admin-shell :deep(.bg-gray-50),
.ssxz-admin-shell :deep(.bg-gray-100) {
  background-color: var(--ssxz-surface, #111827) !important;
}

.ssxz-admin-shell :deep(.text-gray-900),
.ssxz-admin-shell :deep(.dark\:text-white),
.ssxz-admin-shell :deep(.text-zinc-950) {
  color: var(--ssxz-text, #f1f5f9) !important;
}

.ssxz-admin-shell :deep(.text-gray-500),
.ssxz-admin-shell :deep(.text-gray-600),
.ssxz-admin-shell :deep(.dark\:text-dark-400),
.ssxz-admin-shell :deep(.dark\:text-gray-400) {
  color: var(--ssxz-text-muted, #94a3b8) !important;
}

.ssxz-admin-shell :deep(.border-gray-200),
.ssxz-admin-shell :deep(.dark\:border-dark-700),
.ssxz-admin-shell :deep(.dark\:border-dark-800) {
  border-color: var(--ssxz-border, #1f2937) !important;
}
</style>
