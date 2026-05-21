<template>
  <div class="mx-auto grid max-w-7xl gap-5 xl:grid-cols-[280px_minmax(0,1fr)_300px]">
    <aside class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <div class="mb-4">
        <h1 class="text-lg font-semibold text-gray-900 dark:text-white">AI 聊天</h1>
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

        <div class="space-y-2">
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
      </div>
    </aside>

    <section class="flex min-h-[680px] flex-col rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <header class="flex items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-600">
        <div>
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ activeModel?.name }}</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">通过你的后台 Key 和余额计费，客户无需理解接口</p>
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
              <Icon name="chat" size="lg" />
            </div>
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">直接输入问题开始</h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">先把文字聊天做稳，后续再接入联网、文件和作图工作流。</p>
          </div>
        </div>

        <article
          v-for="message in messages"
          :key="message.id"
          class="flex"
          :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            class="max-w-[84%] whitespace-pre-wrap rounded-lg px-4 py-3 text-sm leading-6"
            :class="message.role === 'user'
              ? 'bg-primary-600 text-white'
              : 'border border-gray-200 bg-gray-50 text-gray-800 dark:border-dark-600 dark:bg-dark-900/60 dark:text-dark-100'"
          >
            {{ message.content }}
          </div>
        </article>

        <article v-if="sending" class="flex justify-start">
          <div class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-900/60 dark:text-dark-300">
            正在思考...
          </div>
        </article>
      </div>

      <footer class="border-t border-gray-200 p-4 dark:border-dark-600">
        <p v-if="errorMessage" class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
          {{ errorMessage }}
        </p>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-900/60">
          <textarea
            v-model="draft"
            class="min-h-24 w-full resize-none bg-transparent px-3 py-2 text-sm text-gray-900 outline-none placeholder:text-gray-400 dark:text-white"
            placeholder="输入你的问题，例如：帮我写一个蓝牙耳机详情页卖点文案"
            @keydown.enter.exact.prevent="sendMessage"
          />
          <div class="flex items-center justify-between gap-3 px-2 pb-1">
            <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
              <span class="rounded-md bg-gray-100 px-2 py-1 dark:bg-dark-700">{{ selectedModel }}</span>
              <span>Enter 发送，Shift+Enter 换行</span>
            </div>
            <button type="button" class="btn btn-primary" :disabled="sending || !draft.trim()" @click="sendMessage">
              <Icon v-if="sending" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="arrowUp" size="sm" />
              <span>{{ sending ? '发送中' : '发送' }}</span>
            </button>
          </div>
        </div>
      </footer>
    </section>

    <aside class="space-y-5">
      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">当前能力</h2>
        <div class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <div class="flex items-center gap-2">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            <span>网页聊天入口</span>
          </div>
          <div class="flex items-center gap-2">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            <span>自动选择兼容 Key</span>
          </div>
          <div class="flex items-center gap-2">
            <Icon name="checkCircle" size="sm" class="text-emerald-500" />
            <span>走 Sub2 余额计费</span>
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">下一步</h2>
        <div class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
          <p>接入图片上游后，作图工作台会从占位能力变成真实商业图片生成。</p>
          <router-link to="/image-studio" class="inline-flex text-primary-600 hover:text-primary-700 dark:text-primary-300">
            打开 AI 作图
          </router-link>
        </div>
      </section>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'

type ChatRole = 'user' | 'assistant'

type ChatMessage = {
  id: string
  role: ChatRole
  content: string
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

const authStore = useAuthStore()
const selectedModel = ref('gpt-5.5')
const draft = ref('')
const messages = ref<ChatMessage[]>([])
const sending = ref(false)
const errorMessage = ref('')
const messageListRef = ref<HTMLElement | null>(null)

const activeModel = computed(() => models.find((item) => item.id === selectedModel.value))
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

async function sendMessage() {
  const content = draft.value.trim()
  if (!content || sending.value) return

  const userMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'user',
    content
  }
  messages.value.push(userMessage)
  draft.value = ''
  errorMessage.value = ''
  sending.value = true
  await scrollToBottom()

  try {
    const response = await requestChatCompletion()
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

async function requestChatCompletion() {
  const token = localStorage.getItem('auth_token')
  const payload = {
    model: selectedModel.value,
    messages: messages.value.map((message) => ({
      role: message.role,
      content: message.content
    })),
    temperature: 0.7
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

async function scrollToBottom() {
  await nextTick()
  const el = messageListRef.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}
</script>
