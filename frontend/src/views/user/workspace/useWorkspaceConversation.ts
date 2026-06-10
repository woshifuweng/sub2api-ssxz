import { computed, ref } from 'vue'
import {
  CHAT_MESSAGE_TYPE_TEXT,
  appendMessage,
  chatWorkspaceBackendEnabled,
  createConversation,
  listConversations,
  listMessages,
  type ChatAsset,
  type ChatConversation,
  type ChatMessage
} from '@/api/chatWorkspace'

export type WorkspaceIntent = 'home' | 'chat' | 'image'
export type WorkspaceMessageState = 'loading' | 'error'

export const WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE =
  '统一工作台后端正在接入，暂不可发送。当前仅展示工作台入口。'
export const WORKSPACE_TEXT_ONLY_MESSAGE =
  '统一工作台 v1 当前仅支持文本对话。图片和文件会在 asset/task 后端就绪后接入。'
export const WORKSPACE_HISTORY_FAILED_MESSAGE = '工作台历史暂时无法加载。'
export const WORKSPACE_MESSAGES_FAILED_MESSAGE = '该对话暂时无法加载。'
export const WORKSPACE_SEND_FAILED_MESSAGE = '消息保存失败，请稍后重试。'

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
    errorMessage.value = ''
    if (!backendEnabled.value) {
      conversations.value = []
      activeConversationId.value = null
      loadingHistory.value = false
      loadingMessages.value = false
      return
    }

    loadingHistory.value = true
    try {
      conversations.value = await listConversations()
      if (
        activeConversationId.value !== null &&
        !conversations.value.some((item) => item.id === activeConversationId.value)
      ) {
        activeConversationId.value = null
        messages.value = []
      }
    } catch {
      conversations.value = []
      activeConversationId.value = null
      messages.value = []
      errorMessage.value = WORKSPACE_HISTORY_FAILED_MESSAGE
    } finally {
      loadingHistory.value = false
      loadingMessages.value = false
    }
  }

  async function selectConversation(id: number) {
    if (!backendEnabled.value) {
      errorMessage.value = WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE
      return
    }
    if (id <= 0 || loadingMessages.value) return

    errorMessage.value = ''
    loadingMessages.value = true
    try {
      const nextMessages = await listMessages(id)
      activeConversationId.value = id
      messages.value = nextMessages.map(mapChatMessageToWorkspaceMessage)
    } catch {
      activeConversationId.value = null
      messages.value = []
      errorMessage.value = WORKSPACE_MESSAGES_FAILED_MESSAGE
    } finally {
      loadingMessages.value = false
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

    if (!isTextChatIntent(input.intent) || input.attachments.length > 0) {
      errorMessage.value = WORKSPACE_TEXT_ONLY_MESSAGE
      return false
    }

    sending.value = true
    errorMessage.value = ''
    try {
      let conversationId = activeConversationId.value
      if (conversationId === null) {
        const conversation = await createConversation({ title: deriveConversationTitle(text) })
        conversationId = conversation.id
        activeConversationId.value = conversation.id
        upsertConversation(conversation)
      }

      const savedMessage = await appendMessage(conversationId, {
        message_type: CHAT_MESSAGE_TYPE_TEXT,
        role: 'user',
        content: text,
        model: input.model,
        intent: 'chat'
      })
      messages.value = [...messages.value, mapChatMessageToWorkspaceMessage(savedMessage)]
      await refreshConversationList()
      return true
    } catch {
      errorMessage.value = WORKSPACE_SEND_FAILED_MESSAGE
      return false
    } finally {
      sending.value = false
    }
  }

  async function refreshConversationList() {
    try {
      conversations.value = await listConversations()
    } catch {
      // Sending already succeeded; keep the active conversation visible if sidebar refresh fails.
    }
  }

  function upsertConversation(conversation: ChatConversation) {
    const exists = conversations.value.some((item) => item.id === conversation.id)
    conversations.value = exists
      ? conversations.value.map((item) => (item.id === conversation.id ? conversation : item))
      : [conversation, ...conversations.value]
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

function mapChatMessageToWorkspaceMessage(message: ChatMessage): WorkspaceMessage {
  return {
    id: `message-${message.id}`,
    persistedId: message.id,
    conversationId: message.conversation_id,
    messageType: message.message_type,
    role: message.role === 'assistant' ? 'assistant' : 'user',
    content: message.content,
    createdAt: message.created_at
  }
}

function deriveConversationTitle(text: string) {
  const trimmed = text.trim()
  if (!trimmed) return ''
  return Array.from(trimmed).slice(0, 40).join('')
}

function isTextChatIntent(intent: WorkspaceIntent) {
  return intent === 'home' || intent === 'chat'
}
