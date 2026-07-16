<template>
  <AppSectionShell
    :class="['dashboard-shell', `dashboard-shell--${theme}`]"
    title="仪表盘"
    subtitle="管理 API Key、用量余额、充值订阅、订单记录和通道状态。"
    eyebrow="服务控制台"
    icon="home"
  >
    <FoundationProvider :theme="theme" class="dashboard-foundation">
      <div class="dashboard-workspace">
        <div v-if="loading" class="dashboard-loading">
          <LoadingSpinner />
        </div>

        <section
          v-else-if="statsLoadError"
          data-testid="dashboard-load-error"
          class="dashboard-error"
        >
          <span class="dashboard-error__icon" aria-hidden="true">
            <CircleAlert />
          </span>
          <div>
            <h1>仪表盘数据暂时无法加载</h1>
            <p>当前没有展示占位数据。API Key、用量、订单记录和通道状态仍可从左侧菜单进入。</p>
          </div>
          <button
            type="button"
            data-testid="dashboard-retry"
            class="dashboard-link-button dashboard-link-button--primary"
            :disabled="loading"
            @click="loadStats"
          >
            重试
          </button>
        </section>

        <template v-else-if="stats">
          <section class="dashboard-command-panel">
            <div class="dashboard-command-panel__identity">
              <span class="dashboard-command-panel__icon" aria-hidden="true">
                <Activity />
              </span>
              <div>
                <p class="dashboard-kicker">账户工作台</p>
                <h1>{{ userEmail }}</h1>
                <p>从这里查看调用、消费和服务状态，并进入各项管理功能。</p>
              </div>
            </div>

            <div class="dashboard-command-panel__actions">
              <RouterLink to="/app/keys" class="dashboard-link-button dashboard-link-button--primary">
                <KeyRound />
                创建 API Key
              </RouterLink>
              <RouterLink to="/app/usage" class="dashboard-link-button dashboard-link-button--outline">
                <ChartSpline />
                查看用量
              </RouterLink>
              <RouterLink
                v-if="channelMonitorEnabled"
                to="/app/channel-status"
                class="dashboard-link-button dashboard-link-button--ghost"
              >
                <Server />
                通道状态
              </RouterLink>
            </div>
          </section>

          <section class="dashboard-section" aria-labelledby="dashboard-metrics-title">
            <div class="dashboard-section__heading">
              <div>
                <p class="dashboard-kicker">实时概览</p>
                <h2 id="dashboard-metrics-title">账户与用量</h2>
              </div>
              <span class="dashboard-status-badge">
                <span aria-hidden="true"></span>
                真实统计
              </span>
            </div>
            <UserDashboardStats
              :stats="stats"
              :balance="balance"
              :is-simple="authStore.isSimpleMode"
              :trend="trendData"
              :today-trend="todayTrendData"
            />
          </section>

          <UserDashboardCharts
            v-model:startDate="startDate"
            v-model:endDate="endDate"
            v-model:granularity="granularity"
            :loading="loadingCharts"
            :trend="trendData"
            :models="modelStats"
            :theme="theme"
            @dateRangeChange="loadCharts"
            @granularityChange="loadCharts"
          />

          <div class="dashboard-detail-grid">
            <UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" />
            <UserDashboardQuickActions />
          </div>

          <section class="dashboard-panel" aria-labelledby="dashboard-entry-title">
            <div class="dashboard-panel__heading">
              <div>
                <p class="dashboard-kicker">服务入口</p>
                <h2 id="dashboard-entry-title">管理常用功能</h2>
              </div>
              <p>所有原有入口均保留，按任务直接进入。</p>
            </div>
            <div class="dashboard-entry-list">
              <RouterLink
                v-for="entry in productEntries"
                :key="entry.to"
                :to="entry.to"
                class="dashboard-entry"
              >
                <span class="dashboard-entry__icon" aria-hidden="true">
                  <component :is="entry.icon" />
                </span>
                <span class="dashboard-entry__copy">
                  <span class="dashboard-entry__title-row">
                    <strong>{{ entry.title }}</strong>
                    <span>{{ entry.badge }}</span>
                  </span>
                  <small>{{ entry.description }}</small>
                </span>
                <span class="dashboard-entry__action">
                  {{ entry.action }}
                  <ArrowUpRight />
                </span>
              </RouterLink>
            </div>
          </section>

          <section class="dashboard-panel" aria-labelledby="dashboard-onboarding-title">
            <div class="dashboard-panel__heading">
              <div>
                <p class="dashboard-kicker">接入顺序</p>
                <h2 id="dashboard-onboarding-title">三步完成首次调用</h2>
              </div>
              <p>创建 Key、核对用量，再处理额度与订单。</p>
            </div>
            <ol class="dashboard-onboarding">
              <li v-for="step in onboardingSteps" :key="step.title">
                <span class="dashboard-onboarding__index">{{ step.index }}</span>
                <div>
                  <h3>{{ step.title }}</h3>
                  <p>{{ step.description }}</p>
                  <RouterLink :to="step.to">
                    {{ step.action }}
                    <ArrowUpRight />
                  </RouterLink>
                </div>
              </li>
            </ol>
          </section>
        </template>
      </div>
    </FoundationProvider>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type Component } from 'vue'
import {
  Activity,
  ArrowUpRight,
  Calculator,
  ChartSpline,
  CircleAlert,
  CreditCard,
  Gift,
  KeyRound,
  ReceiptText,
  Server,
  Users
} from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { FoundationProvider } from '@/components/foundation'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { ModelStat, TrendDataPoint, UsageLog } from '@/types'

interface ProductEntry {
  to: string
  icon: Component
  badge: string
  title: string
  description: string
  action: string
}

function getInitialTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'dark'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const authStore = useAuthStore()
const appStore = useAppStore()
const user = computed(() => authStore.user)
const balance = computed(() => user.value?.balance || 0)
const userEmail = computed(() => user.value?.email || '当前用户')
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
const channelMonitorEnabled = computed(() => !!appStore.cachedPublicSettings?.channel_monitor_enabled)
const theme = ref<'light' | 'dark'>(getInitialTheme())
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const statsLoadError = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const todayTrendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
let themeObserver: MutationObserver | null = null

const baseProductEntries: ProductEntry[] = [
  {
    to: '/app/keys',
    icon: KeyRound,
    badge: '主入口',
    title: 'API 密钥',
    description: '创建和管理自己的 API Key，用于 Cherry Studio、Chatbox、CC Switch 等常用客户端。',
    action: '管理 Key'
  },
  {
    to: '/app/usage',
    icon: ChartSpline,
    badge: '用量',
    title: '使用记录',
    description: '查看最近请求、模型消耗和余额变化，确认每次调用是否成功计费。',
    action: '查看明细'
  },
  {
    to: '/app/channel-status',
    icon: Server,
    badge: '通道',
    title: '通道状态',
    description: '查看当前可用通道和模型状态，遇到失败时判断额度、模型或线路问题。',
    action: '查看状态'
  },
  {
    to: '/app/available-channels',
    icon: Calculator,
    badge: '价格',
    title: '模型价格',
    description: '查看当前账号可用模型和价格范围，最终可用性以后端配置和当前 Key 分组为准。',
    action: '查看价格'
  },
  {
    to: '/app/purchase',
    icon: CreditCard,
    badge: '额度',
    title: '充值 / 订阅',
    description: '查看可用充值方式和订阅入口，额度变化以使用记录和订单记录为准。',
    action: '补充额度'
  },
  {
    to: '/app/orders',
    icon: ReceiptText,
    badge: '记录',
    title: '我的订单',
    description: '查看充值、订阅和兑换后的账户记录与到账状态。',
    action: '查看订单'
  },
  {
    to: '/app/redeem',
    icon: Gift,
    badge: '兑换',
    title: '兑换码',
    description: '已有兑换码时可直接兑换到账户额度或试用权益。',
    action: '去兑换'
  },
  {
    to: '/app/affiliate',
    icon: Users,
    badge: '返利',
    title: '邀请返利',
    description: '查看邀请码、邀请用户和返利记录；具体规则以后台策略为准。',
    action: '查看返利'
  }
]

const productEntries = computed(() =>
  baseProductEntries.filter(
    (entry) =>
      (entry.to !== '/app/channel-status' || channelMonitorEnabled.value) &&
      (entry.to !== '/app/purchase' || paymentEnabled.value)
  )
)

const onboardingSteps = computed(() => [
  {
    index: '01',
    title: '先创建 API Key',
    description: '把 Key 配到 Cherry Studio、Chatbox、CC Switch 等客户端，优先完成首次可用连接。',
    to: '/app/keys',
    action: '管理 Key'
  },
  {
    index: '02',
    title: '再查看用量和余额',
    description: '每次调用后回到使用记录查看模型、用量、余额变化和扣费结果。',
    to: '/app/usage',
    action: '查看用量'
  },
  paymentEnabled.value
    ? {
        index: '03',
        title: '最后核对额度和订单记录',
        description: '余额不足时进入充值 / 订阅页，完成后回查订单记录和余额到账情况。',
        to: '/app/purchase',
        action: '补充额度'
      }
    : {
        index: '03',
        title: '最后使用兑换码或核对订单记录',
        description: '当前账号没有在线充值入口时，可先用兑换码补充账户额度；账户变化以使用记录和订单记录为准。',
        to: '/app/redeem',
        action: '去兑换'
      }
])

const formatLD = (date: Date) => [
  date.getFullYear(),
  String(date.getMonth() + 1).padStart(2, '0'),
  String(date.getDate()).padStart(2, '0')
].join('-')
const startDate = ref(formatLD(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatLD(new Date()))
const granularity = ref('day')
const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone

function syncTheme(): void {
  theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const loadStats = async () => {
  loading.value = true
  statsLoadError.value = false
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    stats.value = null
    statsLoadError.value = true
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const response = await Promise.all([
      usageAPI.getDashboardTrend({
        start_date: startDate.value,
        end_date: endDate.value,
        granularity: granularity.value as 'day' | 'hour',
        timezone: browserTimezone
      }),
      usageAPI.getDashboardModels({
        start_date: startDate.value,
        end_date: endDate.value
      })
    ])
    trendData.value = response[0].trend || []
    modelStats.value = response[1].models || []
  } catch (error) {
    console.error('Failed to load charts:', error)
  } finally {
    loadingCharts.value = false
  }
}

const loadTodayTrend = async () => {
  const today = formatLD(new Date())
  try {
    const response = await usageAPI.getDashboardTrend({
      start_date: today,
      end_date: today,
      granularity: 'hour',
      timezone: browserTimezone
    })
    todayTrendData.value = response.trend || []
  } catch (error) {
    todayTrendData.value = []
    console.error('Failed to load today dashboard trend:', error)
  }
}

const loadRecent = async () => {
  loadingUsage.value = true
  try {
    const response = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = response.items.slice(0, 5)
  } catch (error) {
    console.error('Failed to load recent usage:', error)
  } finally {
    loadingUsage.value = false
  }
}

onMounted(() => {
  syncTheme()
  themeObserver = new MutationObserver(syncTheme)
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class']
  })
  void loadStats()
  void loadCharts()
  void loadTodayTrend()
  void loadRecent()
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})
</script>

<style scoped>
.dashboard-shell--light {
  --ssxz-bg: #f6f7f8;
  --ssxz-bg-subtle: #f0f2f4;
  --ssxz-surface: #ffffff;
  --ssxz-surface-raised: #f7f8f9;
  --ssxz-surface-muted: #eceff2;
  --ssxz-surface-code: #f3f5f6;
  --ssxz-code-surface: var(--ssxz-surface-code);
  --ssxz-border: #dde1e5;
  --ssxz-border-strong: #c4cad1;
  --ssxz-text: #17191d;
  --ssxz-text-secondary: #3f4650;
  --ssxz-text-muted: #66707c;
  --ssxz-text-subtle: #88929e;
  --ssxz-primary: #4f6882;
  --ssxz-primary-hover: #415a73;
  --ssxz-primary-soft: rgb(79 104 130 / 0.1);
  --ssxz-accent: #556273;
  --ssxz-accent-strong: #374151;
  --ssxz-accent-soft: rgb(85 98 115 / 0.1);
  --ssxz-action-text: #ffffff;
  --ssxz-body: var(--ssxz-text-secondary);
  --ssxz-subtle: var(--ssxz-text-subtle);
  --ssxz-shadow-card: 0 1px 2px rgb(17 24 39 / 0.06);
  --ssxz-shadow-sm: 0 8px 24px rgb(17 24 39 / 0.08);
  --ssxz-shadow: 0 18px 48px rgb(17 24 39 / 0.1);
  --ssxz-focus-ring: 0 0 0 3px rgb(85 98 115 / 0.2);
}

.dashboard-shell--dark {
  --ssxz-bg: #111214;
  --ssxz-bg-subtle: #141619;
  --ssxz-surface: #1b1b1b;
  --ssxz-surface-raised: #22252a;
  --ssxz-surface-muted: #292d32;
  --ssxz-surface-code: #131416;
  --ssxz-code-surface: var(--ssxz-surface-code);
  --ssxz-border: #34383e;
  --ssxz-border-strong: #4a5059;
  --ssxz-text: #f3f4f6;
  --ssxz-text-secondary: #c7cbd1;
  --ssxz-text-muted: #9aa1ab;
  --ssxz-text-subtle: #707985;
  --ssxz-primary: #9db6ce;
  --ssxz-primary-hover: #b3c7db;
  --ssxz-primary-soft: rgb(157 182 206 / 0.12);
  --ssxz-accent: #aab4c2;
  --ssxz-accent-strong: #d1d5db;
  --ssxz-accent-soft: rgb(170 180 194 / 0.1);
  --ssxz-action-text: #15171a;
  --ssxz-body: var(--ssxz-text-secondary);
  --ssxz-subtle: var(--ssxz-text-subtle);
  --ssxz-shadow-card: 0 1px 2px rgb(0 0 0 / 0.28);
  --ssxz-shadow-sm: 0 8px 24px rgb(0 0 0 / 0.24);
  --ssxz-shadow: 0 18px 48px rgb(0 0 0 / 0.34);
  --ssxz-focus-ring: 0 0 0 3px rgb(170 180 194 / 0.2);
}

.dashboard-foundation {
  min-height: 0;
  background: transparent;
}

.dashboard-workspace {
  display: grid;
  gap: 1rem;
}

.dashboard-loading {
  display: flex;
  min-height: 12rem;
  align-items: center;
  justify-content: center;
}

.dashboard-error,
.dashboard-command-panel,
.dashboard-panel {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  box-shadow: 0 1px 2px hsl(var(--shadow));
}

.dashboard-error {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
}

.dashboard-error__icon,
.dashboard-command-panel__icon,
.dashboard-entry__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
}

.dashboard-error__icon,
.dashboard-command-panel__icon {
  width: 2.5rem;
  height: 2.5rem;
}

.dashboard-error__icon {
  color: hsl(var(--warning));
}

.dashboard-error h1,
.dashboard-error p,
.dashboard-command-panel h1,
.dashboard-command-panel p,
.dashboard-section__heading h2,
.dashboard-panel__heading h2,
.dashboard-panel__heading p,
.dashboard-onboarding h3,
.dashboard-onboarding p {
  margin: 0;
}

.dashboard-error h1 {
  font-size: 1rem;
  font-weight: 650;
}

.dashboard-error p {
  margin-top: 0.25rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.8125rem;
  line-height: 1.25rem;
}

.dashboard-command-panel {
  display: grid;
  grid-template-columns: minmax(17rem, 1fr) auto;
  align-items: center;
  gap: 1.25rem;
  padding: 1.25rem;
}

.dashboard-command-panel__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.9rem;
}

.dashboard-command-panel__identity > div {
  min-width: 0;
}

.dashboard-command-panel__icon svg,
.dashboard-error__icon svg,
.dashboard-entry__icon svg,
.dashboard-link-button svg,
.dashboard-entry__action svg,
.dashboard-onboarding a svg {
  width: 1rem;
  height: 1rem;
  flex: 0 0 auto;
  stroke-width: 1.8;
}

.dashboard-command-panel h1 {
  overflow: hidden;
  margin-top: 0.15rem;
  font-size: 1rem;
  font-weight: 650;
  line-height: 1.4rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-command-panel__identity p:last-child {
  margin-top: 0.2rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  line-height: 1.15rem;
}

.dashboard-kicker {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 650;
  line-height: 1rem;
}

.dashboard-command-panel__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.5rem;
}

.dashboard-link-button {
  display: inline-flex;
  min-height: 2.25rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: 1px solid transparent;
  border-radius: var(--radius);
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.1rem;
  white-space: nowrap;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.dashboard-link-button--primary {
  color: hsl(var(--primary-foreground));
  background: hsl(var(--primary));
  border-color: hsl(var(--primary));
}

.dashboard-link-button--outline {
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  border-color: hsl(var(--input));
}

.dashboard-link-button--ghost {
  color: hsl(var(--muted-foreground));
  background: transparent;
}

.dashboard-link-button:hover {
  background: hsl(var(--accent));
  color: hsl(var(--accent-foreground));
}

.dashboard-link-button--primary:hover {
  background: hsl(var(--primary) / 0.88);
  color: hsl(var(--primary-foreground));
}

.dashboard-section {
  display: grid;
  gap: 0.75rem;
}

.dashboard-section__heading,
.dashboard-panel__heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
}

.dashboard-section__heading h2,
.dashboard-panel__heading h2 {
  margin-top: 0.15rem;
  font-size: 1rem;
  font-weight: 650;
  line-height: 1.5rem;
}

.dashboard-status-badge {
  display: inline-flex;
  min-height: 1.5rem;
  align-items: center;
  gap: 0.4rem;
  border: 1px solid hsl(var(--success) / 0.22);
  border-radius: 999px;
  padding: 0.1875rem 0.625rem;
  color: hsl(var(--success));
  background: hsl(var(--success) / 0.1);
  font-size: 0.6875rem;
  font-weight: 600;
}

.dashboard-status-badge span {
  width: 0.38rem;
  height: 0.38rem;
  border-radius: 999px;
  background: currentColor;
}

.dashboard-detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(17rem, 0.85fr);
  gap: 1rem;
}

.dashboard-panel {
  padding: 1.25rem;
}

.dashboard-panel__heading p {
  max-width: 34rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  line-height: 1.15rem;
}

.dashboard-entry-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: 1rem;
  border-top: 1px solid hsl(var(--border));
}

.dashboard-entry {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.75rem;
  min-height: 5.25rem;
  padding: 0.8rem 0.75rem;
  border-bottom: 1px solid hsl(var(--border) / 0.72);
  color: hsl(var(--foreground));
}

.dashboard-entry:nth-child(odd) {
  border-right: 1px solid hsl(var(--border) / 0.72);
}

.dashboard-entry:hover {
  background: hsl(var(--muted) / 0.55);
}

.dashboard-entry__icon {
  width: 2.25rem;
  height: 2.25rem;
}

.dashboard-entry__copy {
  display: grid;
  min-width: 0;
  gap: 0.2rem;
}

.dashboard-entry__title-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.dashboard-entry__title-row strong {
  font-size: 0.8125rem;
  font-weight: 650;
}

.dashboard-entry__title-row > span {
  border: 1px solid hsl(var(--border));
  border-radius: 999px;
  padding: 0.05rem 0.4rem;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted));
  font-size: 0.625rem;
  font-weight: 600;
}

.dashboard-entry small {
  overflow: hidden;
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-entry__action,
.dashboard-onboarding a {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 600;
  white-space: nowrap;
}

.dashboard-onboarding {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  margin: 1rem 0 0;
  padding: 0;
  border-top: 1px solid hsl(var(--border));
  list-style: none;
}

.dashboard-onboarding li {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.75rem;
  padding: 1rem;
  border-right: 1px solid hsl(var(--border));
}

.dashboard-onboarding li:first-child {
  padding-left: 0;
}

.dashboard-onboarding li:last-child {
  padding-right: 0;
  border-right: 0;
}

.dashboard-onboarding__index {
  display: inline-flex;
  width: 1.75rem;
  height: 1.75rem;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted));
  font-size: 0.625rem;
  font-weight: 700;
}

.dashboard-onboarding h3 {
  font-size: 0.8125rem;
  font-weight: 650;
  line-height: 1.2rem;
}

.dashboard-onboarding p {
  min-height: 3rem;
  margin-top: 0.35rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.dashboard-onboarding a {
  margin-top: 0.65rem;
  color: hsl(var(--foreground));
}

@media (max-width: 1280px) {
  .dashboard-command-panel__actions {
    justify-content: flex-end;
  }
}

@media (max-width: 900px) {
  .dashboard-command-panel,
  .dashboard-detail-grid,
  .dashboard-entry-list,
  .dashboard-onboarding {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-entry:nth-child(odd),
  .dashboard-onboarding li {
    border-right: 0;
  }

  .dashboard-onboarding li,
  .dashboard-onboarding li:first-child,
  .dashboard-onboarding li:last-child {
    padding: 1rem 0;
    border-bottom: 1px solid hsl(var(--border) / 0.72);
  }

  .dashboard-onboarding li:last-child {
    border-bottom: 0;
  }

  .dashboard-onboarding p {
    min-height: 0;
  }
}

@media (max-width: 640px) {
  .dashboard-workspace {
    gap: 0.85rem;
  }

  .dashboard-command-panel,
  .dashboard-panel,
  .dashboard-error {
    padding: 1rem;
  }

  .dashboard-error {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .dashboard-error .dashboard-link-button {
    grid-column: 1 / -1;
    width: 100%;
  }

  .dashboard-command-panel__identity {
    align-items: flex-start;
  }

  .dashboard-command-panel__actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-link-button--ghost {
    grid-column: 1 / -1;
  }

  .dashboard-section__heading,
  .dashboard-panel__heading {
    align-items: flex-start;
  }

  .dashboard-panel__heading {
    flex-direction: column;
  }

  .dashboard-entry {
    grid-template-columns: auto minmax(0, 1fr);
    padding-inline: 0;
  }

  .dashboard-entry__action {
    grid-column: 2;
  }

  .dashboard-entry small {
    white-space: normal;
  }
}
</style>
