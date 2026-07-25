<template>
  <AppSectionShell
    :class="['dashboard-shell', `dashboard-shell--${theme}`]"
    title="仪表盘"
    subtitle="账户用量与服务管理"
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
          <section class="dashboard-section" aria-labelledby="dashboard-metrics-title">
            <div class="dashboard-section__heading">
              <div>
                <p class="dashboard-kicker">概览</p>
                <h2 id="dashboard-metrics-title">账户与实时用量</h2>
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

          <details
            v-if="!stats.total_api_keys"
            data-testid="dashboard-onboarding"
            class="dashboard-onboarding-panel"
          >
            <summary>
              <span class="dashboard-onboarding-panel__summary-copy">
                <span class="dashboard-onboarding-panel__icon" aria-hidden="true">
                  <CircleHelp />
                </span>
                <span>
                  <strong>首次接入指南</strong>
                  <small>创建 Key、完成调用，再核对用量和额度</small>
                </span>
              </span>
              <ChevronDown class="dashboard-onboarding-panel__chevron" aria-hidden="true" />
            </summary>
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
          </details>
        </template>
      </div>
    </FoundationProvider>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  ArrowUpRight,
  ChevronDown,
  CircleAlert,
  CircleHelp
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

function getInitialTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'dark'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const authStore = useAuthStore()
const appStore = useAppStore()
const user = computed(() => authStore.user)
const balance = computed(() => user.value?.balance || 0)
const paymentEnabled = computed(() => !!appStore.cachedPublicSettings?.payment_enabled)
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

.dashboard-foundation {
  min-height: 0;
  background: transparent;
}

.dashboard-workspace {
  display: grid;
  gap: 1.25rem;
}

.dashboard-loading {
  display: flex;
  min-height: 12rem;
  align-items: center;
  justify-content: center;
}

.dashboard-error,
.dashboard-onboarding-panel {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  box-shadow: var(--ssxz-shadow-card);
}

.dashboard-error {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
}

.dashboard-error__icon,
.dashboard-onboarding-panel__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
}

.dashboard-error__icon {
  width: 2.5rem;
  height: 2.5rem;
}

.dashboard-error__icon {
  color: hsl(var(--warning));
}

.dashboard-error h1,
.dashboard-error p,
.dashboard-section__heading h2,
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

.dashboard-error__icon svg,
.dashboard-link-button svg,
.dashboard-onboarding-panel__icon svg,
.dashboard-onboarding-panel__chevron,
.dashboard-onboarding a svg {
  width: 1rem;
  height: 1rem;
  flex: 0 0 auto;
  stroke-width: 1.8;
}

.dashboard-kicker {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 650;
  line-height: 1rem;
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
  box-shadow: 0 1px 2px hsl(var(--button-shadow)), 0 4px 10px hsl(var(--button-shadow-hover));
  transition: background-color 150ms ease, border-color 150ms ease, box-shadow 150ms ease, color 150ms ease, transform 100ms ease;
}

.dashboard-link-button:focus-visible {
  outline: 2px solid hsl(var(--ring));
  outline-offset: 2px;
}

.dashboard-link-button:active {
  transform: translateY(1px);
  box-shadow: none;
}

.dashboard-link-button--primary {
  color: hsl(var(--primary-foreground));
  background: hsl(var(--primary));
  border-color: hsl(var(--primary));
}

.dashboard-link-button:hover {
  background: hsl(var(--accent));
  color: hsl(var(--accent-foreground));
  box-shadow: 0 2px 3px hsl(var(--button-shadow)), 0 6px 14px hsl(var(--button-shadow-hover));
  transform: translateY(-1px);
}

.dashboard-link-button--primary:hover {
  background: hsl(var(--primary) / 0.88);
  color: hsl(var(--primary-foreground));
}

.dashboard-section {
  display: grid;
  gap: 0.75rem;
}

.dashboard-section__heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
}

.dashboard-section__heading h2 {
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
  grid-template-columns: minmax(0, 1.75fr) minmax(18rem, 0.75fr);
  align-items: start;
  gap: 1rem;
}

.dashboard-onboarding a {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 600;
  white-space: nowrap;
}

.dashboard-onboarding-panel {
  overflow: hidden;
}

.dashboard-onboarding-panel > summary {
  display: flex;
  min-height: 4.5rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.8rem 1rem;
  list-style: none;
  cursor: pointer;
}

.dashboard-onboarding-panel > summary::-webkit-details-marker {
  display: none;
}

.dashboard-onboarding-panel > summary:focus-visible {
  outline: 2px solid hsl(var(--ring));
  outline-offset: -2px;
}

.dashboard-onboarding-panel__summary-copy {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
}

.dashboard-onboarding-panel__summary-copy > span:last-child {
  display: grid;
  min-width: 0;
  gap: 0.15rem;
}

.dashboard-onboarding-panel__summary-copy strong {
  font-size: 0.8125rem;
  font-weight: 650;
}

.dashboard-onboarding-panel__summary-copy small {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.dashboard-onboarding-panel__icon {
  width: 2rem;
  height: 2rem;
}

.dashboard-onboarding-panel__chevron {
  color: hsl(var(--muted-foreground));
  transition: transform 160ms ease;
}

.dashboard-onboarding-panel[open] .dashboard-onboarding-panel__chevron {
  transform: rotate(180deg);
}

.dashboard-onboarding {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  margin: 0;
  padding: 0 1rem;
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
  margin-top: 0.35rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.dashboard-onboarding a {
  margin-top: 0.65rem;
  color: hsl(var(--foreground));
}

@media (max-width: 900px) {
  .dashboard-detail-grid,
  .dashboard-onboarding {
    grid-template-columns: minmax(0, 1fr);
  }

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

  .dashboard-section__heading {
    align-items: flex-start;
  }

  .dashboard-onboarding-panel > summary {
    align-items: flex-start;
  }

  .dashboard-onboarding-panel__summary-copy {
    align-items: flex-start;
  }
}
</style>
