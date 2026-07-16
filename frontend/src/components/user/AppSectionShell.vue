<template>
  <div
    class="ssxz-app-shell"
    data-app-shell-boundary="section"
    :class="{ 'ssxz-sidebar-collapsed': sidebarCollapsed, 'ssxz-mobile-nav-open': mobileNavActive }"
  >
    <button
      v-if="mobileNavActive"
      type="button"
      class="ssxz-mobile-sidebar-scrim lg:hidden"
      :aria-label="t('appShell.closeNavigation')"
      @click="closeMobileNav"
    />
    <aside class="ssxz-app-sidebar fixed inset-y-0 left-0 z-30 border-r px-3 py-4">
      <RouterLink
        to="/app/dashboard"
        class="ssxz-brand-link mb-6"
        :title="t('appShell.backToDashboard')"
        :aria-label="t('appShell.backToDashboard')"
        @click="closeMobileNav"
      >
        <BrandLogo class="ssxz-brand-logo" variant="mark" size="2.875rem" />
        <span class="ssxz-brand-copy ssxz-sidebar-text">
          <span class="ssxz-brand-title">AI Gateway</span>
          <span class="ssxz-brand-subtitle">{{ t('appShell.developerConsole') }}</span>
        </span>
      </RouterLink>

      <nav class="ssxz-primary-nav" :aria-label="t('appShell.primaryNavigation')">
        <button
          v-for="item in mainNavItems"
          :key="item.to"
          type="button"
          class="ssxz-nav-item"
          :class="{ 'is-active': isActive(item.to) }"
          :title="item.label"
          :aria-label="item.label"
          @click="handlePrimaryNav(item.to)"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.label }}</span>
        </button>
      </nav>

      <section v-if="showHistorySection" class="ssxz-history" :aria-label="t('appShell.conversationHistory')">
        <div class="ssxz-section-label ssxz-sidebar-text">{{ t('appShell.conversationHistory') }}</div>
        <button
          v-for="item in historyItems"
          :key="item.id"
          type="button"
          class="ssxz-nav-item ssxz-history-item"
          :class="{ 'is-active': item.id === activeConversationId }"
          :title="item.title"
          :aria-label="item.title"
          @click="handleHistorySelect(item.id)"
        >
          <Icon name="chat" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.title || t('appShell.untitledConversation') }}</span>
        </button>
        <p v-if="historyLoading" class="ssxz-empty-history ssxz-sidebar-text">
          {{ t('appShell.syncingHistory') }}
        </p>
        <p v-if="!historyLoading && historyItems.length === 0" class="ssxz-empty-history ssxz-sidebar-text">
          {{ t('appShell.noHistory') }}
        </p>
      </section>

    </aside>

    <main class="ssxz-app-content min-h-screen">
      <header class="ssxz-app-header sticky top-0 z-20 flex items-center border-b px-4 sm:px-6">
        <div class="flex w-full items-center justify-between gap-3">
          <button
            type="button"
            class="ssxz-btn-icon ssxz-sidebar-toggle-desktop"
            :aria-label="navToggleLabel"
            :title="navToggleLabel"
            :aria-expanded="navToggleExpanded"
            @click="toggleShellNav"
          >
            <Icon name="menu" size="sm" />
          </button>
          <div class="flex items-center gap-2">
            <RouterLink
              to="/docs"
              class="ssxz-header-docs"
              :title="t('nav.docs')"
              :aria-label="t('nav.docs')"
            >
              <Icon name="book" size="sm" />
              <span class="hidden sm:inline">{{ t('nav.docs') }}</span>
            </RouterLink>
            <ThemeToggle />
            <div v-if="authStore.isAuthenticated" class="relative">
              <div class="ssxz-account-cluster">
                <span class="ssxz-balance-pill" :title="userBalanceTitle">{{ t('appShell.balance') }} {{ userBalance }}</span>
                <button type="button" class="ssxz-user-button" @click="userMenuOpen = !userMenuOpen">
                <span class="ssxz-user-avatar">{{ userInitial }}</span>
                <span class="hidden max-w-32 truncate sm:inline">{{ userLabel }}</span>
                <Icon name="chevronDown" size="xs" />
                </button>
              </div>
              <div v-if="userMenuOpen" class="ssxz-user-menu">
                <div class="ssxz-menu-summary">
                  <strong>{{ userLabel }}</strong>
                  <span :title="userBalanceTitle">{{ t('appShell.balance') }} {{ userBalance }}</span>
                </div>
                <button v-if="authStore.isAdmin" type="button" class="ssxz-menu-link" @click="openAdminConsole">
                  {{ t('appShell.adminConsole') }}
                </button>
                <button type="button" class="ssxz-menu-link text-red-600 dark:text-red-300" @click="logout">
                  {{ t('nav.logout') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </header>

      <div class="ssxz-app-main relative z-10">
        <section class="ssxz-page-heading">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div class="ssxz-eyebrow">
                <Icon :name="icon" size="xs" />
                {{ eyebrow }}
              </div>
              <h2 class="mt-3 text-2xl font-semibold tracking-normal sm:text-3xl">{{ title }}</h2>
              <p class="mt-2 max-w-3xl text-sm leading-6 text-zinc-600 dark:text-zinc-200">{{ subtitle }}</p>
            </div>
            <slot name="actions" />
          </div>
        </section>

        <slot />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BrandLogo from '@/components/common/BrandLogo.vue'
import Icon from '@/components/icons/Icon.vue'
import ThemeToggle from '@/components/common/ThemeToggle.vue'
import type { ChatConversation } from '@/api/chatWorkspace'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { getSafeLocalStorageItem, setSafeLocalStorageItem } from '@/utils/safeStorage'
import { formatCurrency, formatCurrencyTitle } from '@/utils/format'

type IconName = InstanceType<typeof Icon>['$props']['name']

const props = withDefaults(defineProps<{
  title: string
  subtitle: string
  eyebrow?: string
  icon?: IconName
  historyItems?: ChatConversation[]
  activeConversationId?: number | null
  historyLoading?: boolean
}>(), {
  eyebrow: 'SSXZ AI',
  icon: 'sparkles',
  historyItems: () => [],
  activeConversationId: null,
  historyLoading: false
})

const emit = defineEmits<{
  (e: 'new-chat'): void
  (e: 'select-conversation', id: number): void
}>()

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const userMenuOpen = ref(false)
const SIDEBAR_COLLAPSED_KEY = 'ssxz.app.sidebar.collapsed'
const sidebarCollapsed = ref(readSidebarCollapsed())
const mobileNavOpen = ref(false)
const isDesktopViewport = ref(false)
let desktopMediaQuery: MediaQueryList | null = null

const mainNavItems = computed<Array<{ label: string; to: string; icon: IconName }>>(() => [
  { label: t('nav.dashboard'), to: '/app/dashboard', icon: 'home' },
  { label: t('nav.modelTest'), to: '/app/chat', icon: 'chat' },
  { label: t('nav.image'), to: '/app/image', icon: 'sparkles' },
  { label: t('nav.apiKeys'), to: '/app/keys', icon: 'key' },
  { label: t('nav.models'), to: '/app/available-channels', icon: 'calculator' },
  { label: t('nav.usage'), to: '/app/usage', icon: 'chartBar' },
  { label: t('nav.channelStatus'), to: '/app/channel-status', icon: 'chartBar' },
  { label: t('nav.billing'), to: '/app/purchase', icon: 'creditCard' },
  { label: t('nav.orders'), to: '/app/orders', icon: 'document' },
  { label: t('nav.redeem'), to: '/app/redeem', icon: 'gift' },
  ...(appStore.cachedPublicSettings?.affiliate_enabled
    ? [{ label: t('nav.affiliate'), to: '/app/affiliate', icon: 'users' as IconName }]
    : []),
  { label: t('nav.account'), to: '/app/profile', icon: 'userCircle' }
])

const userLabel = computed(() => authStore.user?.username || authStore.user?.email?.split('@')[0] || t('appShell.accountFallback'))
const userInitial = computed(() => userLabel.value.slice(0, 1).toUpperCase())
const userBalance = computed(() => formatCurrency(authStore.user?.balance || 0))
const userBalanceTitle = computed(() => formatCurrencyTitle(authStore.user?.balance || 0))
const navToggleLabel = computed(() => {
  if (!isDesktopViewport.value) return mobileNavOpen.value ? t('appShell.closeNavigation') : t('appShell.openNavigation')
  return sidebarCollapsed.value ? t('appShell.expandSidebar') : t('appShell.collapseSidebar')
})
const navToggleExpanded = computed(() => !isDesktopViewport.value ? mobileNavOpen.value : !sidebarCollapsed.value)
const mobileNavActive = computed(() => mobileNavOpen.value && !isDesktopViewport.value)
const showHistorySection = computed(() => (
  route.path === '/app/chat' || props.historyLoading || props.historyItems.length > 0
))

function isActive(path: string) {
  const normalizedPath = path.split('?')[0]
  if (normalizedPath === '/app') return route.path === '/app' || route.path === '/app/dashboard'
  return route.path === normalizedPath
}

function handlePrimaryNav(to: string) {
  closeMobileNav()
  if (to === '/app/chat') {
    emit('new-chat')
    if (route.path !== '/app/chat') router.push('/app/chat')
    return
  }
  router.push(to)
}

function openAdminConsole() {
  userMenuOpen.value = false
  router.push('/admin/dashboard')
}

function handleHistorySelect(id: number) {
  emit('select-conversation', id)
  closeMobileNav()
}

function readSidebarCollapsed() {
  return getSafeLocalStorageItem(SIDEBAR_COLLAPSED_KEY) === 'true'
}

function setSidebarCollapsed(value: boolean) {
  sidebarCollapsed.value = value
  setSafeLocalStorageItem(SIDEBAR_COLLAPSED_KEY, value ? 'true' : 'false')
}

function toggleSidebarCollapsed() {
  setSidebarCollapsed(!sidebarCollapsed.value)
}

function toggleShellNav() {
  if (!isDesktopViewport.value) {
    mobileNavOpen.value = !mobileNavOpen.value
    return
  }
  toggleSidebarCollapsed()
}

function closeMobileNav() {
  mobileNavOpen.value = false
}

function syncViewportMode() {
  if (typeof window === 'undefined') return
  if (typeof window.matchMedia !== 'function') {
    isDesktopViewport.value = true
    closeMobileNav()
    return
  }
  isDesktopViewport.value = window.matchMedia('(min-width: 1024px)').matches
  if (isDesktopViewport.value) closeMobileNav()
}

async function logout() {
  await authStore.logout()
  userMenuOpen.value = false
  appStore.showSuccess(t('appShell.loggedOut'))
  router.push('/app')
}

onMounted(() => {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    desktopMediaQuery = window.matchMedia('(min-width: 1024px)')
    syncViewportMode()
    desktopMediaQuery.addEventListener('change', syncViewportMode)
  } else {
    syncViewportMode()
  }
  window.addEventListener('resize', syncViewportMode)
})

onBeforeUnmount(() => {
  desktopMediaQuery?.removeEventListener('change', syncViewportMode)
  window.removeEventListener('resize', syncViewportMode)
})

watch(() => route.fullPath || route.path, closeMobileNav)
</script>

<style scoped>
.ssxz-app-shell {
  min-height: 100vh;
  overflow-x: hidden;
  background: var(--ssxz-bg);
  color: var(--ssxz-text);
}

.dark .ssxz-app-shell {
  color: var(--ssxz-text);
}

.ssxz-app-sidebar {
  width: var(--ssxz-sidebar-width);
  border-color: var(--ssxz-border);
  background: var(--ssxz-surface);
  color: var(--ssxz-body);
}

.ssxz-brand-link {
  display: inline-flex;
  min-height: 3rem;
  width: 100%;
  align-items: center;
  gap: 0.72rem;
  border-radius: var(--ssxz-radius-button);
  color: var(--ssxz-text);
  padding: 0.28rem;
}

.ssxz-brand-link:hover {
  background: color-mix(in srgb, var(--ssxz-primary) 8%, transparent);
}

.ssxz-brand-logo {
  color: var(--ssxz-text);
}

.ssxz-brand-copy {
  display: grid;
  gap: 0.05rem;
}

.ssxz-brand-title {
  font-size: 0.9rem;
  font-weight: 650;
}

.ssxz-brand-subtitle {
  color: var(--ssxz-subtle);
  font-size: 0.7rem;
  font-weight: 500;
}

.ssxz-app-content {
  position: relative;
  z-index: 1;
}

.ssxz-app-header {
  min-height: var(--ssxz-header-height);
  border-color: var(--ssxz-border);
  background: var(--ssxz-surface);
}

.ssxz-app-main {
  margin-inline: auto;
  width: min(100%, var(--ssxz-content-max));
  padding: var(--ssxz-space-page-y) var(--ssxz-space-page-x);
}

.ssxz-page-heading {
  margin-bottom: 1.5rem;
  padding: 0.25rem 0;
}

.ssxz-page-heading h2 {
  color: var(--ssxz-text);
}

.ssxz-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--ssxz-text-muted);
  font-size: 0.76rem;
  font-weight: 550;
}

.ssxz-btn-icon {
  display: inline-flex;
  width: 2.2rem;
  height: 2.2rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: var(--ssxz-body);
}

.ssxz-btn-icon:hover {
  background: color-mix(in srgb, var(--ssxz-primary) 10%, transparent);
  color: var(--ssxz-text);
}

.ssxz-header-docs {
  display: inline-flex;
  min-height: 2.2rem;
  align-items: center;
  gap: 0.4rem;
  border-radius: var(--ssxz-radius-button);
  color: var(--ssxz-body);
  font-size: 0.82rem;
  font-weight: 650;
  padding: 0 0.65rem;
}

.ssxz-header-docs:hover,
.ssxz-header-docs:focus-visible {
  background: color-mix(in srgb, var(--ssxz-primary) 10%, transparent);
  color: var(--ssxz-text);
}

.ssxz-user-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 0.6rem);
  z-index: 30;
  min-width: 12rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
  padding: 0.65rem;
}

.ssxz-menu-link {
  width: 100%;
  border-radius: 0.75rem;
  padding: 0.55rem 0.35rem;
  text-align: left;
  font-size: 0.86rem;
  font-weight: 750;
}

.ssxz-user-avatar {
  align-items: center;
  background: var(--ssxz-primary);
  border-radius: 999px;
  color: var(--ssxz-action-text);
  display: inline-flex;
  font-size: 0.75rem;
  font-weight: 760;
  height: 1.75rem;
  justify-content: center;
  width: 1.75rem;
}

.ssxz-app-sidebar {
  display: none;
}

.ssxz-mobile-sidebar-scrim {
  position: fixed;
  inset: 0;
  z-index: 25;
  border: 0;
  background: rgb(0 0 0 / 0.58);
  backdrop-filter: blur(2px);
}

.ssxz-mobile-nav-open .ssxz-app-sidebar {
  display: block;
  box-shadow: 18px 0 50px rgb(0 0 0 / 0.35);
}

@media (min-width: 1024px) {
  .ssxz-app-sidebar {
    display: block;
  }

  .ssxz-app-content {
    margin-left: var(--ssxz-sidebar-width);
  }

  .ssxz-sidebar-collapsed .ssxz-app-content {
    margin-left: var(--ssxz-sidebar-collapsed-width);
  }

  .ssxz-sidebar-collapsed .ssxz-app-sidebar {
    width: var(--ssxz-sidebar-collapsed-width);
  }

  .ssxz-sidebar-collapsed .ssxz-sidebar-text {
    display: none;
  }
}

.ssxz-nav-item,
.ssxz-theme-toggle {
  display: inline-flex;
  box-sizing: border-box;
  min-height: 3rem;
  min-width: 0;
  width: 100%;
  max-width: 100%;
  align-items: center;
  gap: 0.75rem;
  border-radius: 0.75rem;
  color: var(--ssxz-body);
  font-size: 0.9375rem;
  font-synthesis: none;
  font-weight: 500;
  line-height: 1.35;
  padding: 0.7rem 0.9rem;
  text-align: left;
}

.ssxz-nav-item:hover,
.ssxz-theme-toggle:hover {
  background: color-mix(in srgb, var(--ssxz-primary) 16%, transparent);
  color: var(--ssxz-text);
}

.ssxz-nav-item.is-active {
  background: var(--ssxz-primary);
  color: var(--ssxz-action-text);
  box-shadow: var(--ssxz-shadow-button-subtle);
}

.ssxz-nav-item svg,
.ssxz-theme-toggle svg {
  flex: 0 0 auto;
}

.ssxz-primary-nav {
  display: grid;
  gap: 0.55rem;
}

.ssxz-app-shell :deep(.bg-white),
.ssxz-app-shell :deep(.bg-gray-50),
.ssxz-app-shell :deep(.bg-gray-100),
.ssxz-app-shell :deep(.bg-blue-50),
.ssxz-app-shell :deep(.bg-purple-50),
.ssxz-app-shell :deep(.dark\:bg-blue-950),
.ssxz-app-shell :deep(.dark\:bg-purple-950),
.ssxz-app-shell :deep(.dark\:bg-dark-700),
.ssxz-app-shell :deep(.dark\:bg-dark-800),
.ssxz-app-shell :deep(.dark\:bg-dark-900),
.ssxz-app-shell :deep([class*='bg-white/']),
.ssxz-app-shell :deep([class*='bg-gray-50/']),
.ssxz-app-shell :deep([class*='bg-gray-100/']),
.ssxz-app-shell :deep([class*='dark:bg-dark-800/']),
.ssxz-app-shell :deep([class*='dark:bg-dark-900/']) {
  background-color: color-mix(in srgb, var(--ssxz-surface-raised) 88%, transparent) !important;
}

.ssxz-app-shell :deep(.border-gray-100),
.ssxz-app-shell :deep(.border-gray-200),
.ssxz-app-shell :deep(.border-blue-100),
.ssxz-app-shell :deep(.border-blue-200),
.ssxz-app-shell :deep(.border-purple-100),
.ssxz-app-shell :deep(.dark\:border-blue-900),
.ssxz-app-shell :deep(.dark\:border-purple-900),
.ssxz-app-shell :deep(.dark\:border-dark-600),
.ssxz-app-shell :deep(.dark\:border-dark-700),
.ssxz-app-shell :deep([class*='border-gray-200/']),
.ssxz-app-shell :deep([class*='dark:border-dark-700/']) {
  border-color: var(--ssxz-border) !important;
}

.ssxz-app-shell :deep(.text-gray-900),
.ssxz-app-shell :deep(.text-gray-800),
.ssxz-app-shell :deep(.dark\:text-white),
.ssxz-app-shell :deep(.dark\:text-gray-100) {
  color: var(--ssxz-text) !important;
}

.ssxz-app-shell :deep(.text-gray-700),
.ssxz-app-shell :deep(.text-gray-600),
.ssxz-app-shell :deep(.text-gray-500),
.ssxz-app-shell :deep(.dark\:text-gray-300),
.ssxz-app-shell :deep(.dark\:text-gray-400),
.ssxz-app-shell :deep(.dark\:text-dark-300),
.ssxz-app-shell :deep(.dark\:text-dark-400) {
  color: var(--ssxz-text-muted) !important;
}

.ssxz-app-shell :deep(.text-primary-600),
.ssxz-app-shell :deep(.text-primary-700),
.ssxz-app-shell :deep(.dark\:text-primary-400),
.ssxz-app-shell :deep(.text-blue-600),
.ssxz-app-shell :deep(.dark\:text-blue-400),
.ssxz-app-shell :deep(.text-purple-600),
.ssxz-app-shell :deep(.dark\:text-purple-400),
.ssxz-app-shell :deep(.text-teal-600),
.ssxz-app-shell :deep(.dark\:text-teal-400) {
  color: var(--ssxz-accent) !important;
}

.ssxz-app-shell :deep(a.bg-emerald-600),
.ssxz-app-shell :deep(button.bg-emerald-600),
.ssxz-app-shell :deep(a.hover\:bg-emerald-700:hover),
.ssxz-app-shell :deep(button.hover\:bg-emerald-700:hover),
.ssxz-app-shell :deep([class*='bg-gradient-to']) {
  background: var(--ssxz-primary) !important;
  color: var(--ssxz-action-text) !important;
}

.ssxz-app-shell :deep([class*='hover:bg-primary-50']:hover),
.ssxz-app-shell :deep([class*='dark:hover:bg-primary-950']:hover),
.ssxz-app-shell :deep([class*='hover:border-primary']:hover) {
  background-color: color-mix(in srgb, var(--ssxz-primary) 12%, transparent) !important;
  border-color: var(--ssxz-border-strong) !important;
}

.ssxz-secondary-nav {
  display: grid;
  gap: 0.28rem;
  margin-top: 1.05rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--ssxz-border);
}

.ssxz-utility-item {
  border: 0;
  text-align: left;
}

.ssxz-nav-badge {
  margin-left: auto;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 85%, transparent);
  color: var(--ssxz-subtle);
  font-size: 0.68rem;
  font-weight: 850;
  line-height: 1;
  padding: 0.2rem 0.38rem;
}

.ssxz-utility-panel {
  display: grid;
  gap: 0.3rem;
  margin: 0.7rem 0.15rem 0;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.95rem;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 78%, transparent);
  color: var(--ssxz-subtle);
  font-size: 0.78rem;
  line-height: 1.55;
  padding: 0.72rem 0.78rem;
}

.ssxz-utility-title {
  color: var(--ssxz-text);
  font-size: 0.82rem;
  font-weight: 760;
}

.ssxz-usage-panel {
  display: grid;
  gap: 0.48rem;
  margin-top: 1rem;
}

.ssxz-usage-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.9rem;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 82%, transparent);
  color: var(--ssxz-body);
  font-size: 0.8rem;
  padding: 0.68rem 0.78rem;
}

.ssxz-usage-row strong {
  color: var(--ssxz-text);
  font-size: 0.88rem;
}

.ssxz-usage-note {
  color: var(--ssxz-subtle);
  font-size: 0.76rem;
  line-height: 1.45;
  margin: 0 0.2rem;
}

.ssxz-new-chat {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-sm);
}

.ssxz-history {
  display: grid;
  gap: 0.5rem;
  margin-top: 1.4rem;
  min-width: 0;
  max-width: 100%;
  max-height: min(32rem, calc(100vh - 23rem));
  overflow-x: hidden;
  overflow-y: auto;
  padding-right: 0.15rem;
}

.ssxz-section-label {
  color: var(--ssxz-subtle);
  font-size: 0.76rem;
  font-weight: 760;
  padding: 0 0.7rem;
}

.ssxz-empty-history {
  border: 1px dashed var(--ssxz-border);
  border-radius: 0.95rem;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 68%, transparent);
  color: var(--ssxz-subtle);
  font-size: 0.82rem;
  line-height: 1.5;
  margin: 0 0.15rem;
  padding: 0.8rem 0.85rem;
}

.ssxz-history-item {
  align-items: flex-start;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
}

.ssxz-history-item .ssxz-sidebar-text {
  display: -webkit-box;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-height: 1.35;
  text-align: left;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.ssxz-sidebar-bottom {
  position: absolute;
  bottom: 1rem;
  left: 0.75rem;
  right: 0.75rem;
  display: grid;
  gap: 0.6rem;
}

.ssxz-secondary-links {
  display: grid;
  gap: 0.18rem;
  opacity: 0.84;
}

.ssxz-account-cluster {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  white-space: nowrap;
}

.ssxz-balance-pill {
  display: inline-flex;
  flex: 0 0 auto;
  min-height: 2.1rem;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 88%, transparent);
  color: var(--ssxz-text);
  font-size: 0.82rem;
  font-weight: 760;
  padding: 0 0.75rem;
}

:deep(.ssxz-user-button) {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.45rem;
  white-space: nowrap;
}

.ssxz-menu-summary {
  display: grid;
  gap: 0.18rem;
  border-bottom: 1px solid var(--ssxz-border);
  color: var(--ssxz-body);
  font-size: 0.8rem;
  line-height: 1.45;
  margin-bottom: 0.35rem;
  padding: 0.2rem 0.35rem 0.55rem;
}

.ssxz-menu-summary strong {
  color: var(--ssxz-text);
  font-size: 0.86rem;
}

@media (max-width: 640px) {
  .ssxz-app-main {
    padding: 16px;
  }

  .ssxz-balance-pill {
    display: none;
  }
}
</style>
