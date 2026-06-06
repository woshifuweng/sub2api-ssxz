<template>
  <AppSectionShell
    :title="activeContent.shellTitle"
    :subtitle="activeContent.shellSubtitle"
    :eyebrow="activeContent.eyebrow"
    :icon="activeContent.icon"
  >
    <template #actions>
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

      <div v-if="activeSection === 'chat'" class="workspace-composer">
        <textarea
          v-model="draft"
          rows="4"
          placeholder="Sketch a prompt, outline the audience, or plan the next conversation."
        />
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
            Save draft
          </button>
        </div>
      </div>

      <div v-else class="workspace-status-panel">
        <span class="status-dot" />
        <span>{{ activeContent.status }}</span>
      </div>
    </section>

    <section v-if="activeSection === 'chat' && localMessages.length" class="workspace-local-thread">
      <article v-for="message in localMessages" :key="message.id" class="local-message">
        <span class="local-message-role">{{ message.role }}</span>
        <p>{{ message.content }}</p>
      </article>
    </section>

    <section v-if="activeSection === 'home'" class="workspace-entry-grid" aria-label="Workspace sections">
      <RouterLink v-for="entry in workspaceEntries" :key="entry.to" :to="entry.to" class="workspace-entry-card">
        <span class="workspace-entry-icon">
          <Icon :name="entry.icon" size="sm" />
        </span>
        <span class="workspace-entry-copy">
          <span class="workspace-entry-kicker">{{ entry.kicker }}</span>
          <span class="workspace-entry-title">{{ entry.title }}</span>
          <span class="workspace-entry-description">{{ entry.description }}</span>
        </span>
        <Icon name="arrowUp" size="xs" class="workspace-entry-arrow" />
      </RouterLink>
    </section>

    <section v-else class="workspace-section-grid" :aria-label="`${activeContent.label} workspace preview`">
      <article v-for="card in activeContent.cards" :key="card.title" class="workspace-section-card">
        <span class="workspace-section-card-icon">
          <Icon :name="card.icon" size="sm" />
        </span>
        <span class="workspace-section-card-kicker">{{ card.kicker }}</span>
        <h3>{{ card.title }}</h3>
        <p>{{ card.description }}</p>
      </article>
    </section>

    <section class="workspace-detail-panel">
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

    <section class="workspace-note-strip">
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

interface WorkspaceEntry {
  title: string
  kicker: string
  description: string
  icon: IconName
  to: string
}

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

const route = useRoute()
const draft = ref('')
const localMessages = ref<LocalMessage[]>([])

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image', 'developer', 'billing', 'account']

const workspaceEntries: WorkspaceEntry[] = [
  {
    title: 'AI Chat',
    kicker: 'Conversation planning',
    description: 'Shape prompts, tone, and response goals before starting a model conversation.',
    icon: 'chat',
    to: '/app/chat'
  },
  {
    title: 'AI Image',
    kicker: 'Creative planning',
    description: 'Gather image ideas, style notes, aspect ratios, and reference intent in one place.',
    icon: 'sparkles',
    to: '/app/image'
  },
  {
    title: 'Developer API',
    kicker: 'Integration guide',
    description: 'Preview where keys, models, limits, and release checks will be managed.',
    icon: 'terminal',
    to: '/app/developer'
  },
  {
    title: 'Billing',
    kicker: 'Account overview',
    description: 'See how balance, usage, invoices, and plan details will be organized.',
    icon: 'creditCard',
    to: '/app/billing'
  },
  {
    title: 'Account',
    kicker: 'Profile center',
    description: 'Find profile, preferences, and security settings in a familiar account hub.',
    icon: 'userCircle',
    to: '/app/account'
  }
]

const promptChips = [
  { label: 'Launch pitch', prompt: 'Draft five crisp benefits for a product launch, with audience, angle, and proof points.' },
  { label: 'Image brief', prompt: 'Turn this idea into a visual brief with subject, setting, lighting, color, and style notes.' },
  { label: 'Integration plan', prompt: 'List the checks needed before an integration: model choice, access, limits, errors, and release gates.' }
]

const sectionContent: Record<SectionKey, SectionContent> = {
  home: {
    label: 'Home',
    shellTitle: 'AI Workspace',
    shellSubtitle: 'A calm starting point for chat, image, developer, billing, and account work.',
    eyebrow: 'SSXZ AI Workspace',
    icon: 'sparkles',
    pill: 'Unified workspace',
    heading: 'Start from one workspace and move into the task you need.',
    description: 'Use the quick entries to plan conversations, image work, integrations, billing review, or account settings. Each area is organized now, with live actions coming later.',
    status: 'Preview mode: explore the workspace structure and prepare your next task.',
    primaryAction: { label: 'Open chat', to: '/app/chat', icon: 'chat' },
    secondaryAction: { label: 'View billing', to: '/app/billing', icon: 'creditCard' },
    detailKicker: 'Workspace guide',
    detailTitle: 'Everything has a place',
    detailDescription: 'The main app routes now share one workspace shell, so each section feels connected while keeping its own purpose.',
    cards: [],
    checklist: [
      'Chat, image, developer, billing, and account sections are available from one hub.',
      'Each section focuses on planning and orientation before live actions are added.',
      'Older app links continue to land on the matching workspace section.'
    ]
  },
  chat: {
    label: 'Chat',
    shellTitle: 'AI Chat Workspace',
    shellSubtitle: 'Draft prompts and shape the conversation before model replies are connected.',
    eyebrow: 'Chat Workspace',
    icon: 'chat',
    pill: 'Chat section',
    heading: 'Shape the conversation before you send it.',
    description: 'Use the prompt area to clarify the goal, audience, tone, and expected output. Saved drafts stay on the page as planning notes.',
    status: 'Preview mode: draft locally while model chat is prepared for a later release.',
    primaryAction: { label: 'Image ideas', to: '/app/image', icon: 'sparkles' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Conversation flow',
    detailTitle: 'A clearer prompt, a better first reply',
    detailDescription: 'This area helps you prepare context, constraints, and review criteria before the live chat flow is introduced.',
    cards: [
      {
        title: 'Prompt draft',
        kicker: 'Plan',
        description: 'Capture the goal, audience, and tone before turning the idea into a conversation.',
        icon: 'chat'
      },
      {
        title: 'Model notes',
        kicker: 'Prepare',
        description: 'Keep space for future model choices, response style, and quality checks.',
        icon: 'terminal'
      },
      {
        title: 'Response plan',
        kicker: 'Preview',
        description: 'Outline expected output shape, review steps, and next actions.',
        icon: 'sparkles'
      }
    ],
    checklist: [
      'Drafts remain local to the workspace view.',
      'Prompt chips help you start from a useful structure.',
      'Live replies, history, and usage tracking are reserved for later.'
    ]
  },
  image: {
    label: 'Image',
    shellTitle: 'AI Image Workspace',
    shellSubtitle: 'Plan creative direction, references, and composition before image creation opens.',
    eyebrow: 'Image Workspace',
    icon: 'sparkles',
    pill: 'Image section',
    heading: 'Turn a visual idea into a clear creative brief.',
    description: 'Map the subject, style, composition, aspect ratio, and reference direction so the future image flow has a strong starting point.',
    status: 'Preview mode: prepare the brief now; live creation arrives later.',
    primaryAction: { label: 'Developer setup', to: '/app/developer', icon: 'terminal' },
    secondaryAction: { label: 'Chat draft', to: '/app/chat', icon: 'chat' },
    detailKicker: 'Creative checklist',
    detailTitle: 'From idea to production-ready brief',
    detailDescription: 'Use the cards below to organize creative intent before any generation workflow is introduced.',
    cards: [
      {
        title: 'Creative goal',
        kicker: 'Brief',
        description: 'Define the scene, product, character, or mood you want the image to express.',
        icon: 'sparkles'
      },
      {
        title: 'Style notes',
        kicker: 'Direction',
        description: 'Capture color, lighting, medium, framing, and aspect ratio preferences.',
        icon: 'chat'
      },
      {
        title: 'Reference intent',
        kicker: 'Context',
        description: 'Describe any reference material, constraints, or must-avoid details.',
        icon: 'terminal'
      }
    ],
    checklist: [
      'Brief structure is ready for subject, style, ratio, and references.',
      'The page keeps creative planning separate from live creation.',
      'Billing and usage behavior remain outside this preview.'
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
  .workspace-entry-grid,
  .workspace-section-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .workspace-detail-panel {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .app-workspace-actions,
  .composer-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .workspace-entry-grid,
  .workspace-section-grid {
    grid-template-columns: 1fr;
  }

  .workspace-entry-card {
    min-height: 0;
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
