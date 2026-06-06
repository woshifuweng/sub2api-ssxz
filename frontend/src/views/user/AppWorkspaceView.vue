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
          placeholder="Draft a prompt or plan a conversation. This stays local until chat is connected."
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

    <section v-else class="workspace-section-grid" :aria-label="`${activeContent.label} static preview`">
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
        Router handles /app authentication and returnTo. This component only switches static workspace sections and does not call chat, image, billing, account, or backend APIs.
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
    kicker: 'Static workspace',
    description: 'Prepare prompts and conversation goals before the real chat service is connected.',
    icon: 'chat',
    to: '/app/chat'
  },
  {
    title: 'AI Image',
    kicker: 'Creative planning',
    description: 'Collect image ideas, style notes, aspect ratios, and reference intent without starting a generation job.',
    icon: 'sparkles',
    to: '/app/image'
  },
  {
    title: 'Developer API',
    kicker: 'Integration preview',
    description: 'Review API, model, key, and permission entry points without requesting account data.',
    icon: 'terminal',
    to: '/app/developer'
  },
  {
    title: 'Billing',
    kicker: 'Account entry',
    description: 'Keep balance, usage, invoice, and order entry points visible without loading payment data.',
    icon: 'creditCard',
    to: '/app/billing'
  },
  {
    title: 'Account',
    kicker: 'Profile center',
    description: 'Reserve space for profile, preference, and security settings without reading profile data.',
    icon: 'userCircle',
    to: '/app/account'
  }
]

const promptChips = [
  { label: 'Product pitch', prompt: 'Draft five crisp benefits for a product launch, with audience, angle, and proof points.' },
  { label: 'Image brief', prompt: 'Turn this idea into a visual brief with subject, setting, lighting, color, and style notes.' },
  { label: 'API plan', prompt: 'List the checks needed before an API integration: model, auth, quotas, errors, and release gates.' }
]

const sectionContent: Record<SectionKey, SectionContent> = {
  home: {
    label: 'Home',
    shellTitle: 'AI Workspace',
    shellSubtitle: 'A static landing area for chat, image, developer, billing, and account work. Real services stay disconnected.',
    eyebrow: 'SSXZ AI Workspace',
    icon: 'sparkles',
    pill: 'Unified /app workspace',
    heading: 'Choose a workspace section without triggering backend work.',
    description: 'The shell is active across /app routes. It now reads route metadata and shows the right static section while router auth remains the only login gate.',
    status: 'Static workspace overview. No chat, image, billing, account, or backend requests run here.',
    primaryAction: { label: 'Open chat', to: '/app/chat', icon: 'chat' },
    secondaryAction: { label: 'View billing', to: '/app/billing', icon: 'creditCard' },
    detailKicker: 'Activation status',
    detailTitle: 'Workspace routing is active',
    detailDescription: 'Each /app route points to this shell and passes an appSection value through route meta.',
    cards: [],
    checklist: [
      'Router handles /app authentication before this component mounts.',
      'Each section is a static preview and avoids service calls.',
      'Legacy routes continue to land on the canonical /app section paths.'
    ]
  },
  chat: {
    label: 'Chat',
    shellTitle: 'AI Chat Workspace',
    shellSubtitle: 'Draft prompts and conversation plans locally. The chat workspace client is not called from this screen.',
    eyebrow: 'Static Chat Section',
    icon: 'chat',
    pill: 'Chat section',
    heading: 'Prepare the conversation before real chat is enabled.',
    description: 'Use this section to shape prompts, compare angles, and save local draft notes without sending messages to a model.',
    status: 'Local draft mode only. No conversation, message, or model request is sent.',
    primaryAction: { label: 'Image ideas', to: '/app/image', icon: 'sparkles' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'No runtime client',
    detailTitle: 'Chat stays offline in this batch',
    detailDescription: 'The text area and prompt chips are local UI helpers. Conversation persistence and model replies remain for a later batch.',
    cards: [
      {
        title: 'Prompt draft',
        kicker: 'Local only',
        description: 'Capture the goal, audience, and tone before a real chat flow exists.',
        icon: 'chat'
      },
      {
        title: 'Model placeholder',
        kicker: 'Not connected',
        description: 'Reserve the future model selector area without reading model or account capability data.',
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
      'No chatWorkspace methods are called.',
      'No message is saved outside local component state.',
      'No usage or billing event is produced.'
    ]
  },
  image: {
    label: 'Image',
    shellTitle: 'AI Image Workspace',
    shellSubtitle: 'A static creative planning area for image tasks. Real generation and uploads remain disconnected.',
    eyebrow: 'Static Image Section',
    icon: 'sparkles',
    pill: 'Image section',
    heading: 'Shape image intent without starting a generation task.',
    description: 'Plan subject, style, composition, aspect ratio, and reference intent while AppImageView and generation APIs stay frozen.',
    status: 'Static image brief only. No asset upload, generation job, or image API is triggered.',
    primaryAction: { label: 'Developer setup', to: '/app/developer', icon: 'terminal' },
    secondaryAction: { label: 'Chat draft', to: '/app/chat', icon: 'chat' },
    detailKicker: 'Creative checklist',
    detailTitle: 'Generation remains offline',
    detailDescription: 'This section is for planning the image workflow and does not import or render AppImageView.',
    cards: [
      {
        title: 'Creative goal',
        kicker: 'Brief',
        description: 'Write the target scene, product, character, or mood before a real task exists.',
        icon: 'sparkles'
      },
      {
        title: 'Style notes',
        kicker: 'Static controls',
        description: 'Reserve space for future style, color, lighting, and aspect ratio controls.',
        icon: 'chat'
      },
      {
        title: 'Reference intent',
        kicker: 'No upload',
        description: 'Describe reference material without accepting files or creating assets.',
        icon: 'terminal'
      }
    ],
    checklist: [
      'No image task is created.',
      'No reference asset upload runs.',
      'No generation, billing, or usage status is read.'
    ]
  },
  developer: {
    label: 'Developer',
    shellTitle: 'Developer Workspace',
    shellSubtitle: 'Static API and key management entry points without loading keys, permissions, or admin data.',
    eyebrow: 'Static Developer Section',
    icon: 'terminal',
    pill: 'Developer section',
    heading: 'Prepare API integration without reading private account data.',
    description: 'This section reserves space for keys, models, quotas, and release checks while all API key and permission calls stay disconnected.',
    status: 'Static developer overview only. No key, permission, or admin endpoint is called.',
    primaryAction: { label: 'Billing overview', to: '/app/billing', icon: 'creditCard' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Integration checklist',
    detailTitle: 'API setup is a placeholder',
    detailDescription: 'Use this screen to outline what a future API setup page should explain without requesting real key data.',
    cards: [
      {
        title: 'API keys',
        kicker: 'Placeholder',
        description: 'Reserve the key management area without listing, creating, or revealing keys.',
        icon: 'terminal'
      },
      {
        title: 'Model access',
        kicker: 'Static',
        description: 'Show where model capability guidance can live without capability calls.',
        icon: 'sparkles'
      },
      {
        title: 'Release checks',
        kicker: 'Plan',
        description: 'Collect auth, rate limit, and error handling reminders for later implementation.',
        icon: 'chat'
      }
    ],
    checklist: [
      'No API key endpoint is called.',
      'No permissions or admin data is loaded.',
      'No private account value is rendered.'
    ]
  },
  billing: {
    label: 'Billing',
    shellTitle: 'Billing Workspace',
    shellSubtitle: 'Static balance, usage, and billing entry points without loading financial data or starting checkout.',
    eyebrow: 'Static Billing Section',
    icon: 'creditCard',
    pill: 'Billing section',
    heading: 'Preview billing surfaces without touching checkout flows.',
    description: 'The section keeps account finance navigation visible while order, invoice, usage, recharge, and checkout calls remain disabled.',
    status: 'Static billing overview only. No balance, order, invoice, or checkout request is sent.',
    primaryAction: { label: 'Account settings', to: '/app/account', icon: 'userCircle' },
    secondaryAction: { label: 'Developer setup', to: '/app/developer', icon: 'terminal' },
    detailKicker: 'Financial boundary',
    detailTitle: 'Checkout remains fully disconnected',
    detailDescription: 'This batch does not read balances, create orders, open checkout pages, or produce billing events.',
    cards: [
      {
        title: 'Balance preview',
        kicker: 'Static',
        description: 'Reserve the account balance card without requesting balance data.',
        icon: 'creditCard'
      },
      {
        title: 'Usage summary',
        kicker: 'Static',
        description: 'Hold space for future usage charts without usage API calls.',
        icon: 'chat'
      },
      {
        title: 'Billing history',
        kicker: 'Static',
        description: 'Show a future invoice and order area without reading order records.',
        icon: 'terminal'
      }
    ],
    checklist: [
      'No checkout callback, recharge, order, or invoice flow is touched.',
      'No usage or balance endpoint is called.',
      'No billing event is produced.'
    ]
  },
  account: {
    label: 'Account',
    shellTitle: 'Account Workspace',
    shellSubtitle: 'Static account, preference, and security entry points without profile or auth-state changes.',
    eyebrow: 'Static Account Section',
    icon: 'userCircle',
    pill: 'Account section',
    heading: 'Reserve account settings without reading profile data.',
    description: 'Use this section to frame profile, preferences, notifications, and security areas while auth/session logic remains untouched.',
    status: 'Static account overview only. No profile, auth-state, or permission request is sent.',
    primaryAction: { label: 'Chat workspace', to: '/app/chat', icon: 'chat' },
    secondaryAction: { label: 'Billing overview', to: '/app/billing', icon: 'creditCard' },
    detailKicker: 'Account boundary',
    detailTitle: 'Profile stays untouched',
    detailDescription: 'This screen does not modify login, registration, verification, auth-state, permission, or role behavior.',
    cards: [
      {
        title: 'Profile',
        kicker: 'Placeholder',
        description: 'Reserve profile identity and contact areas without loading user records.',
        icon: 'userCircle'
      },
      {
        title: 'Preferences',
        kicker: 'Static',
        description: 'Show where language, theme, and notifications can live later.',
        icon: 'sparkles'
      },
      {
        title: 'Security',
        kicker: 'Static',
        description: 'Reserve verification and security settings without changing auth logic.',
        icon: 'terminal'
      }
    ],
    checklist: [
      'No profile endpoint is called.',
      'No login, registration, verification, auth-state, or role logic changes.',
      'No credential-like value is read or displayed.'
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
  gap: 1.25rem;
}

.workspace-section-copy {
  max-width: 54rem;
}

.hero-pill {
  display: inline-flex;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  padding: 0.5rem 0.85rem;
  color: var(--ssxz-body);
  font-size: 0.75rem;
  font-weight: 720;
}

.workspace-section-hero h1 {
  margin-top: 1rem;
  max-width: 48rem;
  color: var(--ssxz-text);
  font-size: clamp(2rem, 4vw, 3.5rem);
  font-weight: 760;
  line-height: 1.05;
}

.workspace-section-hero p {
  margin-top: 1rem;
  max-width: 45rem;
  color: var(--ssxz-muted);
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
  color: var(--ssxz-muted);
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
  color: var(--ssxz-muted);
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
  color: var(--ssxz-muted);
  padding: 0.75rem 1rem;
  font-size: 0.82rem;
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

  .workspace-note-strip {
    align-items: flex-start;
    border-radius: var(--ssxz-radius-xl);
  }
}
</style>
