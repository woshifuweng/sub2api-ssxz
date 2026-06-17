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
          @click="emit('select-conversation', item.id)"
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

        <section v-if="activeUtilityContent" class="ssxz-workspace-utility-center" aria-live="polite">
          <div class="ssxz-utility-center-heading">
            <Icon :name="activeUtilityContent.icon" size="sm" />
            <div>
              <h3>{{ activeUtilityContent.label }}</h3>
              <p>{{ activeUtilityContent.description }}</p>
            </div>
          </div>
          <dl v-if="activeUtility === 'usage'" class="ssxz-usage-details">
            <div>
              <dt>剩余余额</dt>
              <dd>${{ userBalance }}</dd>
            </div>
            <div>
              <dt>用量记录</dt>
              <dd>暂无用量记录</dd>
            </div>
          </dl>
          <div v-if="activeUtility === 'developer'" class="ssxz-utility-actions">
            <RouterLink to="/keys" class="ssxz-utility-action">
              打开 API Key / 第三方客户端接入
            </RouterLink>
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
import type { ChatConversation } from '@/api/chatWorkspace'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type IconName = InstanceType<typeof Icon>['$props']['name']
type UtilityId = 'account' | 'developer' | 'usage'

withDefaults(defineProps<{
  title: string
  subtitle: string
  eyebrow?: string
  icon?: IconName
  historyItems?: ChatConversation[]
  activeConversationId?: number | null
  historyLoading?: boolean
}>(), {
  eyebrow: 'SSXZ AI 对话工作台',
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
const activeUtility = ref<UtilityId | null>(null)
const SIDEBAR_COLLAPSED_KEY = 'ssxz.app.sidebar.collapsed'
const sidebarCollapsed = ref(readSidebarCollapsed())
const isDark = ref(document.documentElement.classList.contains('dark'))

const navItems: Array<{ label: string; to: string; icon: IconName }> = [
  { label: '新对话', to: '/app?new=1', icon: 'chat' }
]

const utilityItems: Array<{ id: UtilityId; label: string; icon: IconName; description: string }> = [
  {
    id: 'usage',
    label: '用量中心',
    icon: 'chartBar',
    description: '查看消耗与余额。当前仅展示已确认的账户余额，暂无用量记录时不会编造数据。'
  },
  {
    id: 'developer',
    label: 'API Key / 第三方接入',
    icon: 'key',
    description: '熟练用户可以在这里创建自己的 API Key，并复制 Base URL 接入 CC Switch、Cherry Studio、Chatbox 等第三方客户端。'
  },
  {
    id: 'account',
    label: '账户设置',
    icon: 'userCircle',
    description: '账户信息在右上角展开，可在此查看余额与退出登录。'
  }
]


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

function handlePrimaryNav(to: string) {
  activeUtility.value = null
  if (to === '/app?new=1') {
    emit('new-chat')
    if (route.path !== '/app') router.push('/app')
    return
  }
  router.push(to)
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
@media (min-width: 1024px) {
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

.ssxz-workspace-utility-center {
  display: grid;
  gap: 0.9rem;
  margin: 0 0 1rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1.15rem;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 86%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
  padding: 1rem;
}

.ssxz-utility-center-heading {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.ssxz-utility-center-heading svg {
  margin-top: 0.15rem;
  color: var(--ssxz-primary);
}

.ssxz-utility-center-heading h3 {
  color: var(--ssxz-text);
  font-size: 0.98rem;
  font-weight: 780;
  margin: 0;
}

.ssxz-utility-center-heading p {
  color: var(--ssxz-body);
  font-size: 0.84rem;
  line-height: 1.6;
  margin: 0.2rem 0 0;
}

.ssxz-usage-details {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.7rem;
  margin: 0;
}

.ssxz-usage-details div {
  border: 1px solid var(--ssxz-border);
  border-radius: 0.95rem;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 72%, transparent);
  padding: 0.8rem;
}

.ssxz-usage-details dt {
  color: var(--ssxz-subtle);
  font-size: 0.75rem;
  font-weight: 760;
}

.ssxz-usage-details dd {
  color: var(--ssxz-text);
  font-size: 0.94rem;
  font-weight: 780;
  margin: 0.25rem 0 0;
}

.ssxz-utility-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
}

.ssxz-utility-action {
  display: inline-flex;
  align-items: center;
  border: 1px solid color-mix(in srgb, var(--ssxz-primary) 38%, var(--ssxz-border));
  border-radius: 0.85rem;
  background: color-mix(in srgb, var(--ssxz-primary) 10%, transparent);
  color: var(--ssxz-text);
  font-size: 0.84rem;
  font-weight: 760;
  min-height: 2.35rem;
  padding: 0.55rem 0.78rem;
}

.ssxz-utility-action:hover {
  background: color-mix(in srgb, var(--ssxz-primary) 16%, transparent);
}

@media (max-width: 640px) {
  .ssxz-balance-pill {
    display: none;
  }

  .ssxz-usage-details {
    grid-template-columns: 1fr;
  }
}
</style>
