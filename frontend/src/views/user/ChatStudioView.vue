<template>
  <div class="mx-auto grid max-w-7xl gap-5 xl:grid-cols-[280px_minmax(0,1fr)_320px]">
    <aside class="space-y-4">
      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <div class="mb-4 flex items-center justify-between gap-3">
          <div>
            <h1 class="text-lg font-semibold text-gray-900 dark:text-white">SSXZ AI</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">余额约 {{ balanceText }}</p>
          </div>
          <button type="button" class="btn btn-primary btn-sm" @click="createSession(mode)">
            <Icon name="plus" size="sm" />
            <span>新建</span>
          </button>
        </div>

        <div class="grid grid-cols-2 gap-2">
          <button
            type="button"
            class="rounded-lg border px-3 py-2 text-sm transition"
            :class="mode === 'chat'
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
              : 'border-gray-200 bg-gray-50 text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-200'"
            @click="switchMode('chat')"
          >
            <Icon name="chat" size="sm" class="mx-auto mb-1" />
            对话
          </button>
          <button
            type="button"
            class="rounded-lg border px-3 py-2 text-sm transition"
            :class="mode === 'commerce'
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
              : 'border-gray-200 bg-gray-50 text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-200'"
            @click="switchMode('commerce')"
          >
            <Icon name="sparkles" size="sm" class="mx-auto mb-1" />
            电商
          </button>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <div class="mb-2 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">会话历史</h2>
          <button type="button" class="text-xs text-gray-500 hover:text-primary-600 dark:text-dark-300" @click="clearAllSessions">
            清空
          </button>
        </div>
        <div class="max-h-[360px] space-y-2 overflow-y-auto pr-1">
          <button
            v-for="session in sessions"
            :key="session.id"
            type="button"
            class="w-full rounded-lg border px-3 py-2 text-left transition"
            :class="session.id === activeSessionId
              ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30'
              : 'border-gray-200 bg-gray-50 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-700'"
            @click="activateSession(session.id)"
          >
            <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ session.title }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">
              {{ session.mode === 'commerce' ? '电商文案' : '普通聊天' }} · {{ formatSessionTime(session.updatedAt) }}
            </span>
          </button>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">快捷开始</h2>
        <div class="space-y-2">
          <button
            v-for="item in visibleQuickPrompts"
            :key="item.title"
            type="button"
            class="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-left text-sm transition hover:border-primary-400 hover:bg-primary-50 dark:border-dark-600 dark:bg-dark-700 dark:hover:border-primary-500"
            @click="applyPrompt(item.prompt)"
          >
            <span class="block font-medium text-gray-800 dark:text-dark-100">{{ item.title }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ item.desc }}</span>
          </button>
        </div>
      </section>
    </aside>

    <section class="flex min-h-[760px] flex-col rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <header class="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-600 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ workspaceTitle }}</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            网页内直接使用，后台自动选择兼容 Key，并继续走 Sub2 余额、日志和扣费。
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <select v-model="selectedModel" class="input h-9 w-36 text-sm">
            <option v-for="model in models" :key="model.id" :value="model.id">{{ model.name }}</option>
          </select>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="sending || !canRegenerate" @click="regenerateLast">
            <Icon name="refresh" size="sm" />
            <span>重新生成</span>
          </button>
          <button type="button" class="btn btn-secondary btn-sm" @click="resetActiveSession">
            <Icon name="trash" size="sm" />
            <span>清空上下文</span>
          </button>
        </div>
      </header>

      <div ref="messageListRef" class="flex-1 space-y-4 overflow-y-auto px-5 py-5">
        <div
          v-if="capabilityError"
          class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm leading-6 text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-200"
        >
          {{ capabilityError }}
        </div>
        <div
          v-else-if="loaded && !hasChat"
          class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm leading-6 text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300"
        >
          当前账号暂未检测到可用聊天分组。请在 Sub2 后台给用户分配包含 gpt-5.5 或其他聊天模型的分组。
        </div>

        <div v-if="currentMessages.length === 0" class="flex h-full min-h-[420px] items-center justify-center">
          <div class="max-w-md text-center">
            <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
              <Icon :name="mode === 'commerce' ? 'sparkles' : 'chat'" size="lg" />
            </div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ emptyTitle }}</h3>
            <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ emptyDescription }}</p>
          </div>
        </div>

        <article
          v-for="message in currentMessages"
          :key="message.id"
          class="flex"
          :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            class="max-w-[88%] rounded-lg px-4 py-3 text-sm leading-6"
            :class="message.role === 'user'
              ? 'bg-primary-600 text-white'
              : 'border border-gray-200 bg-gray-50 text-gray-800 dark:border-dark-600 dark:bg-dark-900/60 dark:text-dark-100'"
          >
            <div v-if="message.role === 'assistant'" class="studio-markdown" v-html="renderMarkdown(message.content)" />
            <div v-else class="whitespace-pre-wrap">{{ message.content }}</div>
            <button
              v-if="message.role === 'assistant'"
              type="button"
              class="mt-3 inline-flex items-center gap-1 rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-500 hover:text-primary-600 dark:border-dark-600 dark:text-dark-300"
              @click="copyText(message.content)"
            >
              <Icon name="copy" size="xs" />
              复制
            </button>
          </div>
        </article>

        <article v-if="sending" class="flex justify-start">
          <div class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-900/60 dark:text-dark-300">
            正在生成...
          </div>
        </article>
      </div>

      <footer class="border-t border-gray-200 p-4 dark:border-dark-600">
        <p v-if="errorMessage" class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
          {{ errorMessage }}
        </p>

        <div v-if="mode === 'chat'" class="rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-900/60">
          <textarea
            v-model="draft"
            class="min-h-24 w-full resize-none bg-transparent px-3 py-2 text-sm text-gray-900 outline-none placeholder:text-gray-400 dark:text-white"
            placeholder="输入你的问题，例如：帮我写一份蓝牙耳机详情页卖点文案"
            @keydown.enter.exact.prevent="sendChatMessage"
          />
          <div class="flex items-center justify-between gap-3 px-2 pb-1">
            <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
              <span class="rounded-md bg-gray-100 px-2 py-1 dark:bg-dark-700">{{ selectedModel }}</span>
              <span>Enter 发送，Shift+Enter 换行</span>
            </div>
            <button type="button" class="btn btn-primary" :disabled="sending || !draft.trim() || (loaded && !hasChat)" @click="sendChatMessage">
              <Icon v-if="sending" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="arrowUp" size="sm" />
              <span>{{ sending ? '发送中' : '发送' }}</span>
            </button>
          </div>
        </div>

        <div v-else class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p class="text-xs text-gray-500 dark:text-dark-400">
            当前使用 {{ selectedModel }}，后台会根据所选电商场景自动套用专业提示词模板。
          </p>
          <div class="flex gap-2">
            <button type="button" class="btn btn-secondary" @click="copyText(buildCommerceVisibleRequest(buildCommerceContext()))">
              <Icon name="copy" size="sm" />
              <span>复制需求</span>
            </button>
            <button type="button" class="btn btn-primary" :disabled="sending || !canGenerateCommerce || (loaded && !hasChat)" @click="sendCommerceMessage">
              <Icon v-if="sending" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="sparkles" size="sm" />
              <span>{{ sending ? '生成中' : '生成文案' }}</span>
            </button>
          </div>
        </div>
      </footer>
    </section>

    <aside class="space-y-5">
      <section v-if="mode === 'commerce'" class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">电商工作台</h2>
        <div class="mt-4 space-y-4">
          <div>
            <label class="input-label">场景</label>
            <select v-model="commerceForm.outputGoal" class="input">
              <option v-for="scene in commerceScenes" :key="scene.value" :value="scene.value">{{ scene.label }}</option>
            </select>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ selectedCommerceScene?.desc }}</p>
          </div>

          <div>
            <label class="input-label">商品名称</label>
            <input v-model="commerceForm.productName" class="input" placeholder="例如：蓝牙耳机" />
          </div>

          <div>
            <label class="input-label">核心卖点</label>
            <textarea
              v-model="commerceForm.sellingPoints"
              class="input min-h-24 resize-none"
              placeholder="例如：主动降噪、长续航、轻便、通话清晰"
            />
          </div>

          <div class="grid grid-cols-2 gap-2">
            <div>
              <label class="input-label">平台</label>
              <select v-model="commerceForm.platform" class="input">
                <option v-for="item in commercePlatforms" :key="item" :value="item">{{ item }}</option>
              </select>
            </div>
            <div>
              <label class="input-label">风格</label>
              <select v-model="commerceForm.tone" class="input">
                <option v-for="item in commerceTones" :key="item" :value="item">{{ item }}</option>
              </select>
            </div>
          </div>

          <div>
            <label class="input-label">目标人群</label>
            <input v-model="commerceForm.audience" class="input" placeholder="例如：通勤上班族、学生、运动人群" />
          </div>

          <div>
            <label class="input-label">补充要求</label>
            <textarea
              v-model="commerceForm.extra"
              class="input min-h-20 resize-none"
              placeholder="例如：不要夸大承诺，语气自然，突出性价比"
            />
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">能力说明</h2>
        <div class="mt-3 space-y-3 text-sm text-gray-600 dark:text-dark-300">
          <div class="flex items-start gap-2">
            <Icon name="checkCircle" size="sm" class="mt-0.5 text-emerald-500" />
            <span>网页登录即可使用，不要求客户理解接口。</span>
          </div>
          <div class="flex items-start gap-2">
            <Icon name="checkCircle" size="sm" class="mt-0.5 text-emerald-500" />
            <span>聊天和电商文案都会走后台 Key、余额和日志。</span>
          </div>
          <div class="flex items-start gap-2">
            <Icon name="checkCircle" size="sm" class="mt-0.5 text-emerald-500" />
            <span>会话历史保存在当前浏览器，适合客户继续上次任务。</span>
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">下一步可用</h2>
        <div class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <router-link to="/image-studio" class="block rounded-lg bg-gray-50 px-3 py-2 hover:bg-primary-50 dark:bg-dark-700 dark:hover:bg-primary-900/30">
            商品作图和上传改图
          </router-link>
          <router-link to="/keys" class="block rounded-lg bg-gray-50 px-3 py-2 hover:bg-primary-50 dark:bg-dark-700 dark:hover:bg-primary-900/30">
            API Key 与第三方接入
          </router-link>
          <router-link to="/usage" class="block rounded-lg bg-gray-50 px-3 py-2 hover:bg-primary-50 dark:bg-dark-700 dark:hover:bg-primary-900/30">
            查看扣费和使用记录
          </router-link>
        </div>
      </section>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { useAuthStore } from '@/stores/auth'
import { useUserCapabilities } from '@/composables/useUserCapabilities'
import Icon from '@/components/icons/Icon.vue'

type WorkMode = 'chat' | 'commerce'
type ChatRole = 'user' | 'assistant'
type CommerceGoal = 'full_pack' | 'titles' | 'bullet_points' | 'detail_page' | 'xiaohongshu' | 'live_script' | 'customer_reply' | 'review_reply'

type ChatMessage = {
  id: string
  role: ChatRole
  content: string
}

type ChatSession = {
  id: string
  title: string
  mode: WorkMode
  model: string
  messages: ChatMessage[]
  updatedAt: number
}

type CommerceContext = {
  product_name: string
  selling_points: string
  platform: string
  tone: string
  audience: string
  output_goal: string
  extra: string
}

const STORAGE_KEY = 'ssxz.chat-studio.sessions.v1'

const route = useRoute()
const authStore = useAuthStore()
const {
  chatModels,
  errorMessage: capabilityError,
  hasChat,
  loaded,
  loadCapabilities
} = useUserCapabilities()

const models = computed(() => chatModels.value)

const quickPrompts = [
  {
    mode: 'chat',
    title: '电商文案',
    desc: '生成标题、卖点和详情页结构',
    prompt: '帮我为一款蓝牙耳机写电商详情页文案，突出降噪、长续航、轻便，语气专业、适合商用。'
  },
  {
    mode: 'chat',
    title: '客服回复',
    desc: '把客户问题改成专业答复',
    prompt: '请把下面这段客户咨询回复改得更专业、更有耐心：'
  },
  {
    mode: 'chat',
    title: '代码助手',
    desc: '解释报错或给出实现步骤',
    prompt: '我遇到了一个代码问题，请先帮我定位原因，再给出可执行的解决步骤：'
  },
  {
    mode: 'commerce',
    title: '小红书种草',
    desc: '切到小红书文案场景',
    prompt: 'xiaohongshu'
  },
  {
    mode: 'commerce',
    title: '详情页卖点',
    desc: '生成商品详情页结构',
    prompt: 'detail_page'
  },
  {
    mode: 'commerce',
    title: '差评解释',
    desc: '生成克制、专业的售后回复',
    prompt: 'review_reply'
  }
] as const

const commerceScenes: Array<{ value: CommerceGoal; label: string; desc: string }> = [
  { value: 'full_pack', label: '完整电商套装', desc: '标题、主图短句、详情页、小红书和直播口播一次生成。' },
  { value: 'titles', label: '商品标题', desc: '生成适合投放和搜索的标题方案。' },
  { value: 'bullet_points', label: '五点卖点', desc: '提炼 5 个清晰、有购买理由的核心卖点。' },
  { value: 'detail_page', label: '详情页文案', desc: '输出详情页模块标题和每屏卖点。' },
  { value: 'xiaohongshu', label: '小红书种草', desc: '输出自然口吻的种草笔记。' },
  { value: 'live_script', label: '直播口播', desc: '输出适合主播直接念的口播话术。' },
  { value: 'customer_reply', label: '客服回复', desc: '把客户问题变成专业、克制、有转化力的回复。' },
  { value: 'review_reply', label: '差评解释', desc: '生成售后解释、安抚和补救话术。' }
]

const commercePlatforms = ['淘宝/天猫', '拼多多', '抖音', '小红书', '跨境独立站']
const commerceTones = ['高转化', '种草风', '专业质感', '直播口播', '简洁高级']

const sessions = ref<ChatSession[]>([])
const activeSessionId = ref('')
const selectedModel = ref('gpt-5.5')
const mode = ref<WorkMode>(route.query.mode === 'ecommerce' ? 'commerce' : 'chat')
const draft = ref('')
const sending = ref(false)
const copied = ref(false)
const errorMessage = ref('')
const messageListRef = ref<HTMLElement | null>(null)

const commerceForm = reactive({
  productName: '',
  sellingPoints: '',
  platform: '小红书',
  tone: '种草风',
  audience: '',
  outputGoal: 'full_pack' as CommerceGoal,
  extra: ''
})

const activeSession = computed(() => sessions.value.find((item) => item.id === activeSessionId.value) || null)
const currentMessages = computed(() => activeSession.value?.messages || [])
const visibleQuickPrompts = computed(() => quickPrompts.filter((item) => item.mode === mode.value))
const activeModel = computed(() => models.value.find((item) => item.id === selectedModel.value))
const selectedCommerceScene = computed(() => commerceScenes.find((item) => item.value === commerceForm.outputGoal))
const workspaceTitle = computed(() => mode.value === 'commerce' ? '电商文案工作台' : activeModel.value?.name || 'AI 对话')
const emptyTitle = computed(() => mode.value === 'commerce' ? '填写右侧表单，生成商用文案' : '直接输入问题开始')
const emptyDescription = computed(() => mode.value === 'commerce'
  ? '客户只需要提供商品、卖点和平台，后台会自动补全电商提示词。'
  : '适合日常问答、写作、翻译、总结和代码问题。会话会自动保存在当前浏览器。')
const canGenerateCommerce = computed(() => commerceForm.productName.trim() !== '' || commerceForm.sellingPoints.trim() !== '')
const canRegenerate = computed(() => currentMessages.value.some((message) => message.role === 'user'))
const balanceText = computed(() => {
  const balance = authStore.user?.balance
  if (typeof balance !== 'number') return '--'
  return balance.toFixed(2)
})

onMounted(() => {
  loadCapabilities()
  loadSessions()
  if (!sessions.value.length) {
    createSession(mode.value)
  } else if (!activeSessionId.value) {
    activateSession(sessions.value[0].id)
  }
})

watch(() => route.query.mode, (value) => {
  if (value === 'ecommerce' && mode.value !== 'commerce') {
    switchMode('commerce')
  }
})

watch([selectedModel, mode], () => {
  if (!activeSession.value) return
  activeSession.value.model = selectedModel.value
  activeSession.value.mode = mode.value
  touchSession()
})

watch(chatModels, (nextModels) => {
  if (!nextModels.some((model) => model.id === selectedModel.value)) {
    selectedModel.value = nextModels[0]?.id || 'gpt-5.5'
  }
}, { immediate: true })

function loadSessions() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : null
    if (Array.isArray(parsed?.sessions)) {
      sessions.value = parsed.sessions
        .filter((item: ChatSession) => item && item.id && Array.isArray(item.messages))
        .slice(0, 20)
    }
    if (typeof parsed?.activeSessionId === 'string') {
      activeSessionId.value = parsed.activeSessionId
    }
  } catch {
    sessions.value = []
    activeSessionId.value = ''
  }
}

function saveSessions() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({
    activeSessionId: activeSessionId.value,
    sessions: sessions.value.slice(0, 20)
  }))
}

function createSession(nextMode: WorkMode = 'chat') {
  const session: ChatSession = {
    id: crypto.randomUUID(),
    title: nextMode === 'commerce' ? '新的电商文案' : '新的聊天',
    mode: nextMode,
    model: selectedModel.value,
    messages: [],
    updatedAt: Date.now()
  }
  sessions.value.unshift(session)
  activeSessionId.value = session.id
  mode.value = nextMode
  selectedModel.value = session.model
  errorMessage.value = ''
  saveSessions()
}

function activateSession(id: string) {
  const session = sessions.value.find((item) => item.id === id)
  if (!session) return
  activeSessionId.value = id
  mode.value = session.mode
  selectedModel.value = session.model || 'gpt-5.5'
  errorMessage.value = ''
  saveSessions()
}

function switchMode(nextMode: WorkMode) {
  mode.value = nextMode
  if (!activeSession.value || currentMessages.value.length > 0) {
    createSession(nextMode)
    return
  }
  activeSession.value.mode = nextMode
  activeSession.value.title = nextMode === 'commerce' ? '新的电商文案' : '新的聊天'
  touchSession()
}

function clearAllSessions() {
  sessions.value = []
  activeSessionId.value = ''
  createSession(mode.value)
}

function resetActiveSession() {
  if (!activeSession.value) return
  activeSession.value.messages = []
  activeSession.value.title = mode.value === 'commerce' ? '新的电商文案' : '新的聊天'
  errorMessage.value = ''
  touchSession()
}

function touchSession() {
  if (!activeSession.value) return
  activeSession.value.updatedAt = Date.now()
  const firstUser = activeSession.value.messages.find((message) => message.role === 'user')
  if (firstUser?.content) {
    activeSession.value.title = firstUser.content.replace(/\s+/g, ' ').slice(0, 28)
  }
  sessions.value = [
    activeSession.value,
    ...sessions.value.filter((item) => item.id !== activeSessionId.value)
  ].slice(0, 20)
  saveSessions()
}

function formatSessionTime(value: number) {
  if (!value) return '刚刚'
  const diff = Date.now() - value
  if (diff < 60_000) return '刚刚'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`
  return `${Math.floor(diff / 86_400_000)} 天前`
}

function applyPrompt(prompt: string) {
  if (mode.value === 'commerce') {
    const scene = commerceScenes.find((item) => item.value === prompt)
    if (scene) commerceForm.outputGoal = scene.value
    return
  }
  draft.value = prompt
}

async function sendChatMessage() {
  const content = draft.value.trim()
  if (!content || sending.value) return
  ensureActiveSession('chat')

  const userMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'user',
    content
  }
  activeSession.value?.messages.push(userMessage)
  draft.value = ''
  touchSession()
  await sendMessagesToGateway(activeSession.value?.messages || [], 'general')
}

async function sendCommerceMessage() {
  if (!canGenerateCommerce.value || sending.value) return
  ensureActiveSession('commerce')

  const context = buildCommerceContext()
  const userMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'user',
    content: buildCommerceVisibleRequest(context)
  }
  activeSession.value?.messages.push(userMessage)
  touchSession()
  await sendMessagesToGateway([userMessage], 'ecommerce_copy', context)
}

async function regenerateLast() {
  if (sending.value || !activeSession.value) return
  const messages = activeSession.value.messages
  const lastAssistantIndex = [...messages].reverse().findIndex((message) => message.role === 'assistant')
  if (lastAssistantIndex === 0) {
    messages.pop()
  }
  const lastUser = [...messages].reverse().find((message) => message.role === 'user')
  if (!lastUser) return
  touchSession()
  if (mode.value === 'commerce') {
    await sendMessagesToGateway([lastUser], 'ecommerce_copy', buildCommerceContext())
  } else {
    await sendMessagesToGateway(messages, 'general')
  }
}

function ensureActiveSession(nextMode: WorkMode) {
  if (!activeSession.value) {
    createSession(nextMode)
  }
  if (activeSession.value) {
    activeSession.value.mode = nextMode
    mode.value = nextMode
  }
}

async function sendMessagesToGateway(payloadMessages: ChatMessage[], requestMode: string, commerceContext?: CommerceContext) {
  errorMessage.value = ''
  sending.value = true
  copied.value = false
  await scrollToBottom()

  try {
    const response = await requestChatCompletion(payloadMessages, requestMode, commerceContext)
    activeSession.value?.messages.push({
      id: crypto.randomUUID(),
      role: 'assistant',
      content: extractAssistantText(response)
    })
    touchSession()
    await authStore.refreshUser()
  } catch (error) {
    console.error(error)
    errorMessage.value = error instanceof Error ? error.message : '请求失败，请稍后重试'
  } finally {
    sending.value = false
    await scrollToBottom()
  }
}

async function requestChatCompletion(payloadMessages: ChatMessage[], requestMode: string, commerceContext?: CommerceContext) {
  const token = localStorage.getItem('auth_token')
  const payload = {
    model: selectedModel.value,
    mode: requestMode,
    commerce_context: commerceContext,
    messages: payloadMessages.map((message) => ({
      role: message.role,
      content: message.content
    })),
    temperature: requestMode === 'ecommerce_copy' ? 0.85 : 0.7
  }
  const response = await fetch('/api/v1/chat-studio/complete', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify(payload)
  })
  const data = await response.json().catch(() => null)
  if (!response.ok) {
    const message = data?.error?.message || data?.message || data?.detail || `请求失败 (${response.status})`
    throw new Error(message)
  }
  return data
}

function buildCommerceContext(): CommerceContext {
  return {
    product_name: commerceForm.productName.trim(),
    selling_points: commerceForm.sellingPoints.trim(),
    platform: commerceForm.platform,
    tone: commerceForm.tone,
    audience: commerceForm.audience.trim(),
    output_goal: commerceForm.outputGoal,
    extra: commerceForm.extra.trim()
  }
}

function buildCommerceVisibleRequest(context: CommerceContext) {
  const lines = [
    `场景：${outputGoalLabel(context.output_goal)}`,
    `商品：${context.product_name || '未填写'}`,
    `卖点：${context.selling_points || '未填写'}`,
    `平台：${context.platform}`,
    `风格：${context.tone}`,
    `人群：${context.audience || '未填写'}`
  ]
  if (context.extra) lines.push(`补充：${context.extra}`)
  return lines.join('\n')
}

function outputGoalLabel(value: string) {
  return commerceScenes.find((item) => item.value === value)?.label || '完整电商套装'
}

function extractAssistantText(payload: any): string {
  const content = payload?.choices?.[0]?.message?.content
  if (typeof content === 'string' && content.trim()) {
    return content
  }
  if (Array.isArray(content)) {
    const text = content
      .map((item) => item?.text || item?.content || '')
      .filter(Boolean)
      .join('\n')
      .trim()
    if (text) return text
  }
  return '已收到回复，但返回内容无法在当前页面展示。'
}

function renderMarkdown(content: string) {
  const html = marked.parse(content, { async: false }) as string
  return DOMPurify.sanitize(html)
}

async function copyText(content: string) {
  await navigator.clipboard.writeText(content)
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1600)
}

async function scrollToBottom() {
  await nextTick()
  const el = messageListRef.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}
</script>

<style scoped>
.studio-markdown :deep(h1),
.studio-markdown :deep(h2),
.studio-markdown :deep(h3) {
  margin: 0.75rem 0 0.35rem;
  font-weight: 700;
}

.studio-markdown :deep(p) {
  margin: 0.45rem 0;
}

.studio-markdown :deep(ul),
.studio-markdown :deep(ol) {
  margin: 0.5rem 0;
  padding-left: 1.25rem;
}

.studio-markdown :deep(li) {
  margin: 0.25rem 0;
}

.studio-markdown :deep(strong) {
  font-weight: 700;
}
</style>
