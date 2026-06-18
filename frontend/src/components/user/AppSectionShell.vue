<template>
  <div
    class="ssxz-app-shell"
    data-app-shell-boundary="section"
    :class="{ 'ssxz-sidebar-collapsed': sidebarCollapsed, 'ssxz-mobile-nav-open': mobileNavOpen }"
  >
    <div class="ssxz-app-backdrop" aria-hidden="true" />
    <button
      v-if="mobileNavOpen"
      type="button"
      class="ssxz-mobile-sidebar-scrim lg:hidden"
      aria-label="关闭导航"
      @click="closeMobileNav"
    />
    <aside class="ssxz-app-sidebar fixed inset-y-0 left-0 z-30 w-60 border-r px-3 py-4 backdrop-blur-xl">
      <RouterLink to="/app" class="ssxz-brand-link mb-6" title="返回工作台首页" aria-label="返回工作台首页" @click="closeMobileNav">
        <span class="ssxz-brand-mark">S</span>
        <span class="ssxz-brand-copy ssxz-sidebar-text">
          <span class="ssxz-brand-title">SSXZ AI</span>
          <span class="ssxz-brand-subtitle">AI 创作工作台</span>
        </span>
      </RouterLink>

      <nav class="ssxz-primary-nav" aria-label="主导航">
        <button
          v-for="item in navItems"
          :key="item.to"
          type="button"
          class="ssxz-nav-item ssxz-new-chat"
          :class="{ 'is-active': isActive(item.to) }"
          :title="item.label"
          :aria-label="item.label"
          @click="handlePrimaryNav(item.to)"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.label }}</span>
        </button>
      </nav>

      <nav class="ssxz-secondary-nav" aria-label="工作台入口">
        <button
          v-for="item in utilityItems"
          :key="item.to"
          type="button"
          class="ssxz-nav-item ssxz-utility-item"
          :class="{ 'is-active': isActive(item.to) }"
          :title="item.label"
          :aria-label="item.label"
          @click="handleRouteNav(item.to)"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.label }}</span>
        </button>
      </nav>

      <section class="ssxz-history" aria-label="历史会话">
        <div class="ssxz-section-label ssxz-sidebar-text">历史会话</div>
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
          <span class="ssxz-sidebar-text">{{ item.title || '未命名对话' }}</span>
        </button>
        <p v-if="historyLoading" class="ssxz-empty-history ssxz-sidebar-text">
          正在同步历史...
        </p>
        <p v-if="!historyLoading && historyItems.length === 0" class="ssxz-empty-history ssxz-sidebar-text">
          暂无历史会话
        </p>
      </section>

      <div class="ssxz-sidebar-bottom">
        <button
          type="button"
          class="ssxz-theme-toggle"
          :title="isDark ? '浅色模式' : '深色模式'"
          :aria-label="isDark ? '切换到浅色模式' : '切换到深色模式'"
          @click="toggleTheme"
        >
          <Icon :name="isDark ? 'sun' : 'moon'" size="xs" />
          <span class="ssxz-sidebar-text">{{ isDark ? '浅色模式' : '深色模式' }}</span>
        </button>
      </div>
    </aside>

    <main class="ssxz-app-content min-h-screen">
      <header class="ssxz-app-header sticky top-0 z-20 flex h-16 items-center border-b px-4 backdrop-blur-xl sm:px-6">
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
            <div v-if="authStore.isAuthenticated" class="relative">
              <div class="ssxz-account-cluster">
                <span class="ssxz-balance-pill">余额 ${{ userBalance }}</span>
                <button type="button" class="ssxz-user-button" @click="userMenuOpen = !userMenuOpen">
                <span class="flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold" style="background: linear-gradient(135deg, var(--ssxz-primary), var(--ssxz-accent)); color: white">{{ userInitial }}</span>
                <span class="hidden max-w-32 truncate sm:inline">{{ userLabel }}</span>
                <Icon name="chevronDown" size="xs" />
                </button>
              </div>
              <div v-if="userMenuOpen" class="ssxz-user-menu">
                <div class="ssxz-menu-summary">
                  <strong>{{ userLabel }}</strong>
                  <span>余额 ${{ userBalance }}</span>
                </div>
                <button type="button" class="ssxz-menu-link text-red-600 dark:text-red-300" @click="logout">退出登录</button>
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import type { ChatConversation } from '@/api/chatWorkspace'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type IconName = InstanceType<typeof Icon>['$props']['name']

withDefaults(defineProps<{
  title: string
  subtitle: string
  eyebrow?: string
  icon?: IconName
  historyItems?: ChatConversation[]
  activeConversationId?: number | null
  historyLoading?: boolean
}>(), {
  eyebrow: 'SSXZ AI 工作台',
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
const appStore = useAppStore()
const authStore = useAuthStore()
const userMenuOpen = ref(false)
const SIDEBAR_COLLAPSED_KEY = 'ssxz.app.sidebar.collapsed'
const sidebarCollapsed = ref(readSidebarCollapsed())
const mobileNavOpen = ref(false)
const isDesktopViewport = ref(false)
const isDark = ref(document.documentElement.classList.contains('dark'))
let desktopMediaQuery: MediaQueryList | null = null

const navItems: Array<{ label: string; to: string; icon: IconName }> = [
  { label: '新对话', to: '/app/chat', icon: 'chat' },
  { label: 'AI 作图', to: '/app/image', icon: 'sparkles' }
]

const utilityItems: Array<{ label: string; to: string; icon: IconName }> = [
  { label: '用量中心', to: '/app/usage', icon: 'chartBar' },
  { label: '充值', to: '/app/purchase', icon: 'creditCard' },
  { label: '订单记录', to: '/app/orders', icon: 'clipboard' },
  { label: 'API Key / 第三方接入', to: '/app/keys', icon: 'key' },
  { label: '账户设置', to: '/app/profile', icon: 'userCircle' }
]


const userLabel = computed(() => authStore.user?.username || authStore.user?.email?.split('@')[0] || '账户')
const userInitial = computed(() => userLabel.value.slice(0, 1).toUpperCase())
const userBalance = computed(() => formatMoney(authStore.user?.balance || 0))
const navToggleLabel = computed(() => {
  if (!isDesktopViewport.value) return mobileNavOpen.value ? '关闭导航' : '打开导航'
  return sidebarCollapsed.value ? '展开侧边栏' : '收起侧边栏'
})
const navToggleExpanded = computed(() => !isDesktopViewport.value ? mobileNavOpen.value : !sidebarCollapsed.value)

function isActive(path: string) {
  const normalizedPath = path.split('?')[0]
  if (normalizedPath === '/app') return route.path === '/app' || route.path === '/app/chat'
  return route.path === normalizedPath
}

function handleRouteNav(to: string) {
  closeMobileNav()
  if (route.path !== to) router.push(to)
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

function handleHistorySelect(id: number) {
  emit('select-conversation', id)
  closeMobileNav()
}

function formatMoney(value: number) {
  return Number(value || 0).toFixed(2)
}

function readSidebarCollapsed() {
  try {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
  } catch {
    return false
  }
}

function setSidebarCollapsed(value: boolean) {
  sidebarCollapsed.value = value
  try {
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, value ? 'true' : 'false')
  } catch {}
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
    return
  }
  isDesktopViewport.value = window.matchMedia('(min-width: 1024px)').matches
  if (isDesktopViewport.value) closeMobileNav()
}

async function logout() {
  await authStore.logout()
  userMenuOpen.value = false
  appStore.showSuccess('已退出登录')
  router.push('/app')
}

function initTheme() {
  const saved = localStorage.getItem('theme')
  if (saved) isDark.value = saved === 'dark'
  document.documentElement.classList.toggle('dark', isDark.value)
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  initTheme()
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    desktopMediaQuery = window.matchMedia('(min-width: 1024px)')
    syncViewportMode()
    desktopMediaQuery.addEventListener('change', syncViewportMode)
  } else {
    syncViewportMode()
  }
})

onBeforeUnmount(() => {
  desktopMediaQuery?.removeEventListener('change', syncViewportMode)
})
</script>

<style scoped>
.ssxz-app-shell {
  --ssxz-bg: #f6f8f5;
  --ssxz-surface: #fffdfa;
  --ssxz-surface-raised: #ffffff;
  --ssxz-surface-muted: #eef7f3;
  --ssxz-surface-subtle: #f8faf6;
  --ssxz-border: #dfe8e3;
  --ssxz-border-strong: #1fb6a6;
  --ssxz-text: #111827;
  --ssxz-text-primary: #111827;
  --ssxz-text-secondary: #34423d;
  --ssxz-text-muted: #6b7b75;
  --ssxz-body: #34423d;
  --ssxz-subtle: #6b7b75;
  --ssxz-primary: #0f9f93;
  --ssxz-accent: #64d2b6;
  --ssxz-action: #0f9f93;
  --ssxz-action-soft: #e1f7ef;
  --ssxz-action-text: #ffffff;
  --ssxz-active: #eafaf4;
  --ssxz-surface-elevated: #ffffff;
  --ssxz-input: #fffefb;
  --ssxz-focus: #0f9f93;
  --ssxz-focus-ring: rgb(15 159 147 / 0.16);
  --ssxz-danger: #dc2626;
  --ssxz-disabled: #d7e2dc;
  --ssxz-canvas: #f7f3ec;
  --ssxz-glow-subtle: rgb(100 210 182 / 0.16);
  --ssxz-shadow-sm: 0 8px 24px rgb(15 23 42 / 0.06);
  --ssxz-shadow: 0 18px 52px rgb(15 23 42 / 0.08);
  --ssxz-shadow-lg: 0 18px 46px rgb(15 159 147 / 0.22);
  min-height: 100vh;
  overflow-x: hidden;
  background:
    linear-gradient(90deg, rgb(15 159 147 / 0.035) 1px, transparent 1px),
    linear-gradient(180deg, rgb(15 159 147 / 0.028) 1px, transparent 1px),
    radial-gradient(circle at 70% 18%, rgb(100 210 182 / 0.18), transparent 28rem),
    linear-gradient(135deg, #fbfaf6 0%, var(--ssxz-bg) 48%, #eef6f2 100%);
  background-size: 88px 88px, 88px 88px, auto, auto;
  color: var(--ssxz-text);
}

.dark .ssxz-app-shell {
  --ssxz-bg: #0f1110;
  --ssxz-surface: #1c1d1b;
  --ssxz-surface-raised: #202320;
  --ssxz-surface-muted: #181b18;
  --ssxz-surface-subtle: #171a18;
  --ssxz-border: #303831;
  --ssxz-border-strong: #2fd4bf;
  --ssxz-text: #f7fbf8;
  --ssxz-text-primary: #f7fbf8;
  --ssxz-text-secondary: #d8e2dd;
  --ssxz-text-muted: #93a49d;
  --ssxz-body: #d8e2dd;
  --ssxz-subtle: #93a49d;
  --ssxz-primary: #25c7b5;
  --ssxz-accent: #5ee0bd;
  --ssxz-action: #25c7b5;
  --ssxz-action-soft: #123630;
  --ssxz-action-text: #061312;
  --ssxz-active: #123c36;
  --ssxz-surface-elevated: #232722;
  --ssxz-input: #121512;
  --ssxz-focus: #25c7b5;
  --ssxz-focus-ring: rgb(37 199 181 / 0.18);
  --ssxz-danger: #f87171;
  --ssxz-disabled: #26302c;
  --ssxz-canvas: #171a18;
  --ssxz-glow-subtle: rgb(37 199 181 / 0.12);
  --ssxz-shadow-sm: 0 8px 28px rgb(0 0 0 / 0.18);
  --ssxz-shadow: 0 18px 58px rgb(0 0 0 / 0.25);
  --ssxz-shadow-lg: 0 18px 52px rgb(37 199 181 / 0.18);
  background:
    linear-gradient(90deg, rgb(255 255 255 / 0.025) 1px, transparent 1px),
    linear-gradient(180deg, rgb(255 255 255 / 0.022) 1px, transparent 1px),
    radial-gradient(circle at 68% 20%, rgb(37 199 181 / 0.12), transparent 28rem),
    linear-gradient(135deg, #141614 0%, var(--ssxz-bg) 52%, #111817 100%);
  background-size: 88px 88px, 88px 88px, auto, auto;
}

.ssxz-app-backdrop {
  pointer-events: none;
  position: fixed;
  inset: 0;
  z-index: 0;
  background:
    radial-gradient(circle at 78% 18%, rgb(15 159 147 / 0.13), transparent 20rem),
    radial-gradient(circle at 38% 58%, rgb(100 210 182 / 0.10), transparent 24rem);
}

.ssxz-app-sidebar {
  border-color: var(--ssxz-border);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 88%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
  color: var(--ssxz-body);
}

.ssxz-brand-link {
  display: inline-flex;
  min-height: 3rem;
  width: 100%;
  align-items: center;
  gap: 0.72rem;
  border-radius: 1rem;
  color: var(--ssxz-text);
  padding: 0.28rem;
}

.ssxz-brand-link:hover {
  background: color-mix(in srgb, var(--ssxz-primary) 8%, transparent);
}

.ssxz-brand-mark {
  display: grid;
  width: 2.15rem;
  height: 2.15rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 0.75rem;
  background: linear-gradient(135deg, #111827, #0f9f93);
  color: white;
  font-weight: 900;
}

.dark .ssxz-brand-mark {
  background: linear-gradient(135deg, #f8fafc, #25c7b5);
  color: #071211;
}

.ssxz-brand-copy {
  display: grid;
  gap: 0.05rem;
}

.ssxz-brand-title {
  font-size: 0.95rem;
  font-weight: 850;
}

.ssxz-brand-subtitle {
  color: var(--ssxz-subtle);
  font-size: 0.74rem;
  font-weight: 700;
}

.ssxz-app-content {
  position: relative;
  z-index: 1;
}

.ssxz-app-header {
  border-color: var(--ssxz-border);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 82%, transparent);
}

.ssxz-app-main {
  margin-inline: auto;
  width: min(100%, 96rem);
  padding: 1rem;
}

.ssxz-page-heading {
  margin-bottom: 1rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1.25rem;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 90%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
  padding: 1.1rem;
}

.ssxz-page-heading h2 {
  color: var(--ssxz-text);
}

.ssxz-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-primary) 12%, transparent);
  color: var(--ssxz-primary);
  font-size: 0.76rem;
  font-weight: 800;
  padding: 0.28rem 0.58rem;
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

.ssxz-app-sidebar {
  display: none;
}

.ssxz-mobile-sidebar-scrim {
  position: fixed;
  inset: 0;
  z-index: 25;
  border: 0;
  background: rgb(2 6 23 / 0.58);
  backdrop-filter: blur(2px);
}

.ssxz-mobile-nav-open .ssxz-app-sidebar {
  display: block;
  box-shadow: 18px 0 50px rgb(2 6 23 / 0.35);
}

@media (min-width: 1024px) {
  .ssxz-app-sidebar {
    display: block;
  }

  .ssxz-app-content {
    margin-left: 15rem;
  }

  .ssxz-sidebar-collapsed .ssxz-app-content {
    margin-left: 5rem;
  }

  .ssxz-sidebar-collapsed .ssxz-app-sidebar {
    width: 5rem;
  }

  .ssxz-sidebar-collapsed .ssxz-sidebar-text {
    display: none;
  }
}

.ssxz-nav-item,
.ssxz-theme-toggle {
  display: inline-flex;
  min-height: 2.55rem;
  width: 100%;
  align-items: center;
  gap: 0.65rem;
  border-radius: 0.75rem;
  color: var(--ssxz-body);
  font-size: 0.92rem;
  line-height: 1.3;
  padding: 0.55rem 0.7rem;
}

.ssxz-nav-item:hover,
.ssxz-theme-toggle:hover,
.ssxz-nav-item.is-active {
  background: color-mix(in srgb, var(--ssxz-primary) 10%, transparent);
  color: var(--ssxz-text);
}

.ssxz-nav-item svg,
.ssxz-theme-toggle svg {
  flex: 0 0 auto;
}

.ssxz-primary-nav {
  display: grid;
  gap: 0.45rem;
}

.ssxz-secondary-nav {
  display: grid;
  gap: 0.28rem;
  margin-top: 0.9rem;
}

.ssxz-utility-item {
  border: 0;
  text-align: left;
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
  max-height: min(32rem, calc(100vh - 23rem));
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
}

.ssxz-history-item .ssxz-sidebar-text {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-height: 1.35;
  text-align: left;
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
  .ssxz-balance-pill {
    display: none;
  }
}
</style>
