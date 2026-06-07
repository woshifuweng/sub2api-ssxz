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
          <h1>今天想做什么？</h1>
          <p>直接输入问题，也可以上传图片后描述想怎么处理。</p>
        </section>

        <section v-else class="message-list" aria-label="本地消息流">
          <article
            v-for="message in messages"
            :key="message.id"
            class="message-row"
            :data-role="message.role"
          >
            <span class="message-avatar">{{ message.role === 'user' ? '你' : 'AI' }}</span>
            <div class="message-bubble">
              <p>{{ message.text }}</p>
            </div>
          </article>
        </section>
      </div>

      <section class="composer-zone" aria-label="统一输入框">
        <form class="composer-card" @submit.prevent="submitDraft">
          <div v-if="imagePreviews.length" class="attachment-preview-list">
            <article v-for="image in imagePreviews" :key="image.id" class="attachment-preview-card">
              <img :src="image.url" alt="本地图片预览" @error="removeImagePreview(image.id)" />
              <div class="attachment-copy">
                <span>{{ image.name }}</span>
                <small>本地预览</small>
              </div>
              <button type="button" class="remove-preview" aria-label="移除图片" @click="removeImagePreview(image.id)">
                <Icon name="x" size="xs" />
              </button>
            </article>
            <button type="button" class="clear-previews" @click="clearImagePreviews">清空</button>
          </div>

          <div class="composer-row">
            <div class="plus-button composer-plus" title="添加参考图">
              <span class="composer-plus-icon" aria-hidden="true">
                <Icon name="plus" size="sm" />
              </span>
              <input
                class="composer-file-input"
                type="file"
                accept="image/*"
                multiple
                aria-label="上传图片"
                @change="handleImageSelect"
              />
            </div>

            <textarea
              v-model="draft"
              rows="2"
              :placeholder="composerPlaceholder"
            />

            <button class="send-button" type="submit" :disabled="!canSubmit" aria-label="发送">
              <Icon name="arrowUp" size="sm" />
            </button>
          </div>
        </form>

        <div v-if="messages.length === 0" class="prompt-chips" aria-label="建议输入">
          <button v-for="item in promptChips" :key="item" type="button" @click="draft = item">
            {{ item }}
          </button>
        </div>
      </section>
    </section>

  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']
type SectionKey = string
type MessageRole = 'user' | 'assistant'

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
}

interface ImagePreview {
  id: string
  file: File
  name: string
  url: string
}

const route = useRoute()
const draft = ref('')
const messages = ref<LocalMessage[]>([])
const imagePreviews = ref<ImagePreview[]>([])

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image']

const imageIntentPattern = /生成图片|主图|16:9|海报|封面|改图|参考图|图片|白底|商品图|小红书/

const promptChips = [
  '生成 16:9 电商主图',
  '写商品详情文案',
  '上传参考图优化'
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
    ? '上传参考图，或直接描述你想生成/修改的图片...'
    : '输入你的问题，或上传图片后直接描述你想怎么处理...'
))
const canSubmit = computed(() => draft.value.trim().length > 0 || imagePreviews.value.length > 0)

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

function submitDraft() {
  const content = draft.value.trim()
  if (!content && imagePreviews.value.length === 0) return

  const imageCount = imagePreviews.value.length
  const userText = content || `请参考这 ${imageCount} 张图片。`
  const isImageIntent = imageIntentPattern.test(userText) || imageCount > 0

  messages.value.push({
    id: `user-${Date.now()}`,
    role: 'user',
    text: imageCount > 0 ? `${userText}（已添加 ${imageCount} 张本地图片）` : userText
  })
  messages.value.push({
    id: `assistant-${Date.now()}`,
    role: 'assistant',
    text: isImageIntent
      ? '演示预览：已识别为图片相关需求，尚未生成结果。'
      : '演示预览：已记录你的输入。'
  })

  draft.value = ''
  clearImagePreviews()
}

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
  grid-template-columns: 4rem minmax(0, 8rem) auto;
  align-items: center;
  gap: 0.55rem;
  max-width: 16rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-muted);
  padding: 0.45rem;
}

.attachment-preview-card img {
  height: 4rem;
  width: 4rem;
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

.attachment-copy small {
  color: var(--ssxz-subtle);
  font-size: 0.72rem;
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

.composer-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: end;
  gap: 0.55rem;
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

.composer-file-input {
  position: absolute;
  inset: 0;
  z-index: 5;
  width: 100%;
  height: 100%;
  cursor: pointer;
  opacity: 0;
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
}

.send-button:disabled {
  cursor: default;
  opacity: 0.42;
}

.prompt-chips {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.45rem;
}

.prompt-chips button {
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  font-size: 0.82rem;
  line-height: 1.4;
  padding: 0.38rem 0.7rem;
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
}
</style>
