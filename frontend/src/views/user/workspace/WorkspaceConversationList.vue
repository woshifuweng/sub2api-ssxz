<template>
  <nav class="workspace-conversation-list" aria-label="历史对话">
    <p v-if="loading" class="ssxz-empty-history ssxz-sidebar-text">正在同步历史...</p>
    <button
      v-for="conversation in conversations"
      :key="conversation.id"
      type="button"
      class="ssxz-history-item"
      :class="{ 'is-active': conversation.id === activeConversationId }"
      @click="$emit('select', conversation.id)"
    >
      <Icon name="chat" size="sm" />
      <span>{{ conversation.title || '未命名对话' }}</span>
    </button>
    <p v-if="!loading && conversations.length === 0" class="ssxz-empty-history ssxz-sidebar-text">
      暂无历史对话
    </p>
  </nav>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import type { ChatConversation } from '@/api/chatWorkspace'

defineProps<{
  conversations: ChatConversation[]
  activeConversationId: number | null
  loading: boolean
}>()

defineEmits<{
  (event: 'select', id: number): void
}>()
</script>
