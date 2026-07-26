<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      referrerpolicy="no-referrer"
      :src="homeContentUrl"
      class="h-screen w-full border-0"
      allowfullscreen
    />
    <div v-else v-html="renderedHomeContent"></div>
  </div>

  <FoundationProvider v-else :theme="theme">
    <AetherHomeExperience
      :theme="theme"
      :site-name="siteName"
      :doc-url="docUrl"
      :base-url="displayedApiBaseUrl"
      :is-authenticated="isAuthenticated"
      :dashboard-path="dashboardPath"
      :primary-cta-path="primaryCtaPath"
      :create-key-path="createKeyPath"
      @toggle-theme="toggleTheme"
      @copy="copyCode"
    />
  </FoundationProvider>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AetherHomeExperience from '@/components/home/aether/AetherHomeExperience.vue'
import { FoundationProvider } from '@/components/foundation'
import { useAppStore, useAuthStore } from '@/stores'
import { DEFAULT_SITE_NAME, normalizeSiteName } from '@/utils/brand'
import { resolvePublicApiBaseUrl } from '@/utils/publicApiBaseUrl'
import { renderRichContent } from '@/utils/sanitize'
import { setSafeLocalStorageItem } from '@/utils/safeStorage'
import { sanitizeUrl } from '@/utils/url'

const authStore = useAuthStore()
const appStore = useAppStore()
const route = useRoute()

function getInitialTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'dark'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const theme = ref<'light' | 'dark'>(getInitialTheme())
let themeObserver: MutationObserver | null = null

const siteName = computed(() =>
  normalizeSiteName(appStore.cachedPublicSettings?.site_name || appStore.siteName || DEFAULT_SITE_NAME)
)
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const homeContentUrl = computed(() => sanitizeUrl(homeContent.value))
const renderedHomeContent = computed(() => renderRichContent(homeContent.value))
const isHomeContentUrl = computed(() => Boolean(homeContentUrl.value))
const displayedApiBaseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl
  const currentOrigin = typeof window !== 'undefined' ? window.location.origin : ''
  return resolvePublicApiBaseUrl(configured, currentOrigin)
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/app/dashboard')
// 邀请链接落到首页时保留邀请码，注册页会从 ?aff= 预填(见 RegisterView onMounted)
const registerPath = computed(() => {
  const raw = route.query.aff ?? route.query.affiliate
  const code = Array.isArray(raw) ? raw[0] : raw
  return code ? `/register?aff=${encodeURIComponent(String(code))}` : '/register'
})
const primaryCtaPath = computed(() => isAuthenticated.value ? dashboardPath.value : registerPath.value)
const createKeyPath = computed(() => isAuthenticated.value ? '/app/keys' : registerPath.value)

function syncTheme(): void {
  theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

function toggleTheme(): void {
  const nextTheme = theme.value === 'dark' ? 'light' : 'dark'
  document.documentElement.classList.toggle('dark', nextTheme === 'dark')
  setSafeLocalStorageItem('theme', nextTheme)
  syncTheme()
}

async function copyCode(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    appStore.showSuccess('代码已复制')
  } catch {
    appStore.showError('复制失败，请手动复制')
  }
}

onMounted(() => {
  syncTheme()
  themeObserver = new MutationObserver(syncTheme)
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class']
  })
  void authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings()
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})
</script>
