<template>
  <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
          <Icon name="shield" size="xs" />
          套餐能力说明
        </div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">当前账号能用什么，一眼看清</h2>
        <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-400">
          SSXZ 会按后台分组、模型和上游账号来开放能力。普通用户优先使用网页工具，开发者和高级用户再创建 API Key 接第三方软件。
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <RouterLink to="/apps" class="btn btn-primary btn-sm">
          <Icon name="grid" size="sm" />
          打开工具中心
        </RouterLink>
        <RouterLink to="/purchase" class="btn btn-secondary btn-sm">
          <Icon name="creditCard" size="sm" />
          查看套餐
        </RouterLink>
      </div>
    </div>

    <div v-if="errorMessage" class="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-200">
      {{ errorMessage }}
    </div>

    <div class="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      <RouterLink
        v-for="item in capabilityCards"
        :key="item.key"
        :to="item.to"
        class="group flex min-h-[170px] flex-col justify-between rounded-lg border p-4 transition hover:-translate-y-0.5 hover:shadow-md"
        :class="item.enabled
          ? 'border-gray-200 bg-gray-50 hover:border-primary-400 dark:border-dark-700 dark:bg-dark-800/70 dark:hover:border-primary-500/70'
          : 'border-amber-200 bg-amber-50/70 hover:border-amber-300 dark:border-amber-900/60 dark:bg-amber-900/10'"
      >
        <div>
          <div class="mb-3 flex items-center justify-between gap-2">
            <span class="inline-flex h-9 w-9 items-center justify-center rounded-lg bg-white text-primary-600 shadow-sm dark:bg-dark-900 dark:text-primary-300">
              <Icon :name="item.icon" size="sm" />
            </span>
            <span
              class="rounded-md px-2 py-0.5 text-xs font-semibold"
              :class="item.enabled
                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                : 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-200'"
            >
              {{ item.enabled ? '已开通' : '待开通' }}
            </span>
          </div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.title }}</h3>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">{{ item.description }}</p>
          <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ item.reason }}</p>
        </div>
        <span class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary-600 dark:text-primary-400">
          {{ item.action }}
          <Icon name="arrowRight" size="sm" class="transition group-hover:translate-x-1" />
        </span>
      </RouterLink>
    </div>

    <div class="mt-5 grid gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)]">
      <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/70">
        <div class="mb-3 flex items-center justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">推荐后台分组结构</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">后续新增上游账号时，按能力分组会更好排查和计费。</p>
          </div>
          <span v-if="loading" class="text-xs text-gray-500 dark:text-gray-400">检测中...</span>
        </div>

        <div class="grid gap-2 md:grid-cols-2">
          <div
            v-for="plan in planGroups"
            :key="plan.name"
            class="rounded-lg bg-white p-3 dark:bg-dark-900"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="font-mono text-xs font-semibold text-primary-700 dark:text-primary-300">{{ plan.name }}</span>
              <span class="rounded-md bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                {{ plan.priceHint }}
              </span>
            </div>
            <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ plan.title }}</p>
            <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ plan.description }}</p>
          </div>
        </div>
      </div>

      <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/70">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">当前检测到的分组</h3>
        <div class="mt-3 flex flex-wrap gap-2">
          <span
            v-for="groupName in activeGroupLabels"
            :key="groupName"
            class="rounded-full bg-white px-3 py-1 text-xs font-medium text-gray-700 shadow-sm dark:bg-dark-900 dark:text-gray-200"
          >
            {{ groupName }}
          </span>
        </div>
        <div class="mt-4 space-y-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
          <p>网页聊天、电商文案会继续走 Sub2 的 Key、余额、日志和扣费，不需要客户理解接口。</p>
          <p>API Key 适合 Cherry Studio、Codex、Claude Code、CC Switch 这类第三方工具，不建议普通客户一上来就用。</p>
          <p>作图必须有支持 OpenAI Images API 的上游账号和图片分组，否则页面会显示待开通。</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useUserCapabilities } from '@/composables/useUserCapabilities'
import Icon from '@/components/icons/Icon.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']

const {
  activeGroupLabels,
  capabilities,
  errorMessage,
  loadCapabilities,
  loading
} = useUserCapabilities()

const iconByKey: Record<string, IconName> = {
  chat: 'chat',
  commerce: 'clipboard',
  image: 'sparkles',
  developer: 'terminal'
}

const targetByKey: Record<string, string> = {
  chat: '/ai-chat',
  commerce: '/ai-chat?mode=ecommerce',
  image: '/image-studio',
  developer: '/keys'
}

const actionByKey: Record<string, string> = {
  chat: '开始聊天',
  commerce: '写电商文案',
  image: '进入作图',
  developer: '管理 API Key'
}

const capabilityCards = computed(() => capabilities.value.map((item) => ({
  ...item,
  icon: iconByKey[item.key] || 'grid',
  to: targetByKey[item.key] || '/apps',
  action: actionByKey[item.key] || '打开'
})))

const planGroups = [
  {
    name: 'chat-basic',
    title: '普通聊天包',
    description: '适合日常问答、写作、翻译、总结，主入口是网页 AI 聊天。',
    priceHint: '普通用户'
  },
  {
    name: 'chat-premium',
    title: '高质量模型包',
    description: '适合更高质量的 GPT 文本模型、复杂任务和更稳定的输出。',
    priceHint: '进阶用户'
  },
  {
    name: 'commerce',
    title: '电商工具包',
    description: '适合商品标题、卖点、详情页、小红书、直播口播和客服回复。',
    priceHint: '电商用户'
  },
  {
    name: 'image',
    title: '图片生成包',
    description: '适合文生图、上传改图、商品换背景、白底图和海报图。',
    priceHint: '按张计费'
  },
  {
    name: 'developer',
    title: '开发者 API 包',
    description: '适合第三方客户端、编程工具、SDK 和自动化脚本接入。',
    priceHint: 'API 用户'
  }
]

onMounted(() => {
  loadCapabilities()
})
</script>
