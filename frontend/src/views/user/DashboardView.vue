<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12"><LoadingSpinner /></div>
      <template v-else-if="stats">
        <UserDashboardStats :stats="stats" :balance="user?.balance || 0" :is-simple="authStore.isSimpleMode" />
        <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p class="text-sm font-medium text-primary-600 dark:text-primary-400">开始使用</p>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">选择你要使用的 AI 服务</h2>
            </div>
            <p class="max-w-2xl text-sm text-gray-500 dark:text-gray-400">
              普通用户可以直接聊天，电商用户可以进入电商文案模式，开发者或第三方软件用户可以生成 API Key 后接入。
            </p>
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
            <RouterLink
              v-for="entry in productEntries"
              :key="entry.to"
              :to="entry.to"
              class="group flex min-h-[154px] flex-col justify-between rounded-xl border border-gray-200 bg-gray-50 p-4 transition hover:-translate-y-0.5 hover:border-primary-400 hover:bg-primary-50 hover:shadow-md dark:border-dark-700 dark:bg-dark-800/70 dark:hover:border-primary-500/70 dark:hover:bg-primary-950/30"
            >
              <div>
                <div class="mb-3 flex items-center justify-between gap-2">
                  <span class="rounded-lg bg-white px-2.5 py-1 text-xs font-semibold text-gray-700 shadow-sm dark:bg-dark-900 dark:text-gray-200">
                    {{ entry.badge }}
                  </span>
                  <span class="text-lg text-primary-500 transition group-hover:translate-x-1">-></span>
                </div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ entry.title }}</h3>
                <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">{{ entry.description }}</p>
              </div>
              <span class="mt-4 text-sm font-medium text-primary-600 dark:text-primary-400">{{ entry.action }}</span>
            </RouterLink>
          </div>
        </section>
        <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadCharts" @granularityChange="loadCharts" />
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="lg:col-span-2"><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
          <div class="lg:col-span-1"><UserDashboardQuickActions /></div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'; import { useAuthStore } from '@/stores/auth'; import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'; import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'; import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'; import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat } from '@/types'

const authStore = useAuthStore(); const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null); const loading = ref(false); const loadingUsage = ref(false); const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([]); const modelStats = ref<ModelStat[]>([]); const recentUsage = ref<UsageLog[]>([])
const productEntries = [
  {
    to: '/ai-chat',
    badge: 'GPT-5.5',
    title: 'AI 聊天',
    description: '像官网一样直接提问、写作、翻译、总结资料，适合普通用户日常使用。',
    action: '打开聊天'
  },
  {
    to: '/ai-chat?mode=ecommerce',
    badge: '电商',
    title: '电商文案',
    description: '进入 AI 聊天后切换电商模式，按商品名、卖点和平台生成商用文案。',
    action: '写商品文案'
  },
  {
    to: '/image-studio',
    badge: '图片',
    title: 'AI 作图',
    description: '上传商品图或填写需求生成图片。需要账号分组里有支持图片接口的上游。',
    action: '进入作图'
  },
  {
    to: '/keys',
    badge: 'API',
    title: '第三方接入',
    description: '生成 API Key 后，可接入 Cherry Studio、Codex、Claude Code 等工具。',
    action: '管理密钥'
  }
]

const formatLD = (d: Date) => d.toISOString().split('T')[0]
const startDate = ref(formatLD(new Date(Date.now() - 6 * 86400000))); const endDate = ref(formatLD(new Date())); const granularity = ref('day')

const loadStats = async () => { loading.value = true; try { await authStore.refreshUser(); stats.value = await usageAPI.getDashboardStats() } catch (error) { console.error('Failed to load dashboard stats:', error) } finally { loading.value = false } }
const loadCharts = async () => { loadingCharts.value = true; try { const res = await Promise.all([usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as any }), usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })]); trendData.value = res[0].trend || []; modelStats.value = res[1].models || [] } catch (error) { console.error('Failed to load charts:', error) } finally { loadingCharts.value = false } }
const loadRecent = async () => { loadingUsage.value = true; try { const res = await usageAPI.getByDateRange(startDate.value, endDate.value); recentUsage.value = res.items.slice(0, 5) } catch (error) { console.error('Failed to load recent usage:', error) } finally { loadingUsage.value = false } }

onMounted(() => { loadStats(); loadCharts(); loadRecent() })
</script>
