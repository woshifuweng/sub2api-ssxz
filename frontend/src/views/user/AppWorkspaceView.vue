<template>
  <AppSectionShell
    :title="activeContent.shellTitle"
    :subtitle="activeContent.shellSubtitle"
    :eyebrow="activeContent.eyebrow"
    :icon="activeContent.icon"
  >
    <section v-if="isUnifiedWorkspace" class="minimal-workspace" :data-section="activeSection">
      <p v-if="activeSection === 'image'" class="image-route-note">
        图片生成已合并到对话输入框，直接描述你想要的图片即可。
      </p>

      <div class="conversation-area" :class="{ 'has-local-draft': localDrafts.length > 0 }">
        <section v-if="localDrafts.length === 0" class="empty-state" aria-label="开始对话">
          <h1>今天想做什么？</h1>
          <p>直接输入问题，也可以描述想生成或修改的图片。</p>
        </section>

        <section v-else class="local-thread" aria-label="本地对话草稿">
          <article v-for="item in localDrafts" :key="item.id" class="message-row">
            <span class="message-avatar">你</span>
            <div class="message-bubble">
              <p>{{ item.text }}</p>
            </div>
          </article>

          <article class="assistant-note">
            <span class="message-avatar assistant-avatar">
              <Icon name="sparkles" size="sm" />
            </span>
            <div class="message-bubble">
              <p>{{ latestDraftHint }}</p>
            </div>
          </article>
        </section>
      </div>

      <section class="composer-wrap" aria-label="对话输入框">
        <form class="composer" @submit.prevent="recordDraft">
          <span class="composer-tool" aria-label="附件入口待接入" title="上传参考图">
            <Icon name="plus" size="sm" />
          </span>
          <span class="composer-tool image-tool" aria-label="图片能力待接入" title="图片">
            <Icon name="sparkles" size="sm" />
            图片
          </span>
          <textarea
            v-model="draft"
            rows="1"
            placeholder="输入你的问题，或直接描述想生成的图片…"
          />
          <button class="send-button" type="submit" :disabled="!draft.trim()" aria-label="发送本地草稿">
            <Icon name="arrowUp" size="sm" />
          </button>
        </form>

        <div class="suggestions" aria-label="建议示例">
          <span v-for="item in suggestionChips" :key="item">{{ item }}</span>
        </div>
        <p class="composer-note">当前仅记录本地草稿，不会生成图片、上传文件或扣费。</p>
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
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']
type SectionKey = 'home' | 'chat' | 'image' | 'developer' | 'billing' | 'account'

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

interface LocalDraft {
  id: string
  text: string
}

const route = useRoute()
const draft = ref('')
const localDrafts = ref<LocalDraft[]>([])

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image', 'developer', 'billing', 'account']

const suggestionChips = [
  '生成一张 16:9 电商主图',
  '写一段商品详情文案',
  '优化这张参考图'
]

const sectionContent: Record<SectionKey, SectionContent> = {
  home: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '一个输入框完成聊天、写作和图片需求。',
    eyebrow: '对话',
    icon: 'sparkles',
    pill: '工作台',
    heading: '今天想做什么？',
    description: '这是统一对话入口，图片能力会从输入框附近逐步接入。',
    cards: []
  },
  chat: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '和 /app 相同的统一对话体验。',
    eyebrow: '对话',
    icon: 'chat',
    pill: '对话',
    heading: '继续对话',
    description: '聊天、写作和图片需求都从同一个输入框开始。',
    cards: []
  },
  image: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '图片能力已合并到统一输入框。',
    eyebrow: '对话',
    icon: 'sparkles',
    pill: '图片',
    heading: '直接描述你想要的图片',
    description: '无需进入独立作图页面，说出比例、用途和风格即可。',
    cards: []
  },
  developer: {
    shellTitle: '开发者',
    shellSubtitle: '辅助入口，不影响主对话工作台。',
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
  billing: {
    shellTitle: '账单',
    shellSubtitle: '辅助入口，不触发支付或扣费。',
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
    shellSubtitle: '辅助入口，保持普通用户设置心智。',
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

const activeContent = computed(() => sectionContent[activeSection.value])
const isUnifiedWorkspace = computed(() => (
  activeSection.value === 'home' || activeSection.value === 'chat' || activeSection.value === 'image'
))

const latestDraftHint = computed(() => {
  const latest = localDrafts.value[localDrafts.value.length - 1]?.text ?? ''
  const looksLikeImage = /图片|图|海报|封面|主图|参考图|16:9|4:5|1:1|风格/.test(latest)

  if (looksLikeImage) {
    return '我会把它理解为图片需求草稿：先识别比例、用途和风格；当前不会真实生成、上传或扣费。'
  }

  return '我已记录这条本地草稿；当前不会请求真实模型。'
})

function isSectionKey(value: unknown): value is SectionKey {
  return typeof value === 'string' && sectionKeys.includes(value as SectionKey)
}

function recordDraft() {
  const content = draft.value.trim()
  if (!content) return

  localDrafts.value.push({
    id: `local-${Date.now()}`,
    text: content
  })
  draft.value = ''
}
</script>

<style scoped>
:deep(.ssxz-page-heading) {
  display: none;
}

.minimal-workspace {
  display: grid;
  min-height: calc(100vh - 4rem);
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 1rem;
  padding: 0 clamp(0rem, 2vw, 1.5rem) 1.5rem;
}

.image-route-note {
  justify-self: center;
  max-width: 34rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  font-size: 0.82rem;
  line-height: 1.6;
  padding: 0.42rem 0.75rem;
  text-align: center;
}

.conversation-area {
  display: grid;
  align-items: center;
  justify-items: center;
  min-height: 24rem;
}

.conversation-area.has-local-draft {
  align-items: end;
  justify-items: stretch;
}

.empty-state {
  display: grid;
  justify-items: center;
  gap: 0.65rem;
  text-align: center;
}

.empty-state h1 {
  color: var(--ssxz-text);
  font-size: clamp(1.9rem, 4vw, 3rem);
  font-weight: 760;
  line-height: 1.1;
}

.empty-state p,
.composer-note,
.message-bubble p,
.support-section p {
  color: var(--ssxz-body);
  font-size: 0.9rem;
  line-height: 1.7;
}

.local-thread {
  display: grid;
  gap: 0.85rem;
  margin: 0 auto;
  width: min(100%, 44rem);
}

.message-row,
.assistant-note {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.7rem;
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
  font-size: 0.76rem;
  font-weight: 760;
}

.assistant-avatar {
  color: var(--ssxz-primary);
}

.message-bubble {
  border-radius: 1.1rem;
  background: var(--ssxz-surface-raised);
  padding: 0.85rem 1rem;
}

.composer-wrap {
  display: grid;
  gap: 0.55rem;
  margin: 0 auto;
  width: min(100%, 48rem);
}

.composer {
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr) auto;
  align-items: end;
  gap: 0.45rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1.4rem;
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
  padding: 0.7rem;
}

.composer-tool,
.send-button {
  display: inline-flex;
  height: 2.35rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
}

.composer-tool {
  min-width: 2.35rem;
  gap: 0.35rem;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  font-size: 0.82rem;
  padding: 0 0.7rem;
}

.image-tool {
  color: var(--ssxz-primary);
}

.composer textarea {
  min-height: 2.35rem;
  max-height: 8rem;
  resize: vertical;
  border: 0;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 0.98rem;
  line-height: 1.55;
  outline: none;
  padding: 0.35rem 0.2rem;
}

.composer textarea::placeholder {
  color: var(--ssxz-subtle);
}

.send-button {
  width: 2.35rem;
  border: 0;
  background: var(--ssxz-primary);
  color: white;
}

.send-button:disabled {
  cursor: default;
  opacity: 0.42;
}

.suggestions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.45rem;
}

.suggestions span {
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  font-size: 0.78rem;
  line-height: 1.4;
  padding: 0.36rem 0.65rem;
}

.composer-note {
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
  .minimal-workspace {
    min-height: calc(100vh - 3rem);
    padding: 0 0 1rem;
  }

  .composer {
    grid-template-columns: auto minmax(0, 1fr) auto;
  }

  .image-tool {
    display: none;
  }

  .message-row,
  .assistant-note {
    grid-template-columns: 1fr;
  }
}
</style>
