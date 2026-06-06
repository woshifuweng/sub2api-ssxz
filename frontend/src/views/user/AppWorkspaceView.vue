<template>
  <AppSectionShell
    :title="activeContent.shellTitle"
    :subtitle="activeContent.shellSubtitle"
    :eyebrow="activeContent.eyebrow"
    :icon="activeContent.icon"
  >
    <section v-if="isConversationWorkspace" class="chat-workspace" :data-section="activeSection">
      <div class="chat-statusbar" aria-label="Workspace status">
        <span class="status-pill">Static E-A shell</span>
        <span class="status-copy">One conversation for writing, image intent, references, and follow-up edits.</span>
      </div>

      <div class="conversation-frame">
        <section class="conversation-empty" aria-label="Start a conversation">
          <span v-if="activeSection === 'image'" class="route-note">
            The image entry is now folded into the same conversation composer.
          </span>
          <h1>What would you like to do today?</h1>
          <p>
            Ask for copy, describe an image, mention a ratio like 16:9, or add reference context in the same thread.
          </p>
        </section>

        <section class="conversation-stream" aria-label="Conversation preview">
          <article
            v-for="message in visibleMessages"
            :key="message.id"
            class="message-row"
            :data-role="message.role"
          >
            <span class="message-avatar">
              <Icon :name="message.icon" size="sm" />
            </span>
            <div class="message-bubble">
              <span class="message-label">{{ message.label }}</span>
              <p>{{ message.text }}</p>
            </div>
          </article>

          <article class="image-task-card" aria-label="Pending image task card">
            <div class="image-task-header">
              <span>
                <Icon name="sparkles" size="sm" />
                Image task draft
              </span>
              <strong>pending / not generated / not billed</strong>
            </div>
            <dl>
              <div v-for="item in imageTaskFields" :key="item.label">
                <dt>{{ item.label }}</dt>
                <dd>{{ item.value }}</dd>
              </div>
            </dl>
            <p>
              This card is a static placeholder inside the conversation. Real image creation, files, usage, and billing remain off.
            </p>
          </article>

          <article class="pending-result" aria-label="Pending result placeholder">
            <Icon name="terminal" size="sm" />
            <span>Image result placeholder. A future image result can appear here after real capability is connected.</span>
          </article>
        </section>
      </div>

      <section class="conversation-composer" aria-label="Conversation composer">
        <div class="composer-tools" aria-label="Optional tools">
          <button v-for="tool in composerTools" :key="tool.label" type="button" class="tool-chip">
            <Icon :name="tool.icon" size="xs" />
            {{ tool.label }}
          </button>
        </div>
        <div class="composer-box">
          <textarea
            v-model="draft"
            rows="2"
            placeholder="Ask anything, for example: Generate a 16:9 ecommerce hero image with a clean white background."
          />
          <button type="button" class="send-button" :disabled="!draft.trim()" @click="recordDraft" aria-label="Add local draft">
            <Icon name="arrowUp" size="sm" />
          </button>
        </div>
        <div class="prompt-chips" aria-label="Example prompts">
          <button
            v-for="prompt in promptChips"
            :key="prompt.label"
            type="button"
            @click="applyPrompt(prompt.prompt)"
          >
            {{ prompt.label }}
          </button>
        </div>
      </section>
    </section>

    <section v-else class="support-section" :data-section="activeSection">
      <div class="support-copy">
        <span class="support-pill">{{ activeContent.pill }}</span>
        <h1>{{ activeContent.heading }}</h1>
        <p>{{ activeContent.description }}</p>
      </div>
      <div class="support-grid">
        <article v-for="card in activeContent.cards" :key="card.title" class="support-card">
          <Icon :name="card.icon" size="sm" />
          <span>{{ card.kicker }}</span>
          <h3>{{ card.title }}</h3>
          <p>{{ card.description }}</p>
        </article>
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
  kicker: string
  description: string
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

interface ConversationMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  label: string
  text: string
  icon: IconName
}

interface ComposerTool {
  label: string
  icon: IconName
}

interface ImageTaskField {
  label: string
  value: string
}

const route = useRoute()
const draft = ref('')
const localMessages = ref<ConversationMessage[]>([])

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image', 'developer', 'billing', 'account']

const composerTools: ComposerTool[] = [
  { label: 'Reference', icon: 'sparkles' },
  { label: 'Image mode', icon: 'sparkles' },
  { label: 'Ratio', icon: 'terminal' },
  { label: 'Style', icon: 'chat' },
  { label: 'Use case', icon: 'terminal' },
  { label: 'Prompt help', icon: 'sparkles' }
]

const promptChips = [
  {
    label: '16:9 ecommerce hero',
    prompt: 'Generate a 16:9 ecommerce hero image with a clean white background and a clear product subject.'
  },
  {
    label: 'Write product copy',
    prompt: 'Write concise product detail copy with a clear benefit, trusted tone, and easy-to-scan bullets.'
  },
  {
    label: 'Refine image direction',
    prompt: 'Make the last image direction feel more premium and suitable for a social cover.'
  }
]

const baseMessages: ConversationMessage[] = [
  {
    id: 'user-image-request',
    role: 'user',
    label: 'You',
    text: 'Generate a 16:9 ecommerce hero image with a clean white background and a clear product subject.',
    icon: 'chat'
  },
  {
    id: 'assistant-understanding',
    role: 'assistant',
    label: 'Assistant',
    text: 'I understand this as an image task draft. I can keep it in this conversation as pending, not generated, and not billed.',
    icon: 'sparkles'
  },
  {
    id: 'reference-placeholder',
    role: 'system',
    label: 'Reference placeholder',
    text: 'A reference image can stay attached to this thread later. No file is selected or uploaded in this static shell.',
    icon: 'terminal'
  }
]

const imageTaskFields: ImageTaskField[] = [
  { label: 'Task', value: 'Image draft' },
  { label: 'Ratio', value: '16:9, understood from natural language' },
  { label: 'Use case', value: 'Ecommerce hero / product detail lead image' },
  { label: 'Style', value: 'Clean white background, clear subject, premium but practical' },
  { label: 'State', value: 'pending / not generated / not billed' }
]

const sectionContent: Record<SectionKey, SectionContent> = {
  home: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: 'A single conversation workspace for writing and image task drafts.',
    eyebrow: 'Conversation',
    icon: 'sparkles',
    pill: 'Workspace',
    heading: 'Start from the composer.',
    description: 'The main product surface is one conversation, not separate feature pages.',
    cards: []
  },
  chat: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: 'The chat route shares the same conversation workspace.',
    eyebrow: 'Conversation',
    icon: 'chat',
    pill: 'Chat',
    heading: 'The same conversation surface.',
    description: 'Text and image intent stay in one thread.',
    cards: []
  },
  image: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: 'Image requests are handled from the same composer.',
    eyebrow: 'Conversation',
    icon: 'sparkles',
    pill: 'Image compatibility',
    heading: 'Image work starts in chat.',
    description: 'Old image links now point back to the same conversation-first experience.',
    cards: []
  },
  developer: {
    shellTitle: 'Developer',
    shellSubtitle: 'A supporting area for future API setup.',
    eyebrow: 'Settings',
    icon: 'terminal',
    pill: 'Developer',
    heading: 'Developer API setup belongs in settings, not the main creation flow.',
    description: 'This static area keeps integration planning separate from the conversation workspace.',
    cards: [
      {
        title: 'Keys',
        kicker: 'Preview',
        description: 'A future place for key management guidance.',
        icon: 'terminal'
      },
      {
        title: 'Models',
        kicker: 'Guide',
        description: 'A future place for model access and limits.',
        icon: 'sparkles'
      }
    ]
  },
  billing: {
    shellTitle: 'Billing',
    shellSubtitle: 'A supporting area for balance and usage review.',
    eyebrow: 'Settings',
    icon: 'creditCard',
    pill: 'Billing',
    heading: 'Billing stays away from the main conversation flow.',
    description: 'This static area frames future account finance information without live actions.',
    cards: [
      {
        title: 'Balance',
        kicker: 'Preview',
        description: 'A future overview of account credit.',
        icon: 'creditCard'
      },
      {
        title: 'Usage',
        kicker: 'Preview',
        description: 'A future overview of usage records.',
        icon: 'chat'
      }
    ]
  },
  account: {
    shellTitle: 'Account',
    shellSubtitle: 'A supporting area for profile and preferences.',
    eyebrow: 'Settings',
    icon: 'userCircle',
    pill: 'Account',
    heading: 'Account settings stay simple.',
    description: 'This static area gives profile, preferences, and security settings a clear home.',
    cards: [
      {
        title: 'Profile',
        kicker: 'Settings',
        description: 'A future place for identity details.',
        icon: 'userCircle'
      },
      {
        title: 'Preferences',
        kicker: 'Settings',
        description: 'A future place for language, theme, and notifications.',
        icon: 'sparkles'
      }
    ]
  }
}

const activeSection = computed<SectionKey>(() => {
  const section = route.meta.appSection
  return isSectionKey(section) ? section : 'home'
})

const activeContent = computed(() => sectionContent[activeSection.value])
const isConversationWorkspace = computed(() => (
  activeSection.value === 'home' || activeSection.value === 'chat' || activeSection.value === 'image'
))
const visibleMessages = computed(() => [...baseMessages, ...localMessages.value])

function isSectionKey(value: unknown): value is SectionKey {
  return typeof value === 'string' && sectionKeys.includes(value as SectionKey)
}

function applyPrompt(prompt: string) {
  draft.value = prompt
}

function recordDraft() {
  const content = draft.value.trim()
  if (!content) return

  localMessages.value.push({
    id: `local-${Date.now()}`,
    role: 'user',
    label: 'Local draft',
    text: content,
    icon: 'chat'
  })
  draft.value = ''
}
</script>

<style scoped>
:deep(.ssxz-page-heading) {
  padding-bottom: 0.75rem;
}

:deep(.ssxz-page-heading h2) {
  font-size: 1rem;
}

:deep(.ssxz-page-heading p) {
  max-width: 34rem;
}

.chat-workspace {
  display: grid;
  min-height: calc(100vh - 9rem);
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 1rem;
}

.chat-statusbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.65rem;
  color: var(--ssxz-body);
  font-size: 0.8rem;
}

.status-pill,
.route-note,
.message-label,
.image-task-header span,
.support-pill,
.support-card span {
  color: var(--ssxz-primary);
  font-size: 0.72rem;
  font-weight: 760;
}

.status-pill,
.route-note {
  display: inline-flex;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  padding: 0.35rem 0.62rem;
}

.conversation-frame {
  display: grid;
  gap: 1rem;
  align-content: start;
}

.conversation-empty {
  display: grid;
  justify-items: center;
  gap: 0.6rem;
  margin: 1rem auto 0;
  max-width: 42rem;
  text-align: center;
}

.conversation-empty h1 {
  color: var(--ssxz-text);
  font-size: clamp(2rem, 4vw, 3.4rem);
  font-weight: 760;
  line-height: 1.05;
}

.conversation-empty p {
  max-width: 35rem;
  color: var(--ssxz-body);
  font-size: 0.95rem;
  line-height: 1.7;
}

.conversation-stream {
  display: grid;
  gap: 0.75rem;
  margin: 0 auto;
  width: min(100%, 46rem);
}

.message-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.75rem;
}

.message-row[data-role='user'] {
  margin-left: clamp(0rem, 8vw, 5rem);
}

.message-avatar {
  display: inline-flex;
  height: 2.25rem;
  width: 2.25rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-primary);
}

.message-bubble,
.image-task-card,
.pending-result,
.composer-box,
.support-card {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-sm);
}

.message-bubble {
  border-radius: 1.15rem;
  padding: 0.85rem 1rem;
}

.message-bubble p,
.image-task-card p,
.pending-result,
.support-copy p,
.support-card p {
  color: var(--ssxz-body);
  font-size: 0.88rem;
  line-height: 1.65;
}

.image-task-card {
  display: grid;
  gap: 0.85rem;
  border-color: color-mix(in srgb, var(--ssxz-primary) 34%, var(--ssxz-border));
  border-radius: 1.2rem;
  margin-left: 3rem;
  padding: 1rem;
}

.image-task-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 0.75rem;
}

.image-task-header span {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
}

.image-task-header strong {
  color: var(--ssxz-text);
  font-size: 0.76rem;
  font-weight: 760;
}

.image-task-card dl {
  display: grid;
  gap: 0.55rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.image-task-card div {
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-muted);
  padding: 0.65rem;
}

.image-task-card dt {
  color: var(--ssxz-primary);
  font-size: 0.7rem;
  font-weight: 760;
}

.image-task-card dd {
  margin: 0.22rem 0 0;
  color: var(--ssxz-text);
  font-size: 0.83rem;
  line-height: 1.45;
}

.pending-result {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  border-radius: 1rem;
  margin-left: 3rem;
  padding: 0.85rem 1rem;
}

.pending-result svg {
  color: var(--ssxz-primary);
}

.conversation-composer {
  position: sticky;
  bottom: 1rem;
  display: grid;
  gap: 0.55rem;
  margin: 0 auto;
  width: min(100%, 48rem);
  border: 1px solid var(--ssxz-border);
  border-radius: 1.35rem;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 94%, transparent);
  box-shadow: var(--ssxz-shadow);
  padding: 0.75rem;
  backdrop-filter: blur(18px);
}

.composer-tools,
.prompt-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.tool-chip,
.prompt-chips button {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  font-size: 0.76rem;
  font-weight: 680;
  padding: 0.38rem 0.64rem;
}

.tool-chip svg {
  color: var(--ssxz-primary);
}

.composer-box {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 0.65rem;
  border-radius: 1.05rem;
  padding: 0.7rem;
}

.composer-box textarea {
  min-height: 3.2rem;
  max-height: 9rem;
  resize: vertical;
  border: 0;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 0.98rem;
  line-height: 1.55;
  outline: none;
}

.composer-box textarea::placeholder {
  color: var(--ssxz-subtle);
}

.send-button {
  display: inline-flex;
  height: 2.4rem;
  width: 2.4rem;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 9999px;
  background: var(--ssxz-primary);
  color: white;
}

.send-button:disabled {
  cursor: not-allowed;
  opacity: 0.46;
}

.support-section {
  display: grid;
  gap: 1rem;
}

.support-copy {
  max-width: 42rem;
}

.support-copy h1 {
  margin-top: 0.85rem;
  color: var(--ssxz-text);
  font-size: clamp(1.8rem, 3vw, 2.6rem);
  font-weight: 760;
  line-height: 1.12;
}

.support-copy p {
  margin-top: 0.7rem;
}

.support-grid {
  display: grid;
  gap: 0.85rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.support-card {
  display: grid;
  gap: 0.55rem;
  border-radius: 1rem;
  padding: 1rem;
}

.support-card svg {
  color: var(--ssxz-primary);
}

.support-card h3 {
  color: var(--ssxz-text);
  font-size: 1rem;
  font-weight: 760;
}

@media (max-width: 720px) {
  .chat-workspace {
    min-height: calc(100vh - 8rem);
  }

  .message-row,
  .message-row[data-role='user'] {
    grid-template-columns: 1fr;
    margin-left: 0;
  }

  .image-task-card,
  .pending-result {
    margin-left: 0;
  }

  .image-task-card dl,
  .support-grid {
    grid-template-columns: 1fr;
  }

  .conversation-composer {
    bottom: 0.5rem;
    border-radius: 1rem;
  }
}
</style>
