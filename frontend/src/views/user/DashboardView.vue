<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <section class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="grid gap-6 p-5 lg:grid-cols-[minmax(0,1.25fr)_minmax(360px,0.75fr)] lg:p-6">
            <div class="flex flex-col justify-between gap-6">
              <div>
                <div class="mb-3 flex flex-wrap items-center gap-2">
                  <span class="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
                    <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
                    SSXZ AI 工作台
                  </span>
                  <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                    登录账号 {{ userEmail }}
                  </span>
                </div>
                <h1 class="text-2xl font-bold tracking-normal text-gray-900 dark:text-white md:text-3xl">
                  先看能力，再开始使用 AI
                </h1>
                <p class="mt-3 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-400">
                  普通用户可以直接聊天，电商用户进入文案和作图工具，开发者再创建 API Key。系统会根据账号分组显示可用能力，避免客户点进不能用的入口。
                </p>
              </div>

              <div class="flex flex-wrap gap-2">
                <span
                  v-for="groupName in activeGroupLabels"
                  :key="groupName"
                  class="rounded-full border border-gray-200 bg-gray-50 px-3 py-1 text-xs font-medium text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300"
                >
                  {{ groupName }}
                </span>
              </div>

              <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <RouterLink
                  to="/apps"
                  class="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-800 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-500"
                >
                  <Icon name="grid" size="sm" />
                  轻应用
                </RouterLink>
                <RouterLink
                  to="/ai-chat"
                  class="inline-flex items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-primary-700"
                >
                  <Icon name="chat" size="sm" />
                  开始聊天
                </RouterLink>
                <RouterLink
                  to="/ai-chat?mode=ecommerce"
                  class="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-800 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-500"
                >
                  <Icon name="clipboard" size="sm" />
                  电商文案
                </RouterLink>
                <RouterLink
                  to="/image-studio"
                  class="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-800 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-500"
                >
                  <Icon name="sparkles" size="sm" />
                  AI 作图
                </RouterLink>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/70">
              <div class="mb-4 flex items-center justify-between">
                <div>
                  <p class="text-sm font-semibold text-gray-900 dark:text-white">账户概览</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">费用以实际模型和后台费率为准</p>
                </div>
                <RouterLink to="/usage" class="text-xs font-semibold text-primary-600 hover:text-primary-700 dark:text-primary-400">
                  查看明细
                </RouterLink>
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div class="rounded-lg bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">当前余额</p>
                  <p class="mt-1 text-xl font-bold text-emerald-600 dark:text-emerald-400">${{ formatMoney(balance) }}</p>
                </div>
                <div class="rounded-lg bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">今日消耗</p>
                  <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">${{ formatMoney(stats.today_actual_cost || 0) }}</p>
                </div>
                <div class="rounded-lg bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">今日请求</p>
                  <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">{{ formatCompact(stats.today_requests || 0) }}</p>
                </div>
                <div class="rounded-lg bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">可用 Key</p>
                  <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">{{ stats.active_api_keys || 0 }}/{{ stats.total_api_keys || 0 }}</p>
                </div>
              </div>

              <div class="mt-4 flex flex-wrap gap-2">
                <RouterLink to="/purchase" class="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-semibold text-white transition hover:bg-emerald-700">
                  购买套餐
                </RouterLink>
                <RouterLink to="/redeem" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-gray-700 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200">
                  兑换额度
                </RouterLink>
                <RouterLink to="/keys" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-gray-700 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200">
                  API Key
                </RouterLink>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p class="text-sm font-medium text-primary-600 dark:text-primary-400">账号能力</p>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">当前账号能使用哪些服务</h2>
            </div>
            <p class="max-w-2xl text-sm text-gray-500 dark:text-gray-400">
              这里会根据 Sub2 分组、渠道和模型自动判断。你后续建立套餐后，客户会看到更清楚的“已开通 / 未开通”状态。
            </p>
          </div>

          <div v-if="capabilityError" class="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-200">
            {{ capabilityError }}
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
            <RouterLink
              v-for="entry in productEntries"
              :key="entry.to"
              :to="entry.to"
              class="group flex min-h-[190px] flex-col justify-between rounded-lg border p-4 transition hover:-translate-y-0.5 hover:shadow-md"
              :class="entry.enabled
                ? 'border-gray-200 bg-gray-50 hover:border-primary-400 hover:bg-primary-50 dark:border-dark-700 dark:bg-dark-800/70 dark:hover:border-primary-500/70 dark:hover:bg-primary-950/30'
                : 'border-amber-200 bg-amber-50/60 hover:border-amber-300 dark:border-amber-900/50 dark:bg-amber-900/10'"
            >
              <div>
                <div class="mb-3 flex items-center justify-between gap-2">
                  <span class="inline-flex items-center gap-2 rounded-lg bg-white px-2.5 py-1 text-xs font-semibold text-gray-700 shadow-sm dark:bg-dark-900 dark:text-gray-200">
                    <Icon :name="entry.icon" size="xs" />
                    {{ entry.badge }}
                  </span>
                  <span
                    class="rounded-md px-2 py-0.5 text-xs font-semibold"
                    :class="entry.enabled
                      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                      : 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-200'"
                  >
                    {{ entry.enabled ? '已开通' : '待开通' }}
                  </span>
                </div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ entry.title }}</h3>
                <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">{{ entry.description }}</p>
                <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ entry.reason }}</p>
              </div>
              <span class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary-600 dark:text-primary-400">
                {{ entry.action }}
                <Icon name="arrowRight" size="sm" class="transition group-hover:translate-x-1" />
              </span>
            </RouterLink>
          </div>
        </section>

        <section class="grid gap-4 lg:grid-cols-3">
          <div
            v-for="step in onboardingSteps"
            :key="step.title"
            class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900"
          >
            <div class="mb-4 flex h-9 w-9 items-center justify-center rounded-lg bg-primary-50 text-sm font-bold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
              {{ step.index }}
            </div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ step.title }}</h3>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">{{ step.description }}</p>
            <RouterLink :to="step.to" class="mt-4 inline-flex text-sm font-semibold text-primary-600 hover:text-primary-700 dark:text-primary-400">
              {{ step.action }}
            </RouterLink>
          </div>
        </section>

        <UserDashboardStats :stats="stats" :balance="balance" :is-simple="authStore.isSimpleMode" />

        <UserDashboardCharts
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          v-model:granularity="granularity"
          :loading="loadingCharts"
          :trend="trendData"
          :models="modelStats"
          @dateRangeChange="loadCharts"
          @granularityChange="loadCharts"
        />

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="lg:col-span-2">
            <UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" />
          </div>
          <div class="lg:col-span-1">
            <UserDashboardQuickActions />
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import { useUserCapabilities } from '@/composables/useUserCapabilities'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { ModelStat, TrendDataPoint, UsageLog } from '@/types'

const authStore = useAuthStore()
const {
  activeGroupLabels,
  capabilities,
  errorMessage: capabilityError,
  loadCapabilities
} = useUserCapabilities()

const user = computed(() => authStore.user)
const balance = computed(() => user.value?.balance || 0)
const userEmail = computed(() => user.value?.email || '当前用户')

const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])

const capabilityByKey = computed(() => {
  return Object.fromEntries(capabilities.value.map((item) => [item.key, item]))
})

const productEntries = computed(() => [
  {
    to: '/ai-chat',
    icon: 'chat' as const,
    badge: '普通聊天',
    title: 'AI 聊天',
    description: '像官网一样直接提问、写作、翻译、总结资料，也适合普通客户日常使用。',
    action: '打开聊天',
    enabled: capabilityByKey.value.chat?.enabled ?? true,
    reason: capabilityByKey.value.chat?.reason || '使用后台可用聊天模型'
  },
  {
    to: '/ai-chat?mode=ecommerce',
    icon: 'clipboard' as const,
    badge: '电商工具',
    title: '电商文案',
    description: '按商品名、卖点、平台和风格生成标题、卖点、小红书、直播口播等内容。',
    action: '写商品文案',
    enabled: capabilityByKey.value.commerce?.enabled ?? true,
    reason: capabilityByKey.value.commerce?.reason || '使用聊天模型加电商模板'
  },
  {
    to: '/image-studio',
    icon: 'sparkles' as const,
    badge: '图片工作台',
    title: 'AI 作图',
    description: '用于商品图、白底图、场景图和海报图。需要图片上游账号或图片分组。',
    action: '进入作图',
    enabled: capabilityByKey.value.image?.enabled ?? false,
    reason: capabilityByKey.value.image?.reason || '需要图片分组'
  },
  {
    to: '/keys',
    icon: 'key' as const,
    badge: '开发者',
    title: '第三方接入',
    description: '生成 API Key 后，可接入 Cherry Studio、Codex、Claude Code、CC Switch 等工具。',
    action: '管理密钥',
    enabled: capabilityByKey.value.developer?.enabled ?? true,
    reason: capabilityByKey.value.developer?.reason || '可创建 API Key 接入第三方工具'
  }
])

const onboardingSteps = [
  {
    index: '01',
    title: '普通客户先用网页',
    description: '不需要创建 Key，也不需要懂接口。直接进入 AI 聊天或电商文案，后台会自动完成模型调用和扣费。',
    to: '/ai-chat',
    action: '去聊天'
  },
  {
    index: '02',
    title: '电商客户走固定模板',
    description: '商品名、卖点、平台、风格填好后再生成，结果会比在第三方软件里随便提问更稳定。',
    to: '/ai-chat?mode=ecommerce',
    action: '去电商模式'
  },
  {
    index: '03',
    title: '高级用户再接 API',
    description: '需要 Cherry Studio、编程工具或自动化脚本时，再生成 API Key，并在用量记录里查看每次消耗。',
    to: '/keys',
    action: '去生成 Key'
  }
]

const formatLD = (d: Date) => d.toISOString().split('T')[0]
const startDate = ref(formatLD(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatLD(new Date()))
const granularity = ref('day')

const formatMoney = (value: number) =>
  new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 4
  }).format(value || 0)

const formatCompact = (value: number) =>
  new Intl.NumberFormat('en-US', {
    notation: 'compact',
    maximumFractionDigits: 1
  }).format(value || 0)

const loadStats = async () => {
  loading.value = true
  try {
    await Promise.all([
      authStore.refreshUser(),
      loadCapabilities()
    ])
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const res = await Promise.all([
      usageAPI.getDashboardTrend({
        start_date: startDate.value,
        end_date: endDate.value,
        granularity: granularity.value as 'day' | 'hour'
      }),
      usageAPI.getDashboardModels({
        start_date: startDate.value,
        end_date: endDate.value
      })
    ])
    trendData.value = res[0].trend || []
    modelStats.value = res[1].models || []
  } catch (error) {
    console.error('Failed to load charts:', error)
  } finally {
    loadingCharts.value = false
  }
}

const loadRecent = async () => {
  loadingUsage.value = true
  try {
    const res = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = res.items.slice(0, 5)
  } catch (error) {
    console.error('Failed to load recent usage:', error)
  } finally {
    loadingUsage.value = false
  }
}

onMounted(() => {
  loadStats()
  loadCharts()
  loadRecent()
})
</script>
