<template>
  <AppLayout>
    <div class="space-y-6">
      <header>
        <p class="text-xs font-semibold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
          {{ t('chat.eyebrow') }}
        </p>
        <h1 class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ t('chat.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('chat.description') }}</p>
      </header>

      <div class="grid min-h-[68vh] gap-5 xl:grid-cols-[17rem_minmax(0,1fr)]">
        <aside class="card flex min-h-0 flex-col p-4">
          <button type="button" class="btn btn-primary w-full justify-center" @click="startNewChat">
            <Icon name="plus" size="sm" />
            {{ t('chat.newConversation') }}
          </button>

          <div class="mt-4 min-h-0 flex-1 space-y-1 overflow-y-auto" :aria-label="t('chat.history')">
            <p v-if="workspace.loadingHistory.value" class="px-3 py-4 text-sm text-gray-500 dark:text-gray-400">
              {{ t('chat.loadingHistory') }}
            </p>
            <button
              v-for="conversation in workspace.conversations.value"
              :key="conversation.id"
              type="button"
              class="flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-left text-sm transition-colors"
              :class="conversation.id === workspace.activeConversationId.value
                ? 'bg-primary-50 font-medium text-primary-700 dark:bg-primary-950/40 dark:text-primary-300'
                : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-800'"
              @click="selectConversation(conversation.id)"
            >
              <Icon name="chat" size="sm" class="shrink-0" />
              <span class="truncate">{{ conversation.title || t('chat.untitled') }}</span>
            </button>
            <p
              v-if="!workspace.loadingHistory.value && workspace.conversations.value.length === 0"
              class="px-3 py-4 text-sm text-gray-500 dark:text-gray-400"
            >
              {{ t('chat.noHistory') }}
            </p>
          </div>
        </aside>

        <section class="card flex min-h-[68vh] min-w-0 flex-col overflow-hidden">
          <div class="flex min-h-0 flex-1 flex-col overflow-y-auto p-4 md:p-6">
            <div
              v-if="workspace.messages.value.length === 0 && !workspace.loadingMessages.value"
              class="m-auto max-w-lg py-16 text-center"
            >
              <span class="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-300">
                <Icon name="chat" size="lg" />
              </span>
              <h2 class="mt-4 text-xl font-semibold text-gray-900 dark:text-white">{{ t('chat.emptyTitle') }}</h2>
              <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('chat.emptyDescription') }}</p>
            </div>

            <p v-if="workspace.loadingMessages.value" class="m-auto text-sm text-gray-500 dark:text-gray-400">
              {{ t('chat.loadingMessages') }}
            </p>

            <div v-else class="mx-auto w-full max-w-4xl space-y-4" aria-live="polite">
              <article
                v-for="message in workspace.messages.value"
                :key="message.id"
                class="flex gap-3"
                :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
              >
                <div
                  class="max-w-[88%] rounded-2xl px-4 py-3 text-sm leading-6 md:max-w-[78%]"
                  :class="message.role === 'user'
                    ? 'rounded-br-md bg-primary-600 text-white'
                    : 'rounded-bl-md bg-gray-100 text-gray-800 dark:bg-dark-800 dark:text-gray-100'"
                >
                  <p class="whitespace-pre-wrap break-words">{{ message.content }}</p>
                  <span
                    v-if="message.state === 'sending' || message.state === 'generating'"
                    class="mt-1 block text-xs opacity-70"
                  >
                    {{ message.state === 'sending' ? t('chat.sending') : t('chat.generating') }}
                  </span>
                </div>
              </article>
            </div>
          </div>

          <div class="border-t border-gray-200 p-4 dark:border-dark-700 md:p-5">
            <div v-if="workspace.errorMessage.value" class="mb-3 flex items-center justify-between gap-3 rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300" role="alert">
              <span>{{ workspace.errorMessage.value }}</span>
              <button
                v-if="workspace.canRetryLastFailedSend.value"
                type="button"
                class="shrink-0 font-medium underline"
                :disabled="workspace.sending.value"
                @click="retryLastFailedSend"
              >
                {{ t('chat.retry') }}
              </button>
            </div>

            <form class="mx-auto max-w-4xl" @submit.prevent="submitDraft">
              <div class="mb-3 flex flex-wrap items-center gap-3">
                <label for="chat-model" class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('chat.model') }}</label>
                <select
                  id="chat-model"
                  v-model="selectedModelId"
                  class="input min-w-52 flex-1 sm:flex-none"
                  :disabled="capabilities.loading.value || chatModels.length === 0"
                >
                  <option v-if="chatModels.length === 0" value="">{{ t('chat.noModel') }}</option>
                  <option v-for="model in chatModels" :key="model.id" :value="model.id">
                    {{ model.name || model.id }}
                  </option>
                </select>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('chat.textOnly') }}</span>
              </div>
              <div class="flex items-end gap-3 rounded-2xl border border-gray-300 bg-white p-2 shadow-sm focus-within:border-primary-500 focus-within:ring-2 focus-within:ring-primary-500/20 dark:border-dark-600 dark:bg-dark-850">
                <textarea
                  v-model="draft"
                  rows="2"
                  class="min-h-14 flex-1 resize-none border-0 bg-transparent px-2 py-2 text-sm text-gray-900 outline-none placeholder:text-gray-400 dark:text-white"
                  :placeholder="t('chat.placeholder')"
                  :disabled="workspace.sending.value"
                  @keydown.enter.exact.prevent="submitDraft"
                />
                <button
                  type="submit"
                  class="btn btn-primary h-10 w-10 shrink-0 justify-center p-0"
                  :disabled="!canSend"
                  :aria-label="t('chat.send')"
                  :title="t('chat.send')"
                >
                  <Icon :name="workspace.sending.value ? 'sync' : 'arrowUp'" size="sm" />
                </button>
              </div>
            </form>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useUserCapabilities } from '@/composables/useUserCapabilities'
import { useWorkspaceConversation } from './workspace/useWorkspaceConversation'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const draft = ref('')
const selectedModelId = ref('')
const workspace = useWorkspaceConversation()
const capabilities = useUserCapabilities()
const chatModels = computed(() => capabilities.chatModels.value)
const canSend = computed(() =>
  !workspace.sending.value &&
  draft.value.trim().length > 0 &&
  selectedModelId.value.length > 0
)

async function submitDraft() {
  if (!canSend.value) return
  const text = draft.value.trim()
  const sent = await workspace.sendTextMessage({
    text,
    model: selectedModelId.value,
    intent: 'chat',
    attachments: []
  })
  if (!sent) return
  draft.value = ''
  await syncConversationQuery(workspace.activeConversationId.value)
}

async function retryLastFailedSend() {
  const sent = await workspace.retryLastFailedSend()
  if (sent) {
    await syncConversationQuery(workspace.activeConversationId.value)
  }
}

async function selectConversation(id: number) {
  await workspace.selectConversation(id)
  if (workspace.activeConversationId.value === id) {
    await syncConversationQuery(id)
  }
}

async function startNewChat() {
  draft.value = ''
  await workspace.startNewChat()
  await syncConversationQuery(null)
}

async function syncConversationQuery(id: number | null) {
  const query = { ...route.query }
  if (id === null) {
    delete query.conversation_id
  } else {
    query.conversation_id = String(id)
  }
  await router.replace({ query })
}

function routeConversationID() {
  const raw = Array.isArray(route.query.conversation_id)
    ? route.query.conversation_id[0]
    : route.query.conversation_id
  const id = Number(raw)
  return Number.isInteger(id) && id > 0 ? id : null
}

watch(chatModels, (models) => {
  if (models.some((model) => model.id === selectedModelId.value)) return
  selectedModelId.value = capabilities.defaultTextModel.value || models[0]?.id || ''
}, { immediate: true })

onMounted(async () => {
  await Promise.all([capabilities.loadCapabilities(), workspace.loadHistory()])
  const conversationID = routeConversationID()
  if (conversationID !== null) {
    await selectConversation(conversationID)
  }
})
</script>
