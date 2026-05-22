<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900 lg:p-6">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
              <Icon name="grid" size="xs" />
              SSXZ 轻应用中心
            </div>
            <h1 class="text-2xl font-bold tracking-normal text-gray-900 dark:text-white md:text-3xl">
              不懂提示词，也能直接开始
            </h1>
            <p class="mt-3 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-400">
              这里把常见场景做成固定入口。普通用户点聊天工具，电商用户点文案和作图工具，开发者再去 API Key 页面。
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <RouterLink to="/ai-chat" class="btn btn-primary">
              <Icon name="chat" size="sm" />
              AI 聊天
            </RouterLink>
            <RouterLink to="/image-studio" class="btn btn-secondary">
              <Icon name="sparkles" size="sm" />
              AI 作图
            </RouterLink>
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div class="flex flex-wrap gap-2">
            <button
              v-for="category in categories"
              :key="category.value"
              type="button"
              class="rounded-lg px-3 py-2 text-sm font-medium transition"
              :class="activeCategory === category.value
                ? 'bg-primary-600 text-white shadow-sm'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700'"
              @click="activeCategory = category.value"
            >
              {{ category.label }}
            </button>
          </div>
          <div class="relative w-full md:w-80">
            <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model="keyword"
              class="input pl-9"
              placeholder="搜索：电商、客服、代码、作图..."
            />
          </div>
        </div>
      </section>

      <section>
        <div class="mb-3 flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">推荐工具</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">先做高频需求，后面可以继续扩展成你的模板市场。</p>
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ filteredApps.length }} 个工具</span>
        </div>

        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <RouterLink
            v-for="app in filteredApps"
            :key="app.title"
            :to="app.to"
            class="group flex min-h-[210px] flex-col justify-between rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary-400 hover:shadow-md dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-500/70"
          >
            <div>
              <div class="mb-4 flex items-start justify-between gap-3">
                <span
                  class="inline-flex h-10 w-10 items-center justify-center rounded-lg"
                  :class="app.iconClass"
                >
                  <Icon :name="app.icon" size="md" />
                </span>
                <span class="rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                  {{ app.categoryLabel }}
                </span>
              </div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ app.title }}</h3>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">{{ app.description }}</p>
              <div class="mt-3 flex flex-wrap gap-1.5">
                <span
                  v-for="tag in app.tags"
                  :key="tag"
                  class="rounded-md bg-gray-50 px-2 py-1 text-xs text-gray-500 dark:bg-dark-800/80 dark:text-gray-400"
                >
                  {{ tag }}
                </span>
              </div>
            </div>
            <span class="mt-5 inline-flex items-center gap-1 text-sm font-semibold text-primary-600 dark:text-primary-400">
              {{ app.action }}
              <Icon name="arrowRight" size="sm" class="transition group-hover:translate-x-1" />
            </span>
          </RouterLink>
        </div>

        <div v-if="filteredApps.length === 0" class="rounded-lg border border-dashed border-gray-300 bg-white p-10 text-center dark:border-dark-700 dark:bg-dark-900">
          <Icon name="search" size="lg" class="mx-auto text-gray-400" />
          <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">没有找到对应工具，换个关键词试试。</p>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'

type AppCategory = 'all' | 'chat' | 'commerce' | 'image' | 'developer'
type IconName = InstanceType<typeof Icon>['$props']['name']

type AppEntry = {
  title: string
  description: string
  category: Exclude<AppCategory, 'all'>
  categoryLabel: string
  icon: IconName
  iconClass: string
  tags: string[]
  action: string
  to: string
}

const activeCategory = ref<AppCategory>('all')
const keyword = ref('')

const categories: Array<{ value: AppCategory; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'chat', label: '聊天写作' },
  { value: 'commerce', label: '电商经营' },
  { value: 'image', label: '图片创作' },
  { value: 'developer', label: '开发接入' }
]

const apps: AppEntry[] = [
  {
    title: '通用 AI 聊天',
    description: '日常问答、写作、翻译、总结和头脑风暴，适合普通客户直接使用。',
    category: 'chat',
    categoryLabel: '聊天写作',
    icon: 'chat',
    iconClass: 'bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-300',
    tags: ['问答', '写作', '总结'],
    action: '开始聊天',
    to: '/ai-chat'
  },
  {
    title: '代码问题定位',
    description: '把报错、需求或代码片段交给 AI，先定位原因，再输出可执行步骤。',
    category: 'developer',
    categoryLabel: '开发接入',
    icon: 'terminal',
    iconClass: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200',
    tags: ['代码', '排错', '步骤'],
    action: '打开代码助手',
    to: `/ai-chat?prompt=${encodeURIComponent('我遇到了一个代码问题，请先帮我定位原因，再给出可执行的解决步骤：')}`
  },
  {
    title: '商品标题生成',
    description: '输入商品名和核心卖点，生成更适合搜索和投放的标题方案。',
    category: 'commerce',
    categoryLabel: '电商经营',
    icon: 'clipboard',
    iconClass: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/30 dark:text-emerald-300',
    tags: ['标题', '搜索', '转化'],
    action: '生成标题',
    to: `/ai-chat?mode=ecommerce&scene=titles&productName=${encodeURIComponent('蓝牙耳机')}&sellingPoints=${encodeURIComponent('降噪、长续航、轻便')}`
  },
  {
    title: '五点卖点提炼',
    description: '把杂乱卖点整理成清晰、有购买理由的五点卖点。',
    category: 'commerce',
    categoryLabel: '电商经营',
    icon: 'lightbulb',
    iconClass: 'bg-amber-50 text-amber-600 dark:bg-amber-950/30 dark:text-amber-300',
    tags: ['卖点', '详情页', '导购'],
    action: '提炼卖点',
    to: `/ai-chat?mode=ecommerce&scene=bullet_points&productName=${encodeURIComponent('蓝牙耳机')}&sellingPoints=${encodeURIComponent('降噪、长续航、轻便')}`
  },
  {
    title: '小红书种草文案',
    description: '生成更自然的口播式种草笔记，适合上新、测评和达人素材。',
    category: 'commerce',
    categoryLabel: '电商经营',
    icon: 'sparkles',
    iconClass: 'bg-rose-50 text-rose-600 dark:bg-rose-950/30 dark:text-rose-300',
    tags: ['小红书', '种草', '口语化'],
    action: '写种草文案',
    to: `/ai-chat?mode=ecommerce&scene=xiaohongshu&platform=${encodeURIComponent('小红书')}&tone=${encodeURIComponent('种草风')}&productName=${encodeURIComponent('蓝牙耳机')}`
  },
  {
    title: '客服回复优化',
    description: '把生硬回复改成专业、克制、有安抚和转化力的客服话术。',
    category: 'commerce',
    categoryLabel: '电商经营',
    icon: 'chatBubble',
    iconClass: 'bg-cyan-50 text-cyan-600 dark:bg-cyan-950/30 dark:text-cyan-300',
    tags: ['客服', '售后', '转化'],
    action: '优化回复',
    to: `/ai-chat?mode=ecommerce&scene=customer_reply&extra=${encodeURIComponent('客户咨询内容：')}`
  },
  {
    title: '商品图换背景',
    description: '上传商品图后生成白底图、场景图或更适合平台展示的商品图。',
    category: 'image',
    categoryLabel: '图片创作',
    icon: 'sparkles',
    iconClass: 'bg-violet-50 text-violet-600 dark:bg-violet-950/30 dark:text-violet-300',
    tags: ['商品图', '换背景', '白底图'],
    action: '进入作图',
    to: '/image-studio'
  },
  {
    title: 'API 接入指南',
    description: '生成 Key 后接入 Cherry Studio、Codex、Claude Code、CC Switch 或 SDK。',
    category: 'developer',
    categoryLabel: '开发接入',
    icon: 'key',
    iconClass: 'bg-blue-50 text-blue-600 dark:bg-blue-950/30 dark:text-blue-300',
    tags: ['API Key', 'Base URL', '第三方工具'],
    action: '管理密钥',
    to: '/keys'
  }
]

const filteredApps = computed(() => {
  const needle = keyword.value.trim().toLowerCase()
  return apps.filter((app) => {
    const categoryMatched = activeCategory.value === 'all' || app.category === activeCategory.value
    if (!categoryMatched) return false
    if (!needle) return true
    return [
      app.title,
      app.description,
      app.categoryLabel,
      ...app.tags
    ].join(' ').toLowerCase().includes(needle)
  })
})
</script>
