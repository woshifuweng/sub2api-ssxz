<template>
  <AppSectionShell
    :title="activeContent.shellTitle"
    :subtitle="activeContent.shellSubtitle"
    :eyebrow="activeContent.eyebrow"
    :icon="activeContent.icon"
  >
    <section v-if="isUnifiedWorkspace" class="workspace-page" :data-section="activeSection">
      <div class="workspace-main" :class="{ 'has-messages': messages.length > 0 }">
        <section v-if="messages.length === 0" class="empty-state" aria-label="开始对话">
          <h1>欢迎使用 SSXZ AI 工作台</h1>
          <p>输入问题或上传参考图，开始你的智能工作流。</p>
        </section>

        <section v-else class="message-list" aria-label="本地消息流">
          <article
            v-for="message in messages"
            :key="message.id"
            class="message-row"
            :data-role="message.role"
          >
            <span class="message-avatar">{{ message.role === 'user' ? '你' : 'AI' }}</span>
            <div class="message-bubble" :data-state="message.state">
              <p v-if="message.text">{{ message.text }}</p>
              <div v-if="message.attachments?.length" class="message-attachments" aria-label="消息图片">
                <img
                  v-for="attachment in message.attachments"
                  :key="attachment.id"
                  :src="attachment.url"
                  :alt="attachment.name"
                  :title="attachment.name"
                  @error="removeMessageAttachment(message.id, attachment.id)"
                />
              </div>
            </div>
          </article>
        </section>
      </div>

      <section class="composer-zone" aria-label="统一输入框">
        <form class="composer-card" @submit.prevent="submitDraft">
          <div v-if="imagePreviews.length" class="attachment-preview-list">
            <article v-for="image in imagePreviews" :key="image.id" class="attachment-preview-card">
              <img :src="image.url" alt="参考图" @error="removeImagePreview(image.id)" />
              <div class="attachment-copy">
                <span :title="image.name">{{ image.name }}</span>
                <small class="attachment-status">
                  <Icon name="checkCircle" size="xs" />
                  <span>已添加</span>
                  <span class="attachment-size">{{ image.sizeLabel }}</span>
                </small>
              </div>
              <button type="button" class="remove-preview" aria-label="移除图片" @click="removeImagePreview(image.id)">
                <Icon name="x" size="xs" />
              </button>
            </article>
            <button type="button" class="clear-previews" @click="clearImagePreviews">清空</button>
          </div>

          <textarea
            v-model="draft"
            rows="2"
            :placeholder="composerPlaceholder"
          />

          <div v-if="assetPanelOpen" id="workspace-asset-panel" class="asset-panel" aria-label="上传能力面板">
            <label class="asset-option is-ready" title="上传图片">
              <Icon name="upload" size="sm" />
              <span>
                <strong>图片</strong>
                <small>上传参考图</small>
              </span>
              <input
                class="asset-file-input"
                type="file"
                accept="image/*"
                multiple
                aria-label="上传图片"
                @change="handleImageSelect"
              />
            </label>
            <button
              v-for="item in disabledAssetOptions"
              :key="item.label"
              type="button"
              class="asset-option is-disabled"
              disabled
              title="即将开放"
              :aria-label="`${item.label}，即将开放`"
            >
              <Icon :name="item.icon" size="sm" />
              <span>
                <strong>{{ item.label }}</strong>
                <small>即将开放</small>
              </span>
            </button>
          </div>

          <div class="composer-tool-row" aria-label="输入辅助工具">
            <div class="model-selector">
              <button
                type="button"
                class="model-trigger"
                :disabled="!chatModels.length"
                :aria-expanded="modelMenuOpen"
                aria-controls="workspace-model-menu"
                @click="modelMenuOpen = !modelMenuOpen"
              >
                <span>{{ selectedModelLabel }}</span>
                <Icon name="chevronDown" size="xs" />
              </button>
              <div v-if="modelMenuOpen" id="workspace-model-menu" class="model-menu">
                <button
                  v-for="model in chatModels"
                  :key="model.id"
                  type="button"
                  class="model-option"
                  :class="{ 'is-selected': model.id === activeChatModel }"
                  @click="selectModel(model.id)"
                >
                  <span>{{ model.name || model.id }}</span>
                  <small>{{ getModelCapabilityLabel(model.id) }}</small>
                </button>
                <p v-if="chatModels.length === 0">暂无可用模型</p>
              </div>
            </div>
            <button
              type="button"
              class="plus-button composer-plus"
              title="添加内容"
              :aria-expanded="assetPanelOpen"
              aria-controls="workspace-asset-panel"
              @click="assetPanelOpen = !assetPanelOpen"
            >
              <span class="composer-plus-icon" aria-hidden="true">
                <Icon name="plus" size="sm" />
              </span>
            </button>
            <button
              v-for="item in disabledToolbarItems"
              :key="item.label"
              type="button"
              class="toolbar-tool is-disabled"
              disabled
              aria-disabled="true"
              title="即将开放"
            >
              <Icon :name="item.icon" size="xs" />
              <span>{{ item.label }}</span>
            </button>
            <button class="send-button" type="submit" :disabled="!canSubmit" aria-label="发送">
              <Icon name="arrowUp" size="sm" />
            </button>
          </div>
        </form>
      </section>
    </section>

  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import { apiClient } from '@/api/client'
import { useUserCapabilities } from '@/composables/useUserCapabilities'

type IconName = InstanceType<typeof Icon>['$props']['name']
type SectionKey = string
type MessageRole = 'user' | 'assistant'
type MessageState = 'loading' | 'error'

interface SectionContent {
  shellTitle: string
  shellSubtitle: string
  eyebrow: string
  icon: IconName
}

interface LocalMessage {
  id: string
  role: MessageRole
  text: string
  state?: MessageState
  attachments?: MessageAttachment[]
}

interface ImagePreview {
  id: string
  file: File
  name: string
  size: number
  sizeLabel: string
  url: string
}

interface MessageAttachment {
  id: string
  name: string
  url: string
  type: 'image'
}

interface ChatStudioPayloadMessage {
  role: MessageRole
  content: string
}

interface ChatStudioResponse {
  choices?: Array<{
    message?: {
      content?: string | Array<{ text?: string; content?: string }>
    }
  }>
}

const route = useRoute()
const draft = ref('')
const messages = ref<LocalMessage[]>([])
const imagePreviews = ref<ImagePreview[]>([])
const isSending = ref(false)
const assetPanelOpen = ref(false)
const modelMenuOpen = ref(false)
const selectedModelId = ref('')
const {
  chatModels,
  hasChat,
  loadCapabilities
} = useUserCapabilities()

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image']

const disabledAssetOptions: Array<{ label: string; icon: IconName }> = [
  { label: '文档', icon: 'document' },
  { label: '表格', icon: 'chart' },
  { label: '代码', icon: 'terminal' }
]

const disabledToolbarItems: Array<{ label: string; icon: IconName }> = [
  { label: '联网', icon: 'globe' },
  { label: '记忆', icon: 'lightbulb' },
  { label: '工具箱', icon: 'grid' }
]

const sectionContent: Record<string, SectionContent> = {
  home: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '统一对话入口',
    eyebrow: '对话',
    icon: 'sparkles'
  },
  chat: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '统一对话入口',
    eyebrow: '对话',
    icon: 'chat'
  },
  image: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '图片需求也从对话输入框开始',
    eyebrow: '对话',
    icon: 'sparkles'
  }
}

const activeSection = computed<SectionKey>(() => {
  const section = route.meta.appSection
  return isSectionKey(section) ? section : 'home'
})

const activeContent = computed(() => sectionContent[activeSection.value] ?? sectionContent.home)
const isUnifiedWorkspace = computed(() => (
  activeSection.value === 'home' || activeSection.value === 'chat' || activeSection.value === 'image'
))
const composerPlaceholder = computed(() => (
  activeSection.value === 'image'
    ? '上传参考图，或直接描述你想生成/修改的图片……'
    : '输入你的问题，或上传图片后直接描述你想怎么处理……'
))
const canSubmit = computed(() => !isSending.value && (draft.value.trim().length > 0 || imagePreviews.value.length > 0))
const availableModelIds = computed(() => chatModels.value.map((model) => model.id))
const activeChatModel = computed(() => {
  if (!hasChat.value) return ''
  if (selectedModelId.value && availableModelIds.value.includes(selectedModelId.value)) return selectedModelId.value
  return availableModelIds.value[0] || ''
})
const canUseChatModel = computed(() => Boolean(activeChatModel.value))
const selectedModelLabel = computed(() => {
  const model = chatModels.value.find((item) => item.id === activeChatModel.value)
  return model?.name || activeChatModel.value || '暂无可用模型'
})

function isSectionKey(value: unknown): value is SectionKey {
  return typeof value === 'string' && sectionKeys.includes(value as SectionKey)
}

async function handleImageSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const files = Array.from(target.files ?? []).filter((file) => file.type.startsWith('image/'))
  if (!files.length) {
    target.value = ''
    return
  }

  const slots = Math.max(0, 4 - imagePreviews.value.length)
  const nextImages = (await Promise.all(
    files.slice(0, slots).map((file, index) => readImagePreview(file, index))
  )).filter((image): image is ImagePreview => Boolean(image))
  imagePreviews.value.push(...nextImages)
  assetPanelOpen.value = false
  target.value = ''
}

function readImagePreview(file: File, index: number): Promise<ImagePreview | null> {
  return new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = typeof reader.result === 'string' ? reader.result : ''
      resolve(result.startsWith('data:image/') ? {
        id: `${Date.now()}-${index}-${file.name}`,
        file,
        name: file.name,
        size: file.size,
        sizeLabel: formatFileSize(file.size),
        url: result
      } : null)
    }
    reader.onerror = () => resolve(null)
    reader.readAsDataURL(file)
  })
}

function removeImagePreview(id: string) {
  imagePreviews.value = imagePreviews.value.filter((item) => item.id !== id)
}

function clearImagePreviews() {
  imagePreviews.value = []
}

async function submitDraft() {
  const content = draft.value.trim()
  if (!content && imagePreviews.value.length === 0) return
  if (isSending.value) return

  const userText = content
  const attachments = imagePreviews.value.map(toMessageAttachment)
  messages.value.push({
    id: createMessageId('user'),
    role: 'user',
    text: userText,
    attachments
  })

  draft.value = ''
  clearImagePreviews()

  if (!userText && attachments.length > 0) {
    messages.value.push({
      id: createMessageId('assistant'),
      role: 'assistant',
      text: '请补充你想让我如何处理这张图片。'
    })
    return
  }

  const model = resolveUsableChatModel()
  if (!model) {
    messages.value.push({
      id: createMessageId('assistant'),
      role: 'assistant',
      text: '暂无可用模型，请稍后重试。',
      state: 'error'
    })
    return
  }

  const assistantMessage: LocalMessage = {
    id: createMessageId('assistant'),
    role: 'assistant',
    text: '正在思考...',
    state: 'loading'
  }

  messages.value.push(assistantMessage)

  isSending.value = true
  try {
    const response = await requestChatCompletion(buildChatPayloadMessages(), model)
    assistantMessage.text = extractAssistantText(response)
    delete assistantMessage.state
  } catch (error) {
    console.error('Workspace chat send failed', {
      endpoint: '/chat-studio/complete',
      status: getErrorStatus(error),
      message: getErrorMessage(error)
    })
    assistantMessage.text = '当前模型服务暂不可用，请稍后重试。'
    assistantMessage.state = 'error'
  } finally {
    isSending.value = false
  }
}

function toMessageAttachment(image: ImagePreview): MessageAttachment {
  return {
    id: image.id,
    name: image.name,
    url: image.url,
    type: 'image'
  }
}

function removeMessageAttachment(messageId: string, attachmentId: string) {
  const message = messages.value.find((item) => item.id === messageId)
  if (!message?.attachments) return
  message.attachments = message.attachments.filter((item) => item.id !== attachmentId)
}

function createMessageId(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function buildChatPayloadMessages(): ChatStudioPayloadMessage[] {
  return messages.value
    .filter((message) => !message.state && message.text.trim())
    .map((message) => ({
      role: message.role,
      content: message.text.trim()
    }))
    .slice(-40)
}

function resolveUsableChatModel() {
  if (!canUseChatModel.value) return ''
  return activeChatModel.value
}

function selectModel(modelId: string) {
  selectedModelId.value = modelId
  modelMenuOpen.value = false
}

function getModelCapabilityLabel(_modelId: string) {
  return '聊天'
}

function formatFileSize(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 KB'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  const precision = value >= 10 || unitIndex === 0 ? 0 : 1
  return `${value.toFixed(precision)} ${units[unitIndex]}`
}

async function requestChatCompletion(payloadMessages: ChatStudioPayloadMessage[], model: string) {
  const { data } = await apiClient.post<ChatStudioResponse>('/chat-studio/complete', {
    model,
    mode: 'general',
    messages: payloadMessages,
    temperature: 0.7
  })
  return data
}

function getErrorStatus(error: unknown) {
  if (typeof error !== 'object' || error === null) return undefined
  const value = error as { status?: number; response?: { status?: number } }
  return value.response?.status || value.status
}

function getErrorMessage(error: unknown) {
  if (error instanceof Error && error.message) return error.message
  if (typeof error !== 'object' || error === null) return 'unknown error'
  const value = error as { message?: unknown; response?: { data?: { message?: unknown; detail?: unknown; error?: { message?: unknown } } } }
  const data = value.response?.data
  const message = data?.error?.message || data?.message || data?.detail || value.message
  return typeof message === 'string' && message.trim() ? message : 'unknown error'
}

function extractAssistantText(payload: ChatStudioResponse): string {
  const content = payload?.choices?.[0]?.message?.content
  if (typeof content === 'string' && content.trim()) {
    return content.trim()
  }
  if (Array.isArray(content)) {
    const text = content
      .map((item) => item?.text || item?.content || '')
      .filter(Boolean)
      .join('\n')
      .trim()
    if (text) return text
  }
  return '已收到回复，但当前页面无法展示返回内容。'
}

onMounted(() => {
  loadCapabilities()
})

watch(chatModels, (models) => {
  if (!models.length) {
    selectedModelId.value = ''
    modelMenuOpen.value = false
    return
  }
  if (!models.some((model) => model.id === selectedModelId.value)) {
    selectedModelId.value = models[0].id
  }
}, { immediate: true })

onBeforeUnmount(() => {
  clearImagePreviews()
})
</script>

<style scoped>
:deep(.ssxz-page-heading) {
  display: none;
}

.workspace-page {
  display: grid;
  min-height: calc(100vh - 4rem);
  grid-template-rows: minmax(0, 1fr) auto;
  gap: 1.5rem;
  padding: 1rem 0 1.75rem;
}

.workspace-main {
  display: grid;
  align-items: center;
  justify-items: center;
  min-height: 24rem;
}

.workspace-main.has-messages {
  align-items: end;
}

.empty-state {
  display: grid;
  justify-items: center;
  gap: 0.65rem;
  max-width: 42rem;
  text-align: center;
}

.empty-state h1 {
  color: var(--ssxz-text);
  font-size: clamp(2rem, 4vw, 3rem);
  font-weight: 760;
  line-height: 1.1;
}

.empty-state p,
.message-bubble p {
  color: var(--ssxz-body);
  font-size: 0.95rem;
  line-height: 1.7;
}

.message-list {
  display: grid;
  gap: 0.9rem;
  width: min(100%, 46rem);
}

.message-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.7rem;
}

.message-row[data-role='user'] {
  margin-left: clamp(0rem, 10vw, 6rem);
}

.message-avatar {
  display: inline-flex;
  height: 2rem;
  width: 2rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text);
  font-size: 0.74rem;
  font-weight: 760;
}

.message-bubble {
  border-radius: 1.15rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-sm);
  padding: 0.9rem 1rem;
}

.message-attachments {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.65rem;
}

.message-attachments img {
  height: 5rem;
  width: 5rem;
  border-radius: 0.9rem;
  object-fit: cover;
}

.message-bubble[data-state='loading'] p {
  color: var(--ssxz-subtle);
}

.message-bubble[data-state='error'] {
  border: 1px solid color-mix(in srgb, var(--ssxz-danger, #dc2626) 32%, transparent);
  background: color-mix(in srgb, var(--ssxz-danger, #dc2626) 8%, var(--ssxz-surface-raised));
}

.composer-zone {
  display: grid;
  justify-items: center;
  gap: 0.7rem;
  margin: 0 auto;
  width: min(100%, 54rem);
}

.composer-card {
  display: grid;
  gap: 0.7rem;
  width: min(100%, 52rem);
  min-height: 6rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1.7rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
  padding: 0.8rem;
}

.attachment-preview-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.55rem;
}

.attachment-preview-card {
  display: grid;
  grid-template-columns: 3.75rem minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.55rem;
  width: min(100%, 18rem);
  min-height: 4.65rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-muted);
  padding: 0.45rem;
}

.attachment-preview-card img {
  height: 3.75rem;
  width: 3.75rem;
  border-radius: 0.8rem;
  object-fit: cover;
}

.attachment-copy {
  display: grid;
  min-width: 0;
  gap: 0.15rem;
}

.attachment-copy span {
  overflow: hidden;
  color: var(--ssxz-text);
  font-size: 0.86rem;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.attachment-status {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.28rem;
  color: color-mix(in srgb, #16a34a 82%, var(--ssxz-text));
  font-size: 0.72rem;
  font-weight: 760;
}

.attachment-status svg {
  flex: 0 0 auto;
}

.attachment-size {
  color: var(--ssxz-subtle);
  font-weight: 700;
}

.remove-preview {
  display: inline-flex;
  height: 1.8rem;
  width: 1.8rem;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-text) 9%, transparent);
  color: var(--ssxz-body);
}

.clear-previews {
  border: 0;
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-text) 8%, transparent);
  color: var(--ssxz-subtle);
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.35rem 0.62rem;
}

.composer-tool-row {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.45rem;
  overflow-x: auto;
  padding: 0 0.05rem;
}

.asset-panel {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.5rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1.1rem;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 78%, transparent);
  padding: 0.55rem;
}

.asset-option {
  position: relative;
  display: inline-flex;
  min-height: 3.2rem;
  align-items: center;
  gap: 0.55rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.9rem;
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-body);
  font-size: 0.82rem;
  line-height: 1.2;
  padding: 0.62rem 0.7rem;
  text-align: left;
}

.asset-option span {
  display: grid;
  gap: 0.18rem;
  min-width: 0;
}

.asset-option strong {
  color: var(--ssxz-text);
  font-size: 0.86rem;
}

.asset-option small {
  color: var(--ssxz-subtle);
  font-size: 0.72rem;
}

.asset-option.is-ready {
  cursor: pointer;
  overflow: hidden;
}

.asset-option.is-ready:hover {
  border-color: color-mix(in srgb, var(--ssxz-primary) 46%, var(--ssxz-border));
  background: color-mix(in srgb, var(--ssxz-primary) 7%, var(--ssxz-surface-raised));
}

.asset-file-input {
  position: absolute;
  inset: 0;
  z-index: 5;
  width: 100%;
  height: 100%;
  cursor: pointer;
  opacity: 0;
}

.asset-option.is-disabled {
  cursor: not-allowed;
  opacity: 0.56;
}

.toolbar-tool {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  gap: 0.32rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 80%, transparent);
  color: var(--ssxz-subtle);
  font-size: 0.78rem;
  font-weight: 740;
  padding: 0 0.68rem;
  white-space: nowrap;
}

.toolbar-tool.is-disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.model-selector {
  position: relative;
  min-width: 9rem;
}

.model-trigger {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  gap: 0.36rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 86%, transparent);
  color: var(--ssxz-text);
  font-size: 0.78rem;
  font-weight: 760;
  padding: 0 0.72rem;
}

.model-trigger:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.model-menu {
  position: absolute;
  bottom: calc(100% + 0.45rem);
  left: 0;
  z-index: 20;
  display: grid;
  gap: 0.25rem;
  min-width: 14rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
  padding: 0.45rem;
}

.model-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border: 0;
  border-radius: 0.75rem;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 0.82rem;
  font-weight: 740;
  padding: 0.55rem 0.62rem;
  text-align: left;
}

.model-option:hover,
.model-option.is-selected {
  background: color-mix(in srgb, var(--ssxz-primary) 10%, transparent);
}

.model-option small,
.model-menu p {
  color: var(--ssxz-subtle);
  font-size: 0.72rem;
}

.plus-button,
.send-button {
  display: inline-flex;
  height: 2.55rem;
  width: 2.55rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
}

.plus-button {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text);
  cursor: pointer;
}

.composer-plus-icon {
  position: relative;
  z-index: 1;
  display: inline-flex;
}

.composer-card textarea {
  min-height: 4.6rem;
  max-height: 8rem;
  resize: vertical;
  border: 0;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 1rem;
  line-height: 1.55;
  outline: none;
  padding: 0.4rem 0.2rem;
}

.composer-card textarea::placeholder {
  color: var(--ssxz-subtle);
}

.send-button {
  border: 0;
  background: var(--ssxz-primary);
  color: white;
  margin-left: auto;
}

.send-button:disabled {
  cursor: default;
  opacity: 0.42;
}

@media (max-width: 720px) {
  .workspace-page {
    min-height: calc(100vh - 3rem);
    padding: 0 0 1rem;
  }

  .message-row,
  .message-row[data-role='user'] {
    grid-template-columns: 1fr;
    margin-left: 0;
  }

  .composer-card {
    border-radius: 1.25rem;
  }

  .asset-panel {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
