<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      referrerpolicy="no-referrer"
      :src="homeContentUrl"
      class="h-screen w-full border-0"
      allowfullscreen
    />
    <div v-else v-html="renderedHomeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">
          {{ siteSubtitle }}
        </p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- SSXZ branded default home -->
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
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AetherHomeExperience from '@/components/home/aether/AetherHomeExperience.vue'
import { FoundationProvider } from '@/components/foundation'
import { useAppStore, useAuthStore } from '@/stores'
import { DEFAULT_SITE_NAME, normalizeSiteName } from '@/utils/brand'
import { resolvePublicApiBaseUrl } from '@/utils/publicApiBaseUrl'
import { renderRichContent } from '@/utils/sanitize'
import { setSafeLocalStorageItem } from '@/utils/safeStorage'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
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
const siteLogo = computed(() => sanitizeUrl(
  appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '',
  { allowRelative: true, allowDataUrl: true },
))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const homeContentUrl = computed(() => sanitizeUrl(homeContent.value))
const renderedHomeContent = computed(() => renderRichContent(homeContent.value))
const isHomeContentUrl = computed(() => Boolean(homeContentUrl.value))
const displayedApiBaseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl
  const currentOrigin = typeof window !== 'undefined' ? window.location.origin : ''
  return resolvePublicApiBaseUrl(configured, currentOrigin)
})

const isDark = computed(() => theme.value === 'dark')
const currentYear = computed(() => new Date().getFullYear())
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/app/dashboard')

// Preserve affiliate attribution when the public home page forwards a visitor to registration.
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
    attributeFilter: ['class'],
  })
  void authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings()
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})
</script>
