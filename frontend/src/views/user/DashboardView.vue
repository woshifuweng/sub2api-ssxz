<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <section class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="grid gap-6 p-5 lg:grid-cols-[minmax(0,1.25fr)_minmax(360px,0.75fr)] lg:p-6">
            <div class="flex flex-col justify-between gap-6">
              <div>
                <div class="mb-3 flex flex-wrap items-center gap-2">
                  <span class="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
                    <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
                    SSXZ 图片创作站
                  </span>
                  <span class="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                    登录账号 {{ userEmail }}
                  </span>
                </div>
                <h1 class="text-2xl font-bold tracking-normal text-gray-900 dark:text-white md:text-3xl">
                  先生成图片，再查看和下载
                </h1>
                <p class="mt-3 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-400">
                  普通用户从生图开始：写提示词、生成图片、查看历史、下载成品。余额、用量和充值都在这里串起来，聊天只作为辅助写 prompt 的第二入口。
                </p>
              </div>

              <div class="grid gap-3 sm:grid-cols-3">
                <RouterLink
                  to="/sora"
                  class="inline-flex items-center justify-center gap-2 rounded-xl bg-primary-600 px-4 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-primary-700"
                >
                  <Icon name="sparkles" size="sm" />
                  开始生图
                </RouterLink>
                <RouterLink
                  to="/app/chat"
                  class="inline-flex items-center justify-center gap-2 rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-800 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-500"
                >
                  <Icon name="chat" size="sm" />
                  辅助写 prompt
                </RouterLink>
                <RouterLink
                  to="/usage"
                  class="inline-flex items-center justify-center gap-2 rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm font-semibold text-gray-800 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-primary-500"
                >
                  <Icon name="chart" size="sm" />
                  查看用量
                </RouterLink>
              </div>
            </div>

            <div class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/70">
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
                <div class="rounded-xl bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">当前余额</p>
                  <p class="mt-1 text-xl font-bold text-emerald-600 dark:text-emerald-400">${{ formatMoney(balance) }}</p>
                </div>
                <div class="rounded-xl bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">今日消耗</p>
                  <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">${{ formatMoney(stats.today_actual_cost || 0) }}</p>
                </div>
                <div class="rounded-xl bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">今日请求</p>
                  <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">{{ formatCompact(stats.today_requests || 0) }}</p>
                </div>
                <div class="rounded-xl bg-white p-3 dark:bg-dark-900">
                  <p class="text-xs text-gray-500 dark:text-gray-400">累计请求</p>
                  <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">{{ formatCompact(stats.total_requests || 0) }}</p>
                </div>
              </div>

              <div class="mt-4 flex flex-wrap gap-2">
                <RouterLink to="/purchase" class="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-semibold text-white transition hover:bg-emerald-700">
                  购买套餐
                </RouterLink>
                <RouterLink to="/usage" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-gray-700 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200">
                  用量明细
                </RouterLink>
                <RouterLink to="/orders" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-gray-700 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200">
                  我的订单
                </RouterLink>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p class="text-sm font-medium text-primary-600 dark:text-primary-400">开始使用</p>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">轻量图片工具站</h2>
            </div>
            <p class="max-w-2xl text-sm text-gray-500 dark:text-gray-400">
              常用入口集中在生图、图片历史、下载、余额、用量和充值。复杂的渠道和 API 能力留在后台，不打扰普通用户。
            </p>
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
            <RouterLink
              v-for="entry in productEntries"
              :key="entry.to"
              :to="entry.to"
              class="group flex min-h-[170px] flex-col justify-between rounded-xl border border-gray-200 bg-gray-50 p-4 transition hover:-translate-y-0.5 hover:border-primary-400 hover:bg-primary-50 hover:shadow-md dark:border-dark-700 dark:bg-dark-800/70 dark:hover:border-primary-500/70 dark:hover:bg-primary-950/30"
            >
              <div>
                <div class="mb-3 flex items-center justify-between gap-2">
                  <span class="inline-flex items-center gap-2 rounded-lg bg-white px-2.5 py-1 text-xs font-semibold text-gray-700 shadow-sm dark:bg-dark-900 dark:text-gray-200">
                    <Icon :name="entry.icon" size="xs" />
                    {{ entry.badge }}
                  </span>
                  <Icon name="arrowRight" size="sm" class="text-primary-500 transition group-hover:translate-x-1" />
                </div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ entry.title }}</h3>
                <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">{{ entry.description }}</p>
              </div>
              <span class="mt-4 text-sm font-medium text-primary-600 dark:text-primary-400">{{ entry.action }}</span>
            </RouterLink>
          </div>
        </section>

        <section class="grid gap-4 lg:grid-cols-3">
          <div
            v-for="step in onboardingSteps"
            :key="step.title"
            class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900"
          >
            <div class="mb-4 flex h-9 w-9 items-center justify-center rounded-xl bg-primary-50 text-sm font-bold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
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
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { ModelStat, TrendDataPoint, UsageLog } from '@/types'

const authStore = useAuthStore()
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

const productEntries = [
  {
    to: '/sora',
    icon: 'sparkles' as const,
    badge: '主入口',
    title: '生成图片',
    description: '输入提示词，选择模型和比例，直接生成商品图、场景图、海报图或灵感图。',
    action: '进入生图'
  },
  {
    to: '/sora',
    icon: 'download' as const,
    badge: '图片历史',
    title: '查看和下载',
    description: '回到历史记录查看生成结果，预览图片或视频，并下载需要保留的成品。',
    action: '查看历史'
  },
  {
    to: '/app/chat',
    icon: 'chat' as const,
    badge: '辅助入口',
    title: '聊天写 prompt',
    description: '需要打磨描述、翻译风格词或扩写创意时，用聊天先把提示词写顺。',
    action: '打开聊天'
  },
  {
    to: '/purchase',
    icon: 'creditCard' as const,
    badge: '余额',
    title: '充值和订单',
    description: '查看余额和消耗，余额不足时进入充值页，订单记录可以随时回查。',
    action: '去充值'
  }
]

const onboardingSteps = [
  {
    index: '01',
    title: '先进入生图',
    description: '普通用户不需要理解接口和渠道。打开 Sora 创作，写好提示词后直接生成图片。',
    to: '/sora',
    action: '去生图'
  },
  {
    index: '02',
    title: '再看历史和下载',
    description: '生成结果会进入图片历史。需要使用时预览、保存或下载，不用在聊天记录里翻找。',
    to: '/sora',
    action: '查看历史'
  },
  {
    index: '03',
    title: '最后看余额和用量',
    description: '费用仍由后台统一记录。用户只需要看余额、用量和订单，余额不足时充值。',
    to: '/usage',
    action: '查看用量'
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
    await authStore.refreshUser()
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
