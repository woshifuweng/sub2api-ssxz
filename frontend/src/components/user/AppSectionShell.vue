<template>
  <div class="ssxz-app-shell" data-app-shell-boundary="section" :class="{ 'ssxz-sidebar-collapsed': sidebarCollapsed }">
    <div class="ssxz-app-backdrop" aria-hidden="true" />
    <aside class="ssxz-app-sidebar fixed inset-y-0 left-0 z-30 hidden w-60 border-r px-3 py-4 backdrop-blur-xl lg:block">
      <RouterLink to="/app" class="ssxz-brand-link mb-7" title="返回工作台首页" aria-label="返回工作台首页">
        <span class="ssxz-brand-mark">S</span>
        <span class="ssxz-brand-copy ssxz-sidebar-text">
          <span class="ssxz-brand-title">SSXZ AI</span>
          <span class="ssxz-brand-subtitle">工作台</span>
        </span>
      </RouterLink>

      <nav class="space-y-1">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="ssxz-nav-item"
          :class="{ 'is-active': isActive(item.to) }"
          :title="item.label"
          :aria-label="item.label"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="absolute bottom-4 left-3 right-3 space-y-2">
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
            :aria-label="sidebarCollapsed ? '展开侧边栏' : '折叠侧边栏'"
            :title="sidebarCollapsed ? '展开侧边栏' : '折叠侧边栏'"
            :aria-expanded="!sidebarCollapsed"
            @click="toggleSidebarCollapsed"
          >
            <Icon name="menu" size="sm" />
          </button>
          <div class="flex items-center gap-2">
            <RouterLink v-if="authStore.isAuthenticated" to="/app/billing" class="ssxz-balance-pill hidden sm:inline-flex">
              余额 ${{ balanceText }}
            </RouterLink>
            <div v-if="authStore.isAuthenticated" class="relative">
              <button type="button" class="ssxz-user-button" @click="userMenuOpen = !userMenuOpen">
                <span class="flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold" style="background: linear-gradient(135deg, var(--ssxz-primary), var(--ssxz-accent)); color: white">{{ userInitial }}</span>
                <span class="hidden max-w-32 truncate sm:inline">{{ userLabel }}</span>
                <Icon name="chevronDown" size="xs" />
              </button>
              <div v-if="userMenuOpen" class="ssxz-user-menu">
                <RouterLink class="ssxz-menu-link" to="/app/account" @click="userMenuOpen = false">账户设置</RouterLink>
                <RouterLink class="ssxz-menu-link" to="/app/billing" @click="userMenuOpen = false">余额与账单</RouterLink>
                <RouterLink v-if="isAdmin" class="ssxz-menu-link" to="/admin/dashboard" @click="userMenuOpen = false">管理员后台</RouterLink>
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
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type IconName = InstanceType<typeof Icon>['$props']['name']

withDefaults(defineProps<{
  title: string
  subtitle: string
  eyebrow?: string
  icon?: IconName
}>(), {
  eyebrow: 'SSXZ AI 工作台',
  icon: 'sparkles'
})

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const userMenuOpen = ref(false)
const SIDEBAR_COLLAPSED_KEY = 'ssxz.app.sidebar.collapsed'
const sidebarCollapsed = ref(readSidebarCollapsed())
const isDark = ref(document.documentElement.classList.contains('dark'))

const navItems: Array<{ label: string; to: string; icon: IconName }> = [
  { label: '新对话', to: '/app?new=1', icon: 'chat' },
  { label: 'AI 作图', to: '/app/image', icon: 'sparkles' },
  { label: '开发者 API', to: '/app/developer', icon: 'terminal' },
  { label: '余额充值', to: '/app/billing', icon: 'creditCard' },
  { label: '账户设置', to: '/app/account', icon: 'userCircle' }
]

const balanceText = computed(() => (authStore.user?.balance ?? 0).toFixed(2))
const userLabel = computed(() => authStore.user?.username || authStore.user?.email?.split('@')[0] || '账户')
const userInitial = computed(() => userLabel.value.slice(0, 1).toUpperCase())
const isAdmin = computed(() => authStore.isAdmin)

function isActive(path: string) {
  const normalizedPath = path.split('?')[0]
  if (normalizedPath === '/app') return route.path === '/app' || route.path === '/app/chat'
  return route.path === normalizedPath
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
})
</script>
