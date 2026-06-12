<template>
  <div class="message-list" aria-live="polite">
    <p v-if="loading" class="workspace-loading">Loading conversation...</p>

    <article
      v-for="message in messages"
      :key="message.id"
      class="message-row"
      :data-role="message.role"
      :data-state="message.state || 'ready'"
    >
      <div class="message-avatar">
        <Icon :name="message.role === 'assistant' ? 'sparkles' : 'userCircle'" size="sm" />
      </div>
      <div class="message-bubble">
        <div
          v-if="message.messageType === 'image'"
          class="message-image-card"
          :data-image-state="message.state || 'completed'"
        >
          <div v-if="message.attachments?.length" class="message-attachments">
            <figure v-for="attachment in message.attachments" :key="attachment.id">
              <img :src="attachment.url" :alt="attachment.name" />
              <figcaption>{{ attachment.name }}</figcaption>
            </figure>
          </div>
          <p>{{ message.content || fallbackText(message) }}</p>
        </div>
        <div v-else-if="message.attachments?.length" class="message-attachments">
          <figure v-for="attachment in message.attachments" :key="attachment.id">
            <img :src="attachment.url" :alt="attachment.name" />
            <figcaption>{{ attachment.name }}</figcaption>
          </figure>
        </div>
        <p v-if="message.messageType !== 'image'">{{ message.content || fallbackText(message) }}</p>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import type { WorkspaceMessage } from './useWorkspaceConversation'

defineProps<{
  messages: WorkspaceMessage[]
  loading: boolean
}>()

function fallbackText(message: WorkspaceMessage) {
  if (message.state === 'loading') return 'Image generation is still running.'
  if (message.state === 'error') return 'Image generation failed. Please try again.'
  if (message.messageType === 'image' && message.attachments?.length) return 'Generated image'
  if (message.attachments?.length) return 'Image attached'
  return ''
}
</script>
