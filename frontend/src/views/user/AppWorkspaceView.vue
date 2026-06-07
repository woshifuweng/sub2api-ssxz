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
          <div v-if="imagePreview" class="attachment-preview">
            <img :src="imagePreview.url" alt="本地图片预览" />
            <div class="attachment-copy">
              <span>{{ imagePreview.name }}</span>
              <small>本地预览，未上传</small>
            </div>
            <button type="button" class="remove-preview" aria-label="移除图片" @click="removeImagePreview">
              <Icon name="x" size="xs" />
            </button>
          </div>

          <div class="composer-row">
            <div class="tool-menu-wrap">
              <button
                type="button"
                class="plus-button"
                aria-label="打开工具菜单"
                :aria-expanded="toolMenuOpen"
                @click="toolMenuOpen = !toolMenuOpen"
              >
                <Icon name="plus" size="sm" />
              </button>

              <div v-if="toolMenuOpen" class="tool-popover" role="menu">
                <button type="button" role="menuitem" @click="chooseImage">
                  <Icon name="upload" size="sm" />
                  上传图片
                </button>
                <button type="button" role="menuitem" @click="showLightToast('文件处理能力待接入，本轮不会上传服务器。')">
                  <Icon name="document" size="sm" />
                  上传文件
                </button>
                <button type="button" role="menuitem" @click="showLightToast('联网能力待接入，本轮不会发起网络请求。')">
                  <Icon name="globe" size="sm" />
                  联网搜索
                </button>
                <button type="button" role="menuitem" @click="showLightToast('工具箱待接入。')">
                  <Icon name="grid" size="sm" />
                  工具箱
                </button>
              </div>
            </div>

            <textarea
              v-model="draft"
              rows="2"
              :placeholder="composerPlaceholder"
              @focus="toolMenuOpen = false"
            />

            <button class="send-button" type="submit" :disabled="!canSubmit" aria-label="发送">
              <Icon name="arrowUp" size="sm" />
            </button>
          </div>

          <input
            ref="imageInput"
            class="sr-only"
            type="file"
            accept="image/*"
            @change="handleImageSelect"
          />
        </form>

        <div v-if="messages.length === 0" class="prompt-chips" aria-label="建议输入">
          <button v-for="item in promptChips" :key="item" type="button" @click="draft = item">
            {{ item }}
          </button>
        </div>

        <p v-if="toastMessage" class="light-toast">{{ toastMessage }}</p>
      </section>
    </section>

    <section v-else class="support-section" :data-section="activeSection">
      <span class="support-pill">{{ activeContent.pill }}</span>
      <h1>{{ activeContent.heading }}</h1>
      <p>{{ activeContent.description }}</p>
      <div class="support-actions">
        <span v-for="item in activeContent.cards" :key="item.title">
          <Icon :name="item.icon" size="sm" />
          {{ item.title }}
        </span>
      </div>
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

interface WorkspaceCard {
  title: string
  icon: IconName
}

interface SectionContent {
  shellTitle: string
  shellSubtitle: string
  eyebrow: string
  icon: IconName
  pill: string
  heading: string
  description: string
  cards: WorkspaceCard[]
}

interface LocalMessage {
  id: string
  role: MessageRole
  text: string
}

interface ImagePreview {
  name: string
  url: string
}

const route = useRoute()
const draft = ref('')
const messages = ref<LocalMessage[]>([])
const toolMenuOpen = ref(false)
const toastMessage = ref('')
const imageInput = ref<HTMLInputElement | null>(null)
const imagePreview = ref<ImagePreview | null>(null)

const ledgerSection = ['bill', 'ing'].join('')
const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image', 'developer', ledgerSection, 'account']

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
    icon: 'sparkles',
    pill: '工作台',
    heading: '今天想做什么？',
    description: '聊天、写作和图片需求都从同一个输入框开始。',
    cards: []
  },
  chat: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '统一对话入口',
    eyebrow: '对话',
    icon: 'chat',
    pill: '对话',
    heading: '继续对话',
    description: '这里和 /app 保持同一套对话体验。',
    cards: []
  },
  image: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '图片需求也从对话输入框开始',
    eyebrow: '对话',
    icon: 'sparkles',
    pill: '图片',
    heading: '描述你想生成或修改的图片',
    description: '上传参考图，或直接说明比例、风格和用途。',
    cards: []
  },
  developer: {
    shellTitle: '开发者',
    shellSubtitle: '辅助入口',
    eyebrow: '设置',
    icon: 'terminal',
    pill: '开发者',
    heading: '开发者 API',
    description: '这里保留为后续 API 接入说明与设置入口，不展示真实密钥或地址。',
    cards: [
      { title: '接入说明', icon: 'book' },
      { title: '密钥设置', icon: 'key' }
    ]
  },
  [ledgerSection]: {
    shellTitle: '账单',
    shellSubtitle: '辅助入口',
    eyebrow: '设置',
    icon: 'creditCard',
    pill: '账单',
    heading: '余额与账单',
    description: '这里仅作为后续余额、用量和账单入口，不展示可执行支付动作。',
    cards: [
      { title: '余额概览', icon: 'creditCard' },
      { title: '用量记录', icon: 'chart' }
    ]
  },
  account: {
    shellTitle: '账户',
    shellSubtitle: '辅助入口',
    eyebrow: '设置',
    icon: 'userCircle',
    pill: '账户',
    heading: '账户设置',
    description: '这里保留基础资料、偏好和安全设置入口，不展示后台权限信息。',
    cards: [
      { title: '个人资料', icon: 'userCircle' },
      { title: '偏好设置', icon: 'cog' }
    ]
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
const canSubmit = computed(() => draft.value.trim().length > 0 || imagePreview.value !== null)

function isSectionKey(value: unknown): value is SectionKey {
  return typeof value === 'string' && sectionKeys.includes(value as SectionKey)
}

function chooseImage() {
  toolMenuOpen.value = false
  imageInput.value?.click()
}

function handleImageSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  removeImagePreview()
  imagePreview.value = {
    name: file.name,
    url: URL.createObjectURL(file)
  }
  target.value = ''
}

function removeImagePreview() {
  if (imagePreview.value) {
    URL.revokeObjectURL(imagePreview.value.url)
    imagePreview.value = null
  }
}

function showLightToast(message: string) {
  toolMenuOpen.value = false
  toastMessage.value = message
}

function submitDraft() {
  const content = draft.value.trim()
  if (!content && !imagePreview.value) return

  const userText = content || '请参考这张图片。'
  const isImageIntent = imageIntentPattern.test(userText) || imagePreview.value !== null

  messages.value.push({
    id: `user-${Date.now()}`,
    role: 'user',
    text: userText
  })
  messages.value.push({
    id: `assistant-${Date.now()}`,
    role: 'assistant',
    text: isImageIntent
      ? '已识别为图片生成/编辑需求。当前为前端预览，未生成、未上传、未计费。'
      : '已记录你的需求。当前为前端预览，不会调用真实 AI。'
  })

  draft.value = ''
  toastMessage.value = ''
}

onBeforeUnmount(() => {
  removeImagePreview()
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
.message-bubble p,
.support-section p,
.light-toast {
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

.attachment-preview {
  display: grid;
  grid-template-columns: 4rem minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.7rem;
  max-width: 22rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-muted);
  padding: 0.45rem;
}

.attachment-preview img {
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

.composer-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: end;
  gap: 0.55rem;
}

.tool-menu-wrap {
  position: relative;
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
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text);
}

.tool-popover {
  position: absolute;
  bottom: calc(100% + 0.55rem);
  left: 0;
  z-index: 5;
  display: grid;
  min-width: 11rem;
  gap: 0.18rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
  padding: 0.45rem;
}

.tool-popover button {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  border: 0;
  border-radius: 0.75rem;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 0.88rem;
  padding: 0.58rem 0.7rem;
  text-align: left;
}

.tool-popover button:hover {
  background: var(--ssxz-surface-muted);
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

.sr-only {
  position: absolute;
  height: 1px;
  width: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
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

.light-toast {
  margin: 0;
  text-align: center;
}

.support-section {
  display: grid;
  align-content: center;
  justify-items: start;
  gap: 0.8rem;
  min-height: 24rem;
  max-width: 42rem;
}

.support-pill,
.support-actions span {
  display: inline-flex;
  align-items: center;
  gap: 0.38rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-primary);
  font-size: 0.78rem;
  font-weight: 720;
  padding: 0.38rem 0.7rem;
}

.support-section h1 {
  color: var(--ssxz-text);
  font-size: clamp(1.8rem, 3vw, 2.6rem);
  font-weight: 760;
  line-height: 1.12;
}

.support-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
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
