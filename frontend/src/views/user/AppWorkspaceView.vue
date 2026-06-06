<template>
  <AppSectionShell
    :title="activeContent.shellTitle"
    :subtitle="activeContent.shellSubtitle"
    :eyebrow="activeContent.eyebrow"
    :icon="activeContent.icon"
  >
    <template v-if="showHeaderActions" #actions>
      <div class="app-workspace-actions">
        <RouterLink :to="activeContent.secondaryAction.to" class="ssxz-btn-secondary rounded-full">
          <Icon :name="activeContent.secondaryAction.icon" size="sm" />
          {{ activeContent.secondaryAction.label }}
        </RouterLink>
        <RouterLink :to="activeContent.primaryAction.to" class="ssxz-btn-primary rounded-full">
          <Icon :name="activeContent.primaryAction.icon" size="sm" />
          {{ activeContent.primaryAction.label }}
        </RouterLink>
      </div>
    </template>

    <section class="workspace-section-hero" :data-section="activeSection">
      <div class="workspace-section-copy">
        <div class="hero-pill">{{ activeContent.pill }}</div>
        <h1>{{ activeContent.heading }}</h1>
        <p>{{ activeContent.description }}</p>
      </div>

      <div v-if="isConversationWorkspace" class="workspace-composer unified-composer">
        <textarea
          v-model="draft"
          rows="4"
          placeholder="直接描述你想完成的事，例如：帮我生成一张 16:9 的电商主图，参考这张图改成小红书封面风格。"
        />
        <div class="composer-tool-grid" aria-label="Optional composer tools">
          <button v-for="tool in composerTools" :key="tool.label" type="button" class="composer-tool">
            <Icon :name="tool.icon" size="xs" />
            <span>
              <strong>{{ tool.label }}</strong>
              <small>{{ tool.description }}</small>
            </span>
          </button>
        </div>
        <div class="composer-footer">
          <div class="prompt-chip-row">
            <button
              v-for="prompt in promptChips"
              :key="prompt.label"
              type="button"
              class="prompt-chip"
              @click="applyPrompt(prompt.prompt)"
            >
              {{ prompt.label }}
            </button>
          </div>
          <button type="button" class="ssxz-btn-primary rounded-full" :disabled="!draft.trim()" @click="recordDraft">
            <Icon name="arrowUp" size="sm" />
            记录草稿
          </button>
        </div>
      </div>

      <div v-else class="workspace-status-panel">
        <span class="status-dot" />
        <span>{{ activeContent.status }}</span>
      </div>
    </section>

    <section v-if="isConversationWorkspace" class="unified-conversation-shell" aria-label="Unified conversation preview">
      <div class="conversation-canvas">
        <article v-for="item in conversationPreviewItems" :key="item.title" class="conversation-message-card" :data-kind="item.kind">
          <span class="conversation-avatar">
            <Icon :name="item.icon" size="sm" />
          </span>
          <span class="conversation-message-copy">
            <span class="conversation-message-kicker">{{ item.kicker }}</span>
            <strong>{{ item.title }}</strong>
            <small>{{ item.description }}</small>
          </span>
        </article>

        <article v-for="message in localMessages" :key="message.id" class="conversation-message-card" data-kind="draft">
          <span class="conversation-avatar">
            <Icon name="chat" size="sm" />
          </span>
          <span class="conversation-message-copy">
            <span class="conversation-message-kicker">{{ message.role }}</span>
            <strong>本地草稿已记录</strong>
            <small>{{ message.content }}</small>
          </span>
        </article>
      </div>

      <aside class="composer-intent-panel" aria-label="Parsed image intent preview">
        <span class="pending-badge">图片任务草稿</span>
        <h3>自然语言会先变成会话中的计划卡片</h3>
        <p>
          这里展示的是 E-A 静态壳：比例、用途、风格和参考图关系都来自输入框语义，保持待生成、未计费。
        </p>
        <ul>
          <li v-for="item in intentSummaryItems" :key="item.label">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </li>
        </ul>
        <p v-if="activeSection === 'image'" class="legacy-route-note">
          `/app/image` 仅作为兼容入口展示同一输入框能力；图片请求应回到当前会话中表达。
        </p>
      </aside>
    </section>

    <section v-if="activeSection === 'chat' && localMessages.length && !isConversationWorkspace" class="workspace-local-thread">
      <article v-for="message in localMessages" :key="message.id" class="local-message">
        <span class="local-message-role">{{ message.role }}</span>
        <p>{{ message.content }}</p>
      </article>
    </section>

    <section v-if="!isConversationWorkspace" class="workspace-section-grid" :aria-label="`${activeContent.label} workspace preview`">
      <article v-for="card in activeContent.cards" :key="card.title" class="workspace-section-card">
        <span class="workspace-section-card-icon">
          <Icon :name="card.icon" size="sm" />
        </span>
        <span class="workspace-section-card-kicker">{{ card.kicker }}</span>
        <h3>{{ card.title }}</h3>
        <p>{{ card.description }}</p>
      </article>
    </section>

    <section v-if="!isConversationWorkspace" class="workspace-detail-panel">
      <div>
        <span class="detail-kicker">{{ activeContent.detailKicker }}</span>
        <h2>{{ activeContent.detailTitle }}</h2>
        <p>{{ activeContent.detailDescription }}</p>
      </div>
      <ul>
        <li v-for="item in activeContent.checklist" :key="item">
          <span class="check-dot" />
          <span>{{ item }}</span>
        </li>
      </ul>
    </section>

    <section v-if="!isConversationWorkspace" class="workspace-note-strip">
      <span class="note-dot" />
      <span>
        This workspace is a guided preview: you can move between sections, draft ideas, and review the flow while live actions arrive in later batches.
      </span>
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

interface SectionCard {
  title: string
  kicker: string
  description: string
  icon: IconName
}

interface SectionContent {
  label: string
  shellTitle: string
  shellSubtitle: string
  eyebrow: string
  icon: IconName
  pill: string
  heading: string
  description: string
  status: string
  primaryAction: WorkspaceAction
  secondaryAction: WorkspaceAction
  detailKicker: string
  detailTitle: string
  detailDescription: string
  cards: SectionCard[]
  checklist: string[]
}

interface WorkspaceAction {
  label: string
  to: string
  icon: IconName
}

interface LocalMessage {
  id: number
  role: 'Draft'
  content: string
}

interface ComposerTool {
  label: string
  description: string
  icon: IconName
}

interface ConversationPreviewItem {
  kind: 'user' | 'assistant' | 'reference' | 'image-plan' | 'result'
  kicker: string
  title: string
  description: string
  icon: IconName
}

interface IntentSummaryItem {
  label: string
  value: string
}

const route = useRoute()
const draft = ref('')
const localMessages = ref<LocalMessage[]>([])

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image', 'developer', 'billing', 'account']

const promptChips = [
  { label: '商品详情文案', prompt: '帮我写一段适合详情页首屏的商品卖点文案，语气清晰、可信、适合普通用户阅读。' },
  { label: '16:9 电商主图', prompt: '帮我生成一张 16:9 的电商主图，白色背景，主体清楚，适合商品详情页首图。' },
  { label: '小红书封面', prompt: '参考这张图的氛围，规划一张小红书封面风格的图片任务草稿。' }
]

const composerTools: ComposerTool[] = [
  { label: '参考图', description: '占位壳，不处理文件', icon: 'sparkles' },
  { label: '图片模式', description: '从自然语言识别任务', icon: 'terminal' },
  { label: '比例', description: '如 16:9 / 4:5 / 1:1', icon: 'chat' },
  { label: '风格', description: '如小红书、海报、电商', icon: 'sparkles' },
  { label: '用途', description: '主图、封面、详情页', icon: 'terminal' },
  { label: '提示词优化', description: '后续批次接入', icon: 'chat' }
]

const conversationPreviewItems: ConversationPreviewItem[] = [
  {
    kind: 'user',
    kicker: 'User message',
    title: '帮我生成一张 16:9 的电商主图',
    description: '用户只在同一个输入框描述目标；比例和用途先作为静态语义被识别。',
    icon: 'chat'
  },
  {
    kind: 'reference',
    kicker: 'Reference placeholder',
    title: '参考图留在当前会话',
    description: '参考图能力显示为占位壳；本阶段不选择文件、不上传、不写入资产。',
    icon: 'sparkles'
  },
  {
    kind: 'image-plan',
    kicker: 'Image task draft',
    title: '图片任务草稿：16:9 / 电商主图 / 清爽商品风格',
    description: '计划卡片进入同一会话流，状态为 pending、not generated、not billed。',
    icon: 'terminal'
  },
  {
    kind: 'assistant',
    kicker: 'Assistant placeholder',
    title: '后续可继续追问或修改图片方向',
    description: '例如“改成 4:5”“背景换白色”“更像小红书封面”，仍留在同一 conversation。',
    icon: 'sparkles'
  }
]

const intentSummaryItems: IntentSummaryItem[] = [
  { label: '任务类型', value: '图片任务草稿' },
  { label: '比例', value: '从“16:9 / 4:5”等文字识别' },
  { label: '用途', value: '从“电商主图 / 小红书封面”等语义识别' },
  { label: '状态', value: 'pending / not generated / not billed' }
]

const sectionContent: Record<SectionKey, SectionContent> = {
  home: {
    label: 'Home',
    shellTitle: 'Unified AI Conversation',
    shellSubtitle: 'One conversation window for writing, reference images, and pending image task drafts.',
    eyebrow: 'SSXZ AI',
    icon: 'sparkles',
    pill: 'Unified conversation',
    heading: 'Start with one input. Let text and image tasks share the same thread.',
    description: 'Type a request naturally: write copy, attach a reference placeholder, ask for a 16:9 product hero, or revise an image direction without switching into a separate image page.',
    status: 'Preview mode: composer tools are static, image tasks stay pending, and no live capability is triggered.',
    primaryAction: { label: 'New conversation', to: '/app?new=1', icon: 'chat' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Unified IA',
    detailTitle: 'One composer owns text and image intent',
    detailDescription: 'The app starts from a single conversation surface. Image parameters are optional helpers near the composer, not a required form flow.',
    cards: [],
    checklist: [
      'Users do not choose chat versus image before typing.',
      'Image intent is represented as a pending card in the same conversation.',
      'Developer, billing, and account areas are supporting settings, not primary creation lanes.'
    ]
  },
  chat: {
    label: 'Chat',
    shellTitle: 'Unified AI Conversation',
    shellSubtitle: 'The chat compatibility route renders the same conversation workspace as /app.',
    eyebrow: 'SSXZ AI',
    icon: 'chat',
    pill: 'Compatibility route',
    heading: 'This is the same workspace: text, references, and image drafts in one thread.',
    description: 'Use the composer as the primary product surface. A request like “make this 4:5 for a social cover” stays in the same conversation and becomes a planning card.',
    status: 'Preview mode: drafts and pending visual cards are local planning surfaces.',
    primaryAction: { label: 'New conversation', to: '/app?new=1', icon: 'chat' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Conversation IA',
    detailTitle: 'Text message plus pending image task card',
    detailDescription: 'The chat route remains compatible, but it no longer forms a separate product mode.',
    cards: [
      {
        title: 'User message',
        kicker: 'Text',
        description: 'Capture the goal, audience, tone, and constraints as the first part of the thread.',
        icon: 'chat'
      },
      {
        title: 'Assistant placeholder',
        kicker: 'Reply',
        description: 'Reserve a response area for later capability without implying a live model call.',
        icon: 'terminal'
      },
      {
        title: 'Pending image card',
        kicker: 'Visual',
        description: 'Show the visual brief as a planning card that belongs inside the same conversation.',
        icon: 'sparkles'
      }
    ],
    checklist: [
      'Drafts remain local to the workspace view.',
      'Image planning is represented as a pending card, not a separate product island.',
      'Live replies, history, and usage tracking are reserved for later.'
    ]
  },
  image: {
    label: 'Image',
    shellTitle: 'Unified AI Conversation',
    shellSubtitle: 'Image requests now start from the same composer instead of a separate image workspace.',
    eyebrow: 'SSXZ AI',
    icon: 'sparkles',
    pill: 'Image route compatibility',
    heading: 'Image creation starts in the conversation input.',
    description: 'This route keeps old links safe while pointing the user back to the unified composer. Describe the ratio, style, use case, and reference direction naturally in the same thread.',
    status: 'Preview mode: image capability is represented as a pending conversation draft, not a separate page.',
    primaryAction: { label: 'New conversation', to: '/app?new=1', icon: 'chat' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Compatibility',
    detailTitle: 'Image tools live beside the composer',
    detailDescription: 'The image route no longer owns the main creation flow. It shows the same conversation shell so existing links do not become a separate product mode.',
    cards: [
      {
        title: 'Creation goal',
        kicker: 'Purpose',
        description: 'Name the subject, audience, and final use case before choosing a visual direction.',
        icon: 'sparkles'
      },
      {
        title: 'Style preference',
        kicker: 'Look',
        description: 'Capture mood, color, lighting, medium, and framing as reusable art direction.',
        icon: 'chat'
      },
      {
        title: 'Canvas ratio',
        kicker: 'Format',
        description: 'Compare square, portrait, poster, cover, and custom ratios as planning choices.',
        icon: 'terminal'
      },
      {
        title: 'Reference notes',
        kicker: 'Input',
        description: 'Describe reference images, brand boundaries, and must-avoid details without handling files here.',
        icon: 'sparkles'
      },
      {
        title: 'Prompt structure',
        kicker: 'Words',
        description: 'Break the brief into subject, setting, composition, style, lighting, and constraints.',
        icon: 'chat'
      },
      {
        title: 'Canvas preview',
        kicker: 'Next',
        description: 'Reserve a calm preview area for reviewing the future card while this screen stays descriptive.',
        icon: 'terminal'
      }
    ],
    checklist: [
      'Start from conversation context before writing visual details.',
      'Keep goal, output use, style, ratio, and reference direction visible in one brief.',
      'Use prompt structure to separate subject, composition, lighting, and constraints.',
      'Show the next state as a pending card that can return to the conversation shell.',
      'No live creation, file handling, or financial action starts from this static workspace.'
    ]
  },
  developer: {
    label: 'Developer',
    shellTitle: 'Developer Workspace',
    shellSubtitle: 'Preview integration surfaces for keys, models, limits, and release readiness.',
    eyebrow: 'Developer Workspace',
    icon: 'terminal',
    pill: 'Developer section',
    heading: 'Plan the developer setup before live management tools arrive.',
    description: 'Use this area to understand where access keys, model choices, limits, and launch checks will fit into the product.',
    status: 'Preview mode: integration planning is visible; live management actions come later.',
    primaryAction: { label: 'Billing overview', to: '/app/billing', icon: 'creditCard' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Integration checklist',
    detailTitle: 'A safe place to prepare launch details',
    detailDescription: 'This section frames the information a developer will need before keys, limits, and production checks become interactive.',
    cards: [
      {
        title: 'Access keys',
        kicker: 'Preview',
        description: 'Show where key management guidance and creation flows will live.',
        icon: 'terminal'
      },
      {
        title: 'Model access',
        kicker: 'Guide',
        description: 'Prepare the future area for model availability, limits, and recommended defaults.',
        icon: 'sparkles'
      },
      {
        title: 'Release checks',
        kicker: 'Plan',
        description: 'Collect authentication, rate limit, error handling, and monitoring reminders.',
        icon: 'chat'
      }
    ],
    checklist: [
      'Key management is presented as a future product area.',
      'Model and limit guidance stays descriptive.',
      'Account-sensitive details stay out of this preview.'
    ]
  },
  billing: {
    label: 'Billing',
    shellTitle: 'Billing Workspace',
    shellSubtitle: 'Preview balance, usage, invoices, and plan organization before billing actions open.',
    eyebrow: 'Billing Workspace',
    icon: 'creditCard',
    pill: 'Billing section',
    heading: 'Understand where billing information will live.',
    description: 'This section shows the planned shape for balance, usage, invoices, and plan details before financial actions are opened.',
    status: 'Preview mode: billing organization is visible; live financial actions arrive later.',
    primaryAction: { label: 'Account settings', to: '/app/account', icon: 'userCircle' },
    secondaryAction: { label: 'Developer setup', to: '/app/developer', icon: 'terminal' },
    detailKicker: 'Billing overview',
    detailTitle: 'Clear finance surfaces for later review',
    detailDescription: 'The workspace can explain what users will review here while plan changes, invoices, and usage records remain descriptive.',
    cards: [
      {
        title: 'Balance preview',
        kicker: 'Overview',
        description: 'Prepare a simple balance area for future account credit information.',
        icon: 'creditCard'
      },
      {
        title: 'Usage summary',
        kicker: 'Trends',
        description: 'Show where future usage charts and plan limits can be explained.',
        icon: 'chat'
      },
      {
        title: 'Billing history',
        kicker: 'Records',
        description: 'Reserve a clean area for invoices and order history once those views are enabled.',
        icon: 'terminal'
      }
    ],
    checklist: [
      'Financial action buttons are reserved for a later workflow.',
      'Balance and usage content is framed as an upcoming overview.',
      'Invoices and orders remain descriptive until live records are ready.'
    ]
  },
  account: {
    label: 'Account',
    shellTitle: 'Account Workspace',
    shellSubtitle: 'Preview profile, preferences, and security settings in one account area.',
    eyebrow: 'Account Workspace',
    icon: 'userCircle',
    pill: 'Account section',
    heading: 'Keep account settings easy to find.',
    description: 'This area organizes profile, preferences, notifications, and security settings so the future account page has a clear shape.',
    status: 'Preview mode: settings are outlined while account editing arrives later.',
    primaryAction: { label: 'Chat workspace', to: '/app/chat', icon: 'chat' },
    secondaryAction: { label: 'Billing overview', to: '/app/billing', icon: 'creditCard' },
    detailKicker: 'Account center',
    detailTitle: 'Profile and preferences share one home',
    detailDescription: 'The section gives users a stable place to expect identity, preferences, notifications, and security controls later.',
    cards: [
      {
        title: 'Profile',
        kicker: 'Identity',
        description: 'Prepare space for display name, contact details, and personal context.',
        icon: 'userCircle'
      },
      {
        title: 'Preferences',
        kicker: 'Settings',
        description: 'Group language, theme, and notification preferences in a predictable area.',
        icon: 'sparkles'
      },
      {
        title: 'Security',
        kicker: 'Protection',
        description: 'Frame where verification and security controls will be reviewed.',
        icon: 'terminal'
      }
    ],
    checklist: [
      'Profile details are represented as upcoming settings.',
      'Security controls are descriptive in this preview.',
      'Login and verification behavior remain outside this workspace polish.'
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
const showHeaderActions = computed(() => !isConversationWorkspace.value)

function isSectionKey(value: unknown): value is SectionKey {
  return typeof value === 'string' && sectionKeys.includes(value as SectionKey)
}

function applyPrompt(prompt: string) {
  draft.value = prompt
}

function recordDraft() {
  const content = draft.value.trim()
  if (!content) return
  localMessages.value.unshift({
    id: Date.now(),
    role: 'Draft',
    content
  })
  draft.value = ''
}
</script>

<style scoped>
.app-workspace-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.workspace-section-hero {
  display: grid;
  gap: 1.35rem;
}

.workspace-section-copy {
  max-width: 54rem;
}

.hero-pill {
  display: inline-flex;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  padding: 0.48rem 0.82rem;
  color: var(--ssxz-text);
  font-size: 0.75rem;
  font-weight: 720;
}

.workspace-section-hero h1 {
  margin-top: 1rem;
  max-width: 48rem;
  color: var(--ssxz-text);
  font-size: 2.75rem;
  font-weight: 760;
  line-height: 1.08;
}

.workspace-section-hero p {
  margin-top: 1rem;
  max-width: 45rem;
  color: var(--ssxz-body);
  font-size: 0.95rem;
  line-height: 1.8;
}

.workspace-composer,
.workspace-status-panel,
.workspace-detail-panel {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-2xl);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
}

.workspace-composer {
  padding: 0.85rem;
}

.unified-composer {
  display: grid;
  gap: 0.8rem;
}

.workspace-composer textarea {
  min-height: 8rem;
  width: 100%;
  resize: vertical;
  border: 0;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 1rem;
  line-height: 1.7;
  outline: none;
}

.workspace-composer textarea::placeholder {
  color: var(--ssxz-subtle);
}

.composer-tool-grid {
  display: grid;
  gap: 0.55rem;
  grid-template-columns: repeat(6, minmax(0, 1fr));
}

.composer-tool {
  display: flex;
  min-height: 4.25rem;
  align-items: flex-start;
  gap: 0.5rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-lg);
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text);
  padding: 0.7rem;
  text-align: left;
}

.composer-tool svg {
  color: var(--ssxz-primary);
  margin-top: 0.1rem;
}

.composer-tool span {
  display: grid;
  gap: 0.2rem;
  min-width: 0;
}

.composer-tool strong {
  font-size: 0.78rem;
  font-weight: 760;
}

.composer-tool small {
  color: var(--ssxz-body);
  font-size: 0.72rem;
  line-height: 1.35;
}

.composer-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-top: 1px solid var(--ssxz-border);
  padding-top: 0.75rem;
}

.prompt-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.prompt-chip {
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  padding: 0.45rem 0.75rem;
  font-size: 0.78rem;
  font-weight: 680;
  transition: border-color 0.16s ease, color 0.16s ease, background-color 0.16s ease;
}

.prompt-chip:hover {
  border-color: color-mix(in srgb, var(--ssxz-primary) 42%, var(--ssxz-border));
  background: var(--ssxz-active-bg);
  color: var(--ssxz-primary);
}

.workspace-status-panel {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  color: var(--ssxz-body);
  padding: 1rem;
  line-height: 1.7;
}

.status-dot,
.note-dot,
.check-dot {
  flex: 0 0 auto;
  border-radius: 9999px;
  background: var(--ssxz-success);
}

.status-dot,
.note-dot {
  height: 0.55rem;
  width: 0.55rem;
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--ssxz-success) 18%, transparent);
}

.workspace-local-thread {
  display: grid;
  gap: 0.75rem;
}

.local-message {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-xl);
  background: var(--ssxz-surface-muted);
  padding: 0.9rem;
}

.local-message-role {
  color: var(--ssxz-primary);
  font-size: 0.75rem;
  font-weight: 760;
}

.local-message p {
  margin-top: 0.35rem;
  color: var(--ssxz-body);
  line-height: 1.7;
  white-space: pre-wrap;
}

.unified-conversation-shell {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(20rem, 0.75fr);
  gap: 1rem;
}

.conversation-canvas,
.composer-intent-panel {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-2xl);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-sm);
  padding: 1rem;
}

.conversation-canvas {
  display: grid;
  gap: 0.75rem;
}

.conversation-message-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.8rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-xl);
  background: var(--ssxz-surface-muted);
  padding: 0.9rem;
}

.conversation-message-card[data-kind='image-plan'],
.conversation-message-card[data-kind='result'] {
  border-color: color-mix(in srgb, var(--ssxz-primary) 34%, var(--ssxz-border));
  background: var(--ssxz-active-bg);
}

.conversation-avatar {
  display: inline-flex;
  height: 2.25rem;
  width: 2.25rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-primary);
}

.conversation-message-copy {
  display: grid;
  gap: 0.26rem;
  min-width: 0;
}

.conversation-message-kicker {
  color: var(--ssxz-primary);
  font-size: 0.72rem;
  font-weight: 760;
}

.conversation-message-copy strong {
  color: var(--ssxz-text);
  font-size: 0.95rem;
}

.conversation-message-copy small {
  color: var(--ssxz-body);
  font-size: 0.82rem;
  line-height: 1.58;
  white-space: pre-wrap;
}

.composer-intent-panel {
  align-self: start;
}

.composer-intent-panel h3 {
  margin-top: 0.45rem;
  color: var(--ssxz-text);
  font-size: 1.1rem;
  font-weight: 760;
}

.composer-intent-panel p {
  margin-top: 0.5rem;
  color: var(--ssxz-body);
  font-size: 0.84rem;
  line-height: 1.65;
}

.composer-intent-panel ul {
  display: grid;
  gap: 0.58rem;
  margin: 0.95rem 0 0;
  padding: 0;
}

.composer-intent-panel li {
  display: grid;
  gap: 0.18rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-lg);
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  padding: 0.65rem;
  list-style: none;
}

.composer-intent-panel li span {
  color: var(--ssxz-primary);
  font-size: 0.72rem;
  font-weight: 760;
}

.composer-intent-panel li strong {
  color: var(--ssxz-text);
  font-size: 0.84rem;
  font-weight: 720;
}

.legacy-route-note {
  border-top: 1px solid var(--ssxz-border);
  padding-top: 0.75rem;
}

.conversation-ia-panel {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(18rem, 0.85fr);
  gap: 1rem;
}

.conversation-ia-copy,
.pending-image-plan {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-2xl);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-sm);
  padding: 1rem;
}

.conversation-ia-copy h2,
.pending-image-plan h3 {
  margin-top: 0.3rem;
  color: var(--ssxz-text);
  font-weight: 760;
}

.conversation-ia-copy h2 {
  font-size: 1.35rem;
}

.conversation-ia-copy > p,
.pending-image-plan p {
  margin-top: 0.5rem;
  color: var(--ssxz-body);
  font-size: 0.88rem;
  line-height: 1.7;
}

.conversation-thread-shell {
  display: grid;
  gap: 0.72rem;
  margin-top: 1rem;
}

.conversation-thread-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.75rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-xl);
  background: var(--ssxz-surface-muted);
  padding: 0.82rem;
}

.conversation-thread-icon {
  display: inline-flex;
  height: 2.15rem;
  width: 2.15rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.8rem;
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-primary);
}

.conversation-thread-copy {
  display: grid;
  gap: 0.25rem;
  min-width: 0;
}

.conversation-thread-copy span,
.pending-badge {
  color: var(--ssxz-primary);
  font-size: 0.72rem;
  font-weight: 760;
}

.conversation-thread-copy strong {
  color: var(--ssxz-text);
  font-size: 0.93rem;
}

.conversation-thread-copy small {
  color: var(--ssxz-body);
  font-size: 0.8rem;
  line-height: 1.55;
}

.pending-badge {
  display: inline-flex;
  border: 1px solid color-mix(in srgb, var(--ssxz-primary) 30%, var(--ssxz-border));
  border-radius: 9999px;
  background: var(--ssxz-active-bg);
  padding: 0.4rem 0.62rem;
}

.pending-image-plan ul {
  display: grid;
  gap: 0.58rem;
  margin: 0.95rem 0 0;
  padding: 0;
}

.pending-image-plan li {
  display: flex;
  align-items: flex-start;
  gap: 0.55rem;
  color: var(--ssxz-body);
  font-size: 0.82rem;
  line-height: 1.55;
  list-style: none;
}

.workspace-entry-grid,
.workspace-section-grid {
  display: grid;
  gap: 0.85rem;
}

.workspace-entry-grid {
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.workspace-section-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.workspace-entry-card,
.workspace-section-card {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-xl);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 88%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
  color: var(--ssxz-text);
  padding: 1rem;
}

.workspace-entry-card {
  display: grid;
  min-height: 13rem;
  grid-template-rows: auto 1fr auto;
  gap: 0.8rem;
  text-decoration: none;
  transition: border-color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.workspace-entry-card:hover {
  border-color: color-mix(in srgb, var(--ssxz-primary) 42%, var(--ssxz-border));
  box-shadow: var(--ssxz-shadow);
  transform: translateY(-1px);
}

.workspace-entry-icon,
.workspace-section-card-icon {
  display: inline-flex;
  height: 2.35rem;
  width: 2.35rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-primary);
}

.workspace-entry-copy {
  min-width: 0;
}

.workspace-entry-kicker,
.workspace-section-card-kicker,
.detail-kicker {
  display: block;
  color: var(--ssxz-primary);
  font-size: 0.72rem;
  font-weight: 760;
}

.workspace-entry-title {
  display: block;
  margin-top: 0.25rem;
  font-size: 1rem;
  font-weight: 760;
}

.workspace-entry-description,
.workspace-section-card p,
.workspace-detail-panel p {
  color: var(--ssxz-body);
  font-size: 0.84rem;
  line-height: 1.65;
}

.workspace-entry-description {
  display: block;
  margin-top: 0.45rem;
}

.workspace-entry-arrow {
  color: var(--ssxz-subtle);
  rotate: 45deg;
}

.workspace-section-card {
  display: grid;
  gap: 0.75rem;
}

.workspace-section-card h3,
.workspace-detail-panel h2 {
  color: var(--ssxz-text);
  font-weight: 760;
}

.workspace-section-card h3 {
  font-size: 1rem;
}

.workspace-detail-panel {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(16rem, 0.8fr);
  gap: 1rem;
  padding: 1rem;
}

.workspace-detail-panel h2 {
  margin-top: 0.25rem;
  font-size: 1.25rem;
}

.workspace-detail-panel p {
  margin-top: 0.45rem;
}

.workspace-detail-panel ul {
  display: grid;
  gap: 0.65rem;
  margin: 0;
  padding: 0;
}

.workspace-detail-panel li {
  display: flex;
  align-items: flex-start;
  gap: 0.55rem;
  color: var(--ssxz-body);
  font-size: 0.85rem;
  line-height: 1.6;
  list-style: none;
}

.check-dot {
  height: 0.45rem;
  margin-top: 0.45rem;
  width: 0.45rem;
}

.workspace-note-strip {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  padding: 0.75rem 1rem;
  font-size: 0.82rem;
  line-height: 1.55;
}

@media (max-width: 1120px) {
  .composer-tool-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .unified-conversation-shell,
  .conversation-ia-panel,
  .workspace-detail-panel {
    grid-template-columns: 1fr;
  }

  .workspace-entry-grid,
  .workspace-section-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .app-workspace-actions,
  .composer-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .composer-tool-grid {
    grid-template-columns: 1fr;
  }

  .workspace-entry-grid,
  .workspace-section-grid {
    grid-template-columns: 1fr;
  }

  .workspace-entry-card {
    min-height: 0;
  }

  .conversation-thread-item {
    grid-template-columns: 1fr;
  }

  .conversation-message-card {
    grid-template-columns: 1fr;
  }

  .workspace-section-hero h1 {
    font-size: 2.05rem;
  }

  .workspace-note-strip {
    align-items: flex-start;
    border-radius: var(--ssxz-radius-xl);
  }
}
</style>
