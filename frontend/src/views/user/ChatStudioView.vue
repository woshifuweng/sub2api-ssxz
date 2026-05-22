<template>
  <div class="mx-auto grid max-w-7xl gap-5 xl:grid-cols-[320px_minmax(0,1fr)_280px]">
    <aside class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <div class="mb-4">
        <h1 class="text-lg font-semibold text-gray-900 dark:text-white">AI 工作台</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">余额约 {{ balanceText }}</p>
      </div>

      <div class="space-y-4">
        <div>
          <label class="input-label">模型</label>
          <select v-model="selectedModel" class="input">
            <option v-for="model in models" :key="model.id" :value="model.id">
              {{ model.name }}
            </option>
          </select>
        </div>

        <div>
          <label class="input-label">模式</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              class="rounded-lg border px-3 py-2 text-sm transition"
              :class="mode === 'chat'
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
                : 'border-gray-200 bg-gray-50 text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-200'"
              @click="mode = 'chat'"
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
              @click="mode = 'commerce'"
            >
              <Icon name="sparkles" size="sm" class="mx-auto mb-1" />
              电商文案
            </button>
          </div>
        </div>

        <div v-if="mode === 'chat'" class="space-y-2">
          <button
            v-for="item in quickPrompts"
            :key="item.title"
            type="button"
            class="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-left text-sm text-gray-700 transition hover:border-primary-400 hover:bg-primary-50 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-100 dark:hover:border-primary-500"
            @click="applyPrompt(item.prompt)"
          >
            <span class="block font-medium">{{ item.title }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ item.desc }}</span>
          </button>
        </div>

        <div v-else class="space-y-3">
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
            <label class="input-label">输出内容</label>
            <select v-model="commerceForm.outputGoal" class="input">
              <option value="full_pack">完整电商套装</option>
              <option value="titles">爆款标题</option>
              <option value="xiaohongshu">小红书种草笔记</option>
              <option value="live_script">直播口播话术</option>
              <option value="detail_page">详情页卖点模块</option>
            </select>
          </div>

          <div>
            <label class="input-label">补充要求</label>
            <textarea
              v-model="commerceForm.extra"
              class="input min-h-20 resize-none"
              placeholder="例如：不要夸张承诺，语气自然，突出性价比"
            />
          </div>

          <button
            type="button"
            class="btn btn-primary w-full"
            :disabled="sending || !canGenerateCommerce"
            @click="sendCommerceMessage"
          >
            <Icon v-if="sending" name="refresh" size="sm" class="animate-spin" />
            <Icon v-else name="sparkles" size="sm" />
            <span>{{ sending ? '生成中' : '生成商用文案' }}</span>
          </button>
        </div>
      </div>
    </aside>

    <section class="flex min-h-[720px] flex-col rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <header class="flex items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-600">
        <div>
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ workspaceTitle }}</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            通过后台 Key 和余额计费，客户只需要填写需求并点击生成。
          </p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="resetChat">
          <Icon name="trash" size="sm" />
          <span>清空</span>
        </button>
      </header>

      <div ref="messageListRef" class="flex-1 space-y-4 overflow-y-auto px-5 py-5">
        <div v-if="messages.length === 0" class="flex h-full min-h-[360px] items-center justify-center">
          <div class="max-w-md text-center">
            <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
              <Icon :name="mode === 'commerce' ? 'sparkles' : 'chat'" size="lg" />
            </div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ emptyTitle }}</h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ emptyDescription }}</p>
          </div>
        </div>

        <article
          v-for="message in messages"
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

      <footer v-if="mode === 'chat'" class="border-t border-gray-200 p-4 dark:border-dark-600">
        <p v-if="errorMessage" class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
          {{ errorMessage }}
        </p>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-900/60">
          <textarea
            v-model="draft"
            class="min-h-24 w-full resize-none bg-transparent px-3 py-2 text-sm text-gray-900 outline-none placeholder:text-gray-400 dark:text-white"
            placeholder="输入你的问题，例如：帮我写一个蓝牙耳机详情页卖点文案"
            @keydown.enter.exact.prevent="sendChatMessage"
          />
          <div class="flex items-center justify-between gap-3 px-2 pb-1">
            <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
              <span class="rounded-md bg-gray-100 px-2 py-1 dark:bg-dark-700">{{ selectedModel }}</span>
              <span>Enter 发送，Shift+Enter 换行</span>
            </div>
            <button type="button" class="btn btn-primary" :disabled="sending || !draft.trim()" @click="sendChatMessage">
              <Icon v-if="sending" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="arrowUp" size="sm" />
              <span>{{ sending ? '发送中' : '发送' }}</span>
            </button>
          </div>
        </div>
      </footer>

      <footer v-else class="border-t border-gray-200 p-4 dark:border-dark-600">
        <p v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
          {{ errorMessage }}
        </p>
        <div v-else class="flex items-center justify-between gap-3 text-xs text-gray-500 dark:text-dark-400">
          <span>当前使用 {{ selectedModel }}，由后台自动套用电商提示词模板。</span>
          <span v-if="copied">已复制</span>
        </div>
      </footer>
    </section>

    <aside class="space-y-5">
      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">已接入能力</h2>
        <div class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <div class="flex items-center gap-2">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            <span>网页登录使用</span>
          </div>
          <div class="flex items-center gap-2">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            <span>自动选择兼容 Key</span>
          </div>
          <div class="flex items-center gap-2">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            <span>走 Sub2 余额计费</span>
          </div>
          <div class="flex items-center gap-2">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            <span>隐藏电商 Prompt</span>
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">电商模板会输出</h2>
        <div class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <p>爆款标题、主图文案、详情页卖点、小红书笔记和直播口播会按照平台与风格自动组织。</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">这只是第一版入口，后续可以继续拆成制图、SKU、批量生成和图片下载。</p>
        </div>
      </section>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'

type ChatRole = 'user' | 'assistant'
type WorkMode = 'chat' | 'commerce'

type ChatMessage = {
  id: string
  role: ChatRole
  content: string
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

const models = [
  { id: 'gpt-5.5', name: 'GPT-5.5' },
  { id: 'gpt-5.4', name: 'GPT-5.4' },
  { id: 'gpt-5.2', name: 'GPT-5.2' },
  { id: 'gpt-5.4-mini', name: 'GPT-5.4 Mini' }
]

const quickPrompts = [
  {
    title: '电商文案',
    desc: '生成标题、卖点和详情页结构',
    prompt: '帮我为一款蓝牙耳机写电商详情页文案，突出降噪、长续航、轻便，语气专业、适合商用。'
  },
  {
    title: '客服回复',
    desc: '把客户问题改成专业答复',
    prompt: '请把下面这段客户咨询回复改得更专业、更有耐心：'
  },
  {
    title: '代码助手',
    desc: '解释报错或给出实现步骤',
    prompt: '我遇到了一个代码问题，请先帮我定位原因，再给出可执行的解决步骤：'
  }
]

const commercePlatforms = ['淘宝/天猫', '拼多多', '抖音', '小红书', '跨境独立站']
const commerceTones = ['高转化', '种草风', '专业质感', '直播口播', '简洁高级']

const authStore = useAuthStore()
const selectedModel = ref('gpt-5.5')
const mode = ref<WorkMode>('chat')
const draft = ref('')
const messages = ref<ChatMessage[]>([])
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
  outputGoal: 'full_pack',
  extra: ''
})

const activeModel = computed(() => models.find((item) => item.id === selectedModel.value))
const workspaceTitle = computed(() => mode.value === 'commerce' ? '电商文案工作台' : activeModel.value?.name || 'AI 对话')
const emptyTitle = computed(() => mode.value === 'commerce' ? '填写左侧表单，生成商用文案' : '直接输入问题开始')
const emptyDescription = computed(() => mode.value === 'commerce'
  ? '客户只需要提供商品和卖点，后台会自动补全平台化提示词。'
  : '先把网页登录对话做稳定，后续再继续接入联网、文件和图片工作流。')
const canGenerateCommerce = computed(() => commerceForm.productName.trim() !== '' || commerceForm.sellingPoints.trim() !== '')
const balanceText = computed(() => {
  const balance = authStore.user?.balance
  if (typeof balance !== 'number') return '--'
  return balance.toFixed(2)
})

function applyPrompt(prompt: string) {
  draft.value = prompt
}

function resetChat() {
  messages.value = []
  errorMessage.value = ''
}

async function sendChatMessage() {
  const content = draft.value.trim()
  if (!content || sending.value) return

  const userMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'user',
    content
  }
  messages.value.push(userMessage)
  draft.value = ''
  await sendMessagesToGateway(messages.value, 'general')
}

async function sendCommerceMessage() {
  if (!canGenerateCommerce.value || sending.value) return

  const context = buildCommerceContext()
  const userMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'user',
    content: buildCommerceVisibleRequest(context)
  }
  messages.value.push(userMessage)
  await sendMessagesToGateway([userMessage], 'ecommerce_copy', context)
}

async function sendMessagesToGateway(payloadMessages: ChatMessage[], requestMode: string, commerceContext?: CommerceContext) {
  errorMessage.value = ''
  sending.value = true
  copied.value = false
  await scrollToBottom()

  try {
    const response = await requestChatCompletion(payloadMessages, requestMode, commerceContext)
    messages.value.push({
      id: crypto.randomUUID(),
      role: 'assistant',
      content: extractAssistantText(response)
    })
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
    `商品：${context.product_name || '未填写'}`,
    `卖点：${context.selling_points || '未填写'}`,
    `平台：${context.platform}`,
    `风格：${context.tone}`,
    `人群：${context.audience || '未填写'}`,
    `输出：${outputGoalLabel(context.output_goal)}`
  ]
  if (context.extra) lines.push(`补充：${context.extra}`)
  return lines.join('\n')
}

function outputGoalLabel(value: string) {
  const labels: Record<string, string> = {
    full_pack: '完整电商套装',
    titles: '爆款标题',
    xiaohongshu: '小红书种草笔记',
    live_script: '直播口播话术',
    detail_page: '详情页卖点模块'
  }
  return labels[value] || '完整电商套装'
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
