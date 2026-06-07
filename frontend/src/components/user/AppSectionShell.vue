<template>
  <div class="ssxz-app-shell" data-app-shell-boundary="section" :class="{ 'ssxz-sidebar-collapsed': sidebarCollapsed }">
    <div class="ssxz-app-backdrop" aria-hidden="true" />
    <aside class="ssxz-app-sidebar fixed inset-y-0 left-0 z-30 hidden w-60 border-r px-3 py-4 backdrop-blur-xl lg:block">
      <RouterLink to="/app" class="ssxz-brand-link mb-6" title="返回工作台首页" aria-label="返回工作台首页">
        <span class="ssxz-brand-mark">S</span>
        <span class="ssxz-brand-copy ssxz-sidebar-text">
          <span class="ssxz-brand-title">SSXZ AI</span>
          <span class="ssxz-brand-subtitle">对话工作台</span>
        </span>
      </RouterLink>

      <nav class="ssxz-primary-nav" aria-label="主导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="ssxz-nav-item ssxz-new-chat"
          :class="{ 'is-active': isActive(item.to) }"
          :title="item.label"
          :aria-label="item.label"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <nav class="ssxz-secondary-nav" aria-label="工作台入口">
        <button
          v-for="item in utilityItems"
          :key="item.id"
          type="button"
          class="ssxz-nav-item ssxz-utility-item"
          :class="{ 'is-active': activeUtility === item.id }"
          :title="item.label"
          :aria-label="item.label"
          :aria-expanded="activeUtility === item.id"
          @click="toggleUtilityPanel(item.id)"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.label }}</span>
        </button>
      </nav>

      <section v-if="activeUtilityContent" class="ssxz-utility-panel" aria-live="polite">
        <div class="ssxz-utility-title ssxz-sidebar-text">{{ activeUtilityContent.label }}</div>
        <p class="ssxz-sidebar-text">{{ activeUtilityContent.description }}</p>
      </section>

      <section class="ssxz-usage-panel" aria-label="用量信息">
        <div class="ssxz-section-label ssxz-sidebar-text">用量信息</div>
        <div class="ssxz-usage-row">
          <span class="ssxz-sidebar-text">当前余额</span>
          <strong>${{ userBalance }}</strong>
        </div>
        <p class="ssxz-usage-note ssxz-sidebar-text">暂无用量数据</p>
      </section>

      <section class="ssxz-history" aria-label="历史会话">
        <div class="ssxz-section-label ssxz-sidebar-text">历史会话</div>
        <RouterLink
          v-for="item in historyItems"
          :key="item.label"
          :to="item.to"
          class="ssxz-nav-item"
          :title="item.label"
          :aria-label="item.label"
        >
          <Icon :name="item.icon" size="sm" />
          <span class="ssxz-sidebar-text">{{ item.label }}</span>
        </RouterLink>
        <p v-if="historyItems.length === 0" class="ssxz-empty-history ssxz-sidebar-text">
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
            :aria-label="sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'"
            :title="sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'"
            :aria-expanded="!sidebarCollapsed"
            @click="toggleSidebarCollapsed"
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
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type IconName = InstanceType<typeof Icon>['$props']['name']
type UtilityId = 'developer' | 'billing' | 'account'

withDefaults(defineProps<{
  title: string
  subtitle: string
  eyebrow?: string
  icon?: IconName
}>(), {
  eyebrow: 'SSXZ AI 对话工作台',
  icon: 'sparkles'
})

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const userMenuOpen = ref(false)
const activeUtility = ref<UtilityId | null>(null)
const SIDEBAR_COLLAPSED_KEY = 'ssxz.app.sidebar.collapsed'
const sidebarCollapsed = ref(readSidebarCollapsed())
const isDark = ref(document.documentElement.classList.contains('dark'))

const navItems: Array<{ label: string; to: string; icon: IconName }> = [
  { label: '新对话', to: '/app?new=1', icon: 'chat' }
]

const utilityItems: Array<{ id: UtilityId; label: string; icon: IconName; description: string }> = [
  {
    id: 'developer',
    label: '开发者 API',
    icon: 'terminal',
    description: '开发接入能力将在工作台内呈现；你也可以直接在对话里描述接入需求。'
  },
  {
    id: 'billing',
    label: '余额充值',
    icon: 'creditCard',
    description: '当前余额显示在右上角，充值能力将以工作台形态接入。'
  },
  {
    id: 'account',
    label: '账户设置',
    icon: 'userCircle',
    description: '账户信息在右上角展开，可在此查看余额与退出登录。'
  }
]

const historyItems: Array<{ label: string; to: string; icon: IconName }> = []

const userLabel = computed(() => authStore.user?.username || authStore.user?.email?.split('@')[0] || '账户')
const userInitial = computed(() => userLabel.value.slice(0, 1).toUpperCase())
const userBalance = computed(() => formatMoney(authStore.user?.balance || 0))
const activeUtilityContent = computed(() => utilityItems.find((item) => item.id === activeUtility.value) ?? null)

function isActive(path: string) {
  const normalizedPath = path.split('?')[0]
  if (normalizedPath === '/app') return route.path === '/app' || route.path === '/app/chat' || route.path === '/app/image'
  return route.path === normalizedPath
}

function toggleUtilityPanel(id: UtilityId) {
  activeUtility.value = activeUtility.value === id ? null : id
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

<style scoped>
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
}

.ssxz-balance-pill {
  display: inline-flex;
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
