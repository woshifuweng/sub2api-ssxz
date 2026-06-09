import { computed, ref } from 'vue'
import {
  CHAT_MESSAGE_TYPE_TEXT,
  chatWorkspaceBackendEnabled,
  type ChatAsset,
  type ChatConversation
} from '@/api/chatWorkspace'

export type WorkspaceIntent = 'home' | 'chat' | 'image'
export type WorkspaceMessageState = 'loading' | 'error'

export const WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE =
  '统一工作台后端正在接入，暂不可发送。当前仅展示工作台入口。'

export interface WorkspaceAttachment {
  id: string
  name: string
  url: string
  type: 'image'
  asset?: ChatAsset
}

export interface WorkspaceMessage {
  id: string
  persistedId?: number
  conversationId?: number
  messageType: string
  role: 'user' | 'assistant'
  content: string
  state?: WorkspaceMessageState
  attachments?: WorkspaceAttachment[]
  createdAt?: string
}

export interface SendTextMessageInput {
  text: string
  model: string
  intent: WorkspaceIntent
  attachments: WorkspaceAttachment[]
}

interface UseWorkspaceConversationOptions {
  backendEnabled?: boolean
}

export function useWorkspaceConversation(options: UseWorkspaceConversationOptions = {}) {
  const backendEnabled = ref(options.backendEnabled ?? chatWorkspaceBackendEnabled)
  const conversations = ref<ChatConversation[]>([])
  const activeConversationId = ref<number | null>(null)
  const messages = ref<WorkspaceMessage[]>([])
  const loadingHistory = ref(false)
  const loadingMessages = ref(false)
  const sending = ref(false)
  const errorMessage = ref('')

  const activeConversation = computed(() =>
    conversations.value.find((item) => item.id === activeConversationId.value) || null
  )

  async function loadHistory() {
    conversations.value = []
    activeConversationId.value = null
    loadingHistory.value = false
    loadingMessages.value = false
    errorMessage.value = ''
  }

  async function selectConversation(_id: number) {
    if (!backendEnabled.value) {
      errorMessage.value = WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE
      return
    }
  }

  async function startNewChat() {
    activeConversationId.value = null
    messages.value = []
    errorMessage.value = ''
  }

  async function sendTextMessage(input: SendTextMessageInput) {
    const text = input.text.trim()
    if (!text && input.attachments.length === 0) return false
    if (sending.value) return false

    if (!backendEnabled.value) {
      errorMessage.value = WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE
      return false
    }

    return false
  }

  return {
    activeConversation,
    activeConversationId,
    backendEnabled,
    conversations,
    errorMessage,
    loadingHistory,
    loadingMessages,
    messages,
    sending,
    loadHistory,
    selectConversation,
    sendTextMessage,
    startNewChat
  }
}

export function createLocalWorkspaceMessage(
  role: 'user' | 'assistant',
  content: string,
  attachments: WorkspaceAttachment[]
): WorkspaceMessage {
  return {
    id: `${role}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    messageType: CHAT_MESSAGE_TYPE_TEXT,
    role,
    content,
    attachments
  }
}
