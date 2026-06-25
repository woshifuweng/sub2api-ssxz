<template>
  <AppSectionShell
    :title="activeContent.shellTitle"
    :subtitle="activeContent.shellSubtitle"
    :eyebrow="activeContent.eyebrow"
    :icon="activeContent.icon"
    :history-items="workspace.conversations.value"
    :active-conversation-id="workspace.activeConversationId.value"
    :history-loading="workspace.loadingHistory.value"
    @new-chat="startNewChat"
    @select-conversation="selectConversation"
  >
    <section class="workspace-page" :data-section="activeSection">
      <div class="workspace-main" :class="{ 'has-messages': workspace.messages.value.length > 0 }">
        <section
          v-if="workspace.messages.value.length === 0"
          class="empty-state"
          aria-label="开始对话"
        >
          <h1>欢迎使用 SSXZ AI 工作台</h1>
          <p>{{ emptyStateCopy }}</p>
        </section>

        <WorkspaceMessageList
          v-else
          :messages="workspace.messages.value"
          :loading="workspace.loadingMessages.value"
        />
      </div>

      <p v-if="workspace.errorMessage.value" class="workspace-error" role="alert">
        {{ workspace.errorMessage.value }}
      </p>
      <p v-else-if="!workspace.backendEnabled.value" class="workspace-notice" role="status">
        统一工作台后端正在接入，暂不可发送。当前仅展示工作台入口。
      </p>

      <section class="composer-zone" aria-label="统一输入框">
        <WorkspaceComposer
          v-model="draft"
          :selected-model="activeChatModel"
          :models="chatModels"
          :intent="workspaceIntent"
          :image-capability-available="imageCapabilityAvailable"
          :backend-enabled="workspace.backendEnabled.value"
          :sending="workspace.sending.value || assets.registering.value"
          :asset-previews="assets.previews.value"
          :rejected-files="assets.rejectedFiles.value"
          :web-search-available="webSearchAvailable"
          :web-search-enabled="webSearchRequested"
          @update:selected-model="selectedModelId = $event"
          @toggle-web-search="toggleWebSearch"
          @files="handleFiles"
          @unsupported-files="handleUnsupportedFiles"
          @remove-asset="assets.removePreview"
          @submit="submitDraft"
        />
      </section>
    </section>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import { useUserCapabilities } from '@/composables/useUserCapabilities'
import { useAppStore } from '@/stores/app'
import WorkspaceComposer from './workspace/WorkspaceComposer.vue'
import WorkspaceMessageList from './workspace/WorkspaceMessageList.vue'
import { useWorkspaceAssets } from './workspace/useWorkspaceAssets'
import {
  WORKSPACE_TEXT_ONLY_MESSAGE,
  useWorkspaceConversation,
  type WorkspaceIntent
} from './workspace/useWorkspaceConversation'

type IconName = InstanceType<typeof Icon>['$props']['name']
type SectionKey = 'home' | 'chat' | 'image'

interface SectionContent {
  shellTitle: string
  shellSubtitle: string
  eyebrow: string
  icon: IconName
}

const route = useRoute()
const appStore = useAppStore()
const draft = ref('')
const selectedModelId = ref('')
const webSearchRequested = ref(false)
const workspace = useWorkspaceConversation()
const assets = useWorkspaceAssets()
const capabilities = useUserCapabilities()
const {
  chatModels,
  hasChat,
  loadCapabilities
} = capabilities
const defaultTextModel = capabilities.defaultTextModel ?? ref('')

const sectionKeys: readonly SectionKey[] = ['home', 'chat', 'image']

const sectionContent: Record<SectionKey, SectionContent> = {
  home: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '直接输入问题，开始对话。',
    eyebrow: '对话工作台',
    icon: 'chat'
  },
  chat: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '当前开放文本对话 beta。',
    eyebrow: '对话工作台',
    icon: 'chat'
  },
  image: {
    shellTitle: 'SSXZ AI',
    shellSubtitle: '当前开放文本对话 beta。',
    eyebrow: '对话工作台',
    icon: 'chat'
  }
}

const activeSection = computed<SectionKey>(() => {
  const section = route.meta.appSection
  return isSectionKey(section) ? section : 'home'
})
const availableModelIds = computed(() => chatModels.value.map((model) => model.id))
const activeChatModel = computed(() => {
  if (!hasChat.value) return ''
  if (selectedModelId.value && availableModelIds.value.includes(selectedModelId.value)) return selectedModelId.value
  return defaultTextModel.value || availableModelIds.value[0] || ''
})
const imageCapabilityAvailable = computed(() => false)
const webSearchAvailable = computed(() => appStore.cachedPublicSettings?.web_search?.available === true)
const textBetaMode = computed(() => hasChat.value)
const activeModelLabel = computed(() =>
  chatModels.value.find((model) => model.id === activeChatModel.value)?.name || 'Deepseek-V4-Flash'
)
const workspaceIntent = computed<WorkspaceIntent>(() =>
  textBetaMode.value ? 'chat' : activeSection.value
)
const activeContent = computed<SectionContent>(() => {
  if (textBetaMode.value) {
    return {
      shellTitle: 'SSXZ AI',
      shellSubtitle: `当前开放 ${activeModelLabel.value} 文本对话。`,
      eyebrow: '文本对话 beta',
      icon: 'chat'
    }
  }
  return sectionContent[activeSection.value]
})
const emptyStateCopy = computed(() => {
  if (textBetaMode.value) return `当前开放 ${activeModelLabel.value} 文本对话。直接输入问题，开始对话。`
  if (activeSection.value === 'image') return '输入你想处理的图像需求。'
  if (activeSection.value === 'chat') return '输入问题后，这段对话会进入左侧历史，刷新页面也不会丢失。'
  return '直接输入问题，开始对话。'
})

function isSectionKey(value: unknown): value is SectionKey {
  return typeof value === 'string' && sectionKeys.includes(value as SectionKey)
}

async function handleFiles(files: File[]) {
  if (!imageCapabilityAvailable.value) {
    assets.clearPreviews()
    workspace.errorMessage.value = WORKSPACE_TEXT_ONLY_MESSAGE
    return
  }

  await assets.addFiles(files)
}

function handleUnsupportedFiles() {
  assets.clearPreviews()
  workspace.errorMessage.value = WORKSPACE_TEXT_ONLY_MESSAGE
}

async function submitDraft() {
  const text = draft.value.trim()
  if (!text && assets.previews.value.length === 0) return
  if (!activeChatModel.value || workspace.sending.value || assets.registering.value) return
  if (assets.previews.value.length > 0 && !imageCapabilityAvailable.value) {
    workspace.errorMessage.value = WORKSPACE_TEXT_ONLY_MESSAGE
    return
  }

  const sent = await workspace.sendTextMessage({
    text,
    model: activeChatModel.value,
    intent: workspaceIntent.value,
    attachments: assets.getLocalAttachments(),
    webSearchRequested: webSearchAvailable.value && webSearchRequested.value
  })
  if (sent) {
    draft.value = ''
    assets.clearPreviews()
    webSearchRequested.value = false
  }
}

async function selectConversation(id: number) {
  draft.value = ''
  assets.clearPreviews()
  webSearchRequested.value = false
  await workspace.selectConversation(id)
}

async function startNewChat() {
  draft.value = ''
  assets.clearPreviews()
  webSearchRequested.value = false
  await workspace.startNewChat()
}

function toggleWebSearch() {
  if (!webSearchAvailable.value) return
  webSearchRequested.value = !webSearchRequested.value
}

onMounted(async () => {
  await Promise.all([
    loadCapabilities(),
    workspace.loadHistory()
  ])
})

watch(chatModels, (models) => {
  if (!models.length) {
    selectedModelId.value = ''
    return
  }
  if (!models.some((model) => model.id === selectedModelId.value)) {
    selectedModelId.value = defaultTextModel.value || models[0].id
  }
}, { immediate: true })

watch(webSearchAvailable, (available) => {
  if (!available) {
    webSearchRequested.value = false
  }
}, { immediate: true })
</script>

<style scoped>
:deep(.ssxz-page-heading) {
  display: none;
}

.workspace-page {
  display: grid;
  min-height: calc(100vh - 4rem);
  grid-template-rows: minmax(0, 1fr) auto auto;
  gap: 1rem;
  padding: 1rem 0 max(2rem, env(safe-area-inset-bottom));
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
  width: min(46rem, 100%);
  padding: 1rem;
  text-align: center;
}

.empty-state h1 {
  margin: 0;
  color: var(--ssxz-text);
  font-size: clamp(2.25rem, 4.6vw, 3.6rem);
  font-weight: 760;
  line-height: 1.06;
  text-wrap: balance;
  word-break: keep-all;
}

.empty-state p,
.workspace-error {
  margin: 1rem auto 0;
  max-width: 42rem;
  color: var(--ssxz-subtle);
  font-size: 1rem;
  line-height: 1.75;
}

.workspace-error {
  margin-top: 0;
  border: 1px solid color-mix(in srgb, #ef4444 36%, transparent);
  border-radius: 0.75rem;
  background: color-mix(in srgb, #ef4444 9%, var(--ssxz-surface));
  color: #b91c1c;
  padding: 0.75rem 1rem;
}

.workspace-notice {
  margin: 0 auto;
  max-width: 42rem;
  border: 1px solid color-mix(in srgb, var(--ssxz-primary) 28%, transparent);
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--ssxz-primary) 8%, var(--ssxz-surface));
  color: var(--ssxz-text);
  padding: 0.75rem 1rem;
  text-align: center;
}

.composer-zone {
  position: sticky;
  bottom: max(1rem, env(safe-area-inset-bottom));
  z-index: 6;
  width: min(56rem, 100%);
  margin: 0 auto;
}

:deep(.message-list) {
  display: grid;
  width: min(56rem, 100%);
  gap: 1rem;
}

:deep(.workspace-loading) {
  color: var(--ssxz-subtle);
  font-size: 0.9rem;
  text-align: center;
}

:deep(.message-row) {
  display: grid;
  grid-template-columns: 2.25rem minmax(0, 1fr);
  gap: 0.75rem;
  align-items: start;
  width: min(48rem, 100%);
}

:deep(.message-row[data-role='user']) {
  grid-template-columns: minmax(0, 1fr) 2.25rem;
  justify-self: end;
}

:deep(.message-row[data-role='user'] .message-avatar) {
  grid-column: 2;
  grid-row: 1;
}

:deep(.message-row[data-role='user'] .message-bubble) {
  grid-column: 1;
  justify-self: end;
  background: var(--ssxz-primary);
  color: #fff;
}

:deep(.message-avatar) {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface);
  color: var(--ssxz-text);
}

:deep(.message-bubble) {
  max-width: min(40rem, 100%);
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface);
  color: var(--ssxz-text);
  padding: 0.9rem 1rem;
  box-shadow: 0 16px 45px color-mix(in srgb, #0f172a 8%, transparent);
}

:deep(.message-row[data-state='failed'] .message-bubble) {
  border-color: color-mix(in srgb, #ef4444 34%, transparent);
}

:deep(.message-row[data-state='sending'] .message-bubble),
:deep(.message-row[data-state='generating'] .message-bubble) {
  border-color: color-mix(in srgb, var(--ssxz-primary) 28%, transparent);
}

:deep(.message-state) {
  margin: 0 0 0.35rem;
  color: var(--ssxz-subtle);
  font-size: 0.74rem;
  line-height: 1.3;
}

:deep(.message-bubble p) {
  margin: 0;
  white-space: pre-wrap;
  line-height: 1.75;
}

:deep(.message-attachments) {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
  margin-bottom: 0.65rem;
}

:deep(.message-attachments figure) {
  margin: 0;
}

:deep(.message-attachments img) {
  width: 9rem;
  height: 6.5rem;
  object-fit: cover;
  border-radius: 0.75rem;
}

:deep(.message-attachments figcaption) {
  max-width: 9rem;
  margin-top: 0.25rem;
  overflow: hidden;
  color: var(--ssxz-subtle);
  font-size: 0.72rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.composer-card) {
  display: grid;
  gap: 0.75rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1.25rem;
  background: color-mix(in srgb, var(--ssxz-surface) 94%, transparent);
  box-shadow: 0 24px 70px color-mix(in srgb, #0f172a 16%, transparent);
  padding: 0.8rem;
  backdrop-filter: blur(18px);
}

:deep(.composer-card textarea) {
  width: 100%;
  min-height: 4.25rem;
  resize: vertical;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 1rem;
  line-height: 1.65;
}

:deep(.composer-card textarea::placeholder) {
  color: var(--ssxz-subtle);
}

:deep(.attachment-preview-list) {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
}

:deep(.attachment-preview-card) {
  position: relative;
  display: grid;
  grid-template-columns: 3.75rem minmax(0, 8rem) auto;
  gap: 0.55rem;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-muted);
  padding: 0.4rem 0.5rem;
}

:deep(.attachment-preview-card img) {
  width: 3.75rem;
  height: 3rem;
  object-fit: cover;
  border-radius: 0.6rem;
}

:deep(.attachment-preview-card span) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--ssxz-text);
  font-size: 0.8rem;
}

:deep(.attachment-preview-card small) {
  color: var(--ssxz-subtle);
  font-size: 0.72rem;
}

:deep(.remove-preview) {
  display: grid;
  width: 1.75rem;
  height: 1.75rem;
  place-items: center;
  border: 0;
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-surface) 85%, transparent);
  color: var(--ssxz-subtle);
  cursor: pointer;
}

:deep(.asset-panel) {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.55rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-muted);
  padding: 0.65rem;
}

:deep(.asset-option) {
  position: relative;
  display: flex;
  min-height: 4rem;
  gap: 0.6rem;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.75rem;
  background: var(--ssxz-surface);
  color: var(--ssxz-text);
  cursor: pointer;
  padding: 0.65rem;
  text-align: left;
}

:deep(.asset-option small),
:deep(.asset-panel-note),
:deep(.asset-panel-error) {
  color: var(--ssxz-subtle);
  font-size: 0.74rem;
}

:deep(.asset-panel-error) {
  grid-column: 1 / -1;
  margin: 0;
  color: #b91c1c;
}

:deep(.asset-panel-note) {
  grid-column: 1 / -1;
  margin: 0;
}

:deep(.asset-file-input) {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

:deep(.composer-tool-row) {
  display: flex;
  min-height: 2.6rem;
  flex-wrap: wrap;
  gap: 0.45rem;
  align-items: center;
}

:deep(.model-selector) {
  position: relative;
}

:deep(.model-trigger),
:deep(.toolbar-tool),
:deep(.send-button) {
  display: inline-flex;
  min-height: 2.35rem;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface);
  color: var(--ssxz-text);
  cursor: pointer;
  padding: 0 0.75rem;
  font-size: 0.84rem;
}

:deep(.model-menu) {
  position: absolute;
  bottom: calc(100% + 0.5rem);
  left: 0;
  z-index: 20;
  display: grid;
  min-width: 14rem;
  gap: 0.25rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.9rem;
  background: var(--ssxz-surface);
  box-shadow: 0 20px 55px color-mix(in srgb, #0f172a 18%, transparent);
  padding: 0.45rem;
}

:deep(.model-option) {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  border: 0;
  border-radius: 0.65rem;
  background: transparent;
  color: var(--ssxz-text);
  cursor: pointer;
  padding: 0.55rem 0.65rem;
  text-align: left;
}

:deep(.model-option.is-selected),
:deep(.model-option:hover),
:deep(.toolbar-tool:hover),
:deep(.model-trigger:hover) {
  background: color-mix(in srgb, var(--ssxz-primary) 10%, transparent);
}

:deep(.toolbar-tool.is-unavailable),
:deep(.toolbar-tool:disabled),
:deep(.asset-option.is-unavailable) {
  cursor: not-allowed;
  opacity: 0.72;
  background: color-mix(in srgb, var(--ssxz-surface-muted) 88%, transparent);
  color: var(--ssxz-subtle);
}

:deep(.toolbar-tool.is-unavailable:hover),
:deep(.toolbar-tool:disabled:hover),
:deep(.asset-option.is-unavailable:hover) {
  background: color-mix(in srgb, var(--ssxz-surface-muted) 88%, transparent);
}

:deep(.toolbar-tool.is-active) {
  border-color: color-mix(in srgb, var(--ssxz-primary) 50%, transparent);
  background: color-mix(in srgb, var(--ssxz-primary) 12%, transparent);
  color: var(--ssxz-primary);
}

:deep(.toolbar-tool small) {
  color: var(--ssxz-subtle);
  font-size: 0.68rem;
}

:deep(.send-button) {
  margin-left: auto;
  width: 2.45rem;
  padding: 0;
  background: var(--ssxz-primary);
  color: #fff;
}

:deep(.send-button:disabled),
:deep(.model-trigger:disabled) {
  cursor: not-allowed;
  opacity: 0.5;
}

@media (max-width: 720px) {
  .workspace-page {
    min-height: calc(100dvh - 4rem);
    padding: 0.5rem 0 1rem;
  }

  .empty-state h1 {
    font-size: 2.35rem;
  }

  .composer-zone {
    width: min(100%, calc(100vw - 1rem));
    bottom: max(0.75rem, env(safe-area-inset-bottom));
  }

  :deep(.asset-panel) {
    grid-template-columns: 1fr 1fr;
  }

  :deep(.message-row),
  :deep(.message-row[data-role='user']) {
    grid-template-columns: 1fr;
  }

  :deep(.message-avatar) {
    display: none;
  }
}
</style>
