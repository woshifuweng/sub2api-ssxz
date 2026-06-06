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

    <section
      v-if="isUnifiedWorkspaceSection"
      class="conversation-ia-panel"
      :aria-label="`${activeContent.label} conversation workspace structure`"
    >
      <div class="conversation-ia-copy">
        <span class="detail-kicker">{{ conversationPanel.kicker }}</span>
        <h2>{{ conversationPanel.title }}</h2>
        <p>{{ conversationPanel.description }}</p>

        <div class="conversation-thread-shell" aria-label="Conversation preview">
          <article v-for="item in conversationPanel.thread" :key="item.title" class="conversation-thread-item">
            <span class="conversation-thread-icon">
              <Icon :name="item.icon" size="sm" />
            </span>
            <span class="conversation-thread-copy">
              <span>{{ item.kicker }}</span>
              <strong>{{ item.title }}</strong>
              <small>{{ item.description }}</small>
            </span>
          </article>
        </div>
      </div>

      <aside class="pending-image-plan" aria-label="Pending image plan card">
        <span class="pending-badge">{{ conversationPanel.imagePlan.badge }}</span>
        <h3>{{ conversationPanel.imagePlan.title }}</h3>
        <p>{{ conversationPanel.imagePlan.description }}</p>
        <ul>
          <li v-for="item in conversationPanel.imagePlan.items" :key="item">
            <span class="check-dot" />
            <span>{{ item }}</span>
          </li>
        </ul>
      </aside>
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

type UnifiedSectionKey = Extract<SectionKey, 'home' | 'chat' | 'image'>

interface ConversationThreadItem {
  kicker: string
  title: string
  description: string
  icon: IconName
}

interface PendingImagePlan {
  badge: string
  title: string
  description: string
  items: string[]
}

interface ConversationPanel {
  kicker: string
  title: string
  description: string
  thread: ConversationThreadItem[]
  imagePlan: PendingImagePlan
}

const route = useRoute()
const draft = ref('')
const localMessages = ref<LocalMessage[]>([])

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image', 'developer', 'billing', 'account']
const unifiedSectionKeys: readonly UnifiedSectionKey[] = ['home', 'chat', 'image']

const workspaceEntries: WorkspaceEntry[] = [
  {
    title: 'AI Chat',
    kicker: 'Conversation lane',
    description: 'Work with text messages and image planning cards inside one conversation shell.',
    icon: 'chat',
    to: '/app/chat'
  },
  {
    title: 'AI Image',
    kicker: 'Image planning lane',
    description: 'Turn a visual brief into a pending image card that belongs to the same workspace.',
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
  { label: 'Image plan', prompt: 'Turn this idea into a visual plan with subject, setting, lighting, color, ratio, and reference notes.' },
  { label: 'Integration plan', prompt: 'List the checks needed before an integration: model choice, access, limits, errors, and release gates.' }
]

const conversationPanels: Record<UnifiedSectionKey, ConversationPanel> = {
  home: {
    kicker: 'Unified conversation',
    title: 'One workspace for text and image planning',
    description: 'The home view should feel like the start of the same AI conversation, with recent context and pending visual work visible together.',
    thread: [
      {
        kicker: 'User message',
        title: 'Plan a product launch story',
        description: 'A text prompt can start the conversation, set the audience, and define the creative goal.',
        icon: 'chat'
      },
      {
        kicker: 'Assistant placeholder',
        title: 'Reply outline waits in the same thread',
        description: 'The workspace can reserve a calm response area without calling a model in this phase.',
        icon: 'sparkles'
      },
      {
        kicker: 'Image plan',
        title: 'Visual brief is attached to the conversation',
        description: 'A pending image card can sit beside the text flow instead of living on a separate island.',
        icon: 'terminal'
      }
    ],
    imagePlan: {
      badge: 'Pending image plan',
      title: 'Visual card belongs to the current conversation',
      description: 'Use this shell to show that image work is planned from conversation context and can be reviewed before any live action exists.',
      items: ['Goal and output use stay visible.', 'Style, ratio, and references are grouped.', 'The card remains planning-only in this phase.']
    }
  },
  chat: {
    kicker: 'Conversation view',
    title: 'Text and image planning share the same thread',
    description: 'The chat route should show how a future conversation can hold text notes, reply placeholders, and pending visual cards together.',
    thread: [
      {
        kicker: 'Text message',
        title: 'Describe the campaign or question',
        description: 'The message area starts with ordinary text planning and local drafts.',
        icon: 'chat'
      },
      {
        kicker: 'Assistant placeholder',
        title: 'Response area is prepared, not live',
        description: 'Copy and structure can be reviewed while live replies remain scheduled for later.',
        icon: 'sparkles'
      },
      {
        kicker: 'Pending image card',
        title: 'Image brief can appear inside the conversation',
        description: 'Creative goal, style, ratio, and reference notes are visible as part of the same thread.',
        icon: 'terminal'
      }
    ],
    imagePlan: {
      badge: 'Planning card',
      title: 'Image work starts from chat context',
      description: 'The card explains what the future visual work needs, while staying safely in preview mode.',
      items: ['No live reply is requested.', 'No visual work is started.', 'No financial action is connected.']
    }
  },
  image: {
    kicker: 'Image planning view',
    title: 'Build the visual brief, then return it to the conversation',
    description: 'The image route remains useful as a focused planning surface, but the output is a pending card that belongs to the same workspace history.',
    thread: [
      {
        kicker: 'Conversation context',
        title: 'Start from the current user goal',
        description: 'The image plan should inherit intent from the workspace rather than behave like an isolated page.',
        icon: 'chat'
      },
      {
        kicker: 'Creative brief',
        title: 'Organize goal, style, ratio, and references',
        description: 'The planning section collects the ingredients a designer or creator would review first.',
        icon: 'sparkles'
      },
      {
        kicker: 'Pending card',
        title: 'Send the brief back to the conversation shell',
        description: 'The card stays pending and descriptive until a later batch introduces live capability.',
        icon: 'terminal'
      }
    ],
    imagePlan: {
      badge: 'Planning handoff',
      title: 'Image brief links back to the workspace thread',
      description: 'This area clarifies how the visual plan can be reviewed alongside text messages and future history.',
      items: ['Reference area is descriptive only.', 'Canvas preview remains a planning surface.', 'The next state is pending, not completed.']
    }
  }
}

const sectionContent: Record<SectionKey, SectionContent> = {
  home: {
    label: 'Home',
    shellTitle: 'AI Workspace',
    shellSubtitle: 'A calm conversation workspace for text, image planning, developer, billing, and account work.',
    eyebrow: 'SSXZ AI Workspace',
    icon: 'sparkles',
    pill: 'Unified conversation workspace',
    heading: 'Start with one AI thread, then branch into the work it needs.',
    description: 'Use the workspace home to see how conversation, image planning, recent context, and account areas belong to one product surface instead of separate tools.',
    status: 'Preview mode: text and image planning are organized together while live actions arrive later.',
    primaryAction: { label: 'Open chat', to: '/app/chat', icon: 'chat' },
    secondaryAction: { label: 'Plan image', to: '/app/image', icon: 'sparkles' },
    detailKicker: 'Workspace IA',
    detailTitle: 'Conversation first, sections second',
    detailDescription: 'The main app routes now share a conversation-centered structure: chat holds the thread, image planning creates a pending visual card, and the home view keeps the flow understandable.',
    cards: [],
    checklist: [
      'The home view introduces one conversation workspace rather than a dashboard.',
      'Image planning is shown as part of the same thread and history shell.',
      'Developer, billing, and account areas remain supporting sections for later refinement.'
    ]
  },
  chat: {
    label: 'Chat',
    shellTitle: 'Conversation Workspace',
    shellSubtitle: 'Plan text messages and pending image cards inside the same AI thread.',
    eyebrow: 'Conversation Workspace',
    icon: 'chat',
    pill: 'Unified thread',
    heading: 'Write, plan, and attach visual work in one conversation.',
    description: 'Use the prompt area to sketch the user message, then review how a pending image plan can sit in the same thread without starting live work.',
    status: 'Preview mode: drafts and pending visual cards are local planning surfaces.',
    primaryAction: { label: 'Plan image card', to: '/app/image', icon: 'sparkles' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Conversation IA',
    detailTitle: 'Text message plus pending image card',
    detailDescription: 'The chat route now explains how text planning, assistant placeholders, and image cards share a single conversation shell.',
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
    shellTitle: 'Image Planning Workspace',
    shellSubtitle: 'Turn conversation context into a focused visual brief and pending card.',
    eyebrow: 'Image Planning',
    icon: 'sparkles',
    pill: 'Conversation image plan',
    heading: 'Shape the image brief, then keep it with the conversation.',
    description: 'Organize goal, output use, style, ratio, reference direction, prompt structure, and preview notes as a planning card that can live beside text messages.',
    status: 'Preview mode: this image section prepares a pending planning card for the shared workspace.',
    primaryAction: { label: 'Back to thread', to: '/app/chat', icon: 'chat' },
    secondaryAction: { label: 'Workspace home', to: '/app', icon: 'sparkles' },
    detailKicker: 'Image planning flow',
    detailTitle: 'Goal, use, style, canvas, reference, prompt, pending card',
    detailDescription: 'The image route is a focused planning desk for the same conversation: it gathers visual intent and returns a reviewable card to the shared thread.',
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
const isUnifiedWorkspaceSection = computed(() => isUnifiedSectionKey(activeSection.value))
const conversationPanel = computed(() => (
  isUnifiedSectionKey(activeSection.value) ? conversationPanels[activeSection.value] : conversationPanels.home
))

function isSectionKey(value: unknown): value is SectionKey {
  return typeof value === 'string' && sectionKeys.includes(value as SectionKey)
}

function isUnifiedSectionKey(value: SectionKey): value is UnifiedSectionKey {
  return unifiedSectionKeys.includes(value as UnifiedSectionKey)
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

  .workspace-section-hero h1 {
    font-size: 2.05rem;
  }

  .workspace-note-strip {
    align-items: flex-start;
    border-radius: var(--ssxz-radius-xl);
  }
}
</style>
