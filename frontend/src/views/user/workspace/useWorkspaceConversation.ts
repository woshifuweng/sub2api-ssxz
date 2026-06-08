import { computed, ref } from 'vue'
import { apiClient } from '@/api/client'
import {
  CHAT_MESSAGE_TYPE_ERROR_CARD,
  CHAT_MESSAGE_TYPE_TEXT,
  type ChatAsset,
  type ChatConversation,
  type ChatMessage,
  appendMessage,
  createConversation,
  listConversations,
  listMessages
} from '@/api/chatWorkspace'

export type WorkspaceIntent = 'home' | 'chat' | 'image'
export type WorkspaceMessageState = 'loading' | 'error'

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

interface ChatStudioResponse {
  choices?: Array<{
    message?: {
      content?: string | Array<{ text?: string; content?: string }>
    }
  }>
}

export function useWorkspaceConversation() {
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
    loadingHistory.value = true
    errorMessage.value = ''
    try {
      conversations.value = await listConversations()
    } catch (error) {
      errorMessage.value = normalizeErrorMessage(error, '无法读取历史会话')
    } finally {
      loadingHistory.value = false
    }
  }

  async function selectConversation(id: number) {
    activeConversationId.value = id
    loadingMessages.value = true
    errorMessage.value = ''
    try {
      const persisted = await listMessages(id)
      messages.value = persisted.map(mapPersistedMessage)
    } catch (error) {
      errorMessage.value = normalizeErrorMessage(error, '无法读取会话消息')
    } finally {
      loadingMessages.value = false
    }
  }

  async function startNewChat() {
    activeConversationId.value = null
    messages.value = []
    errorMessage.value = ''
  }

  async function ensureConversationForAssets(title: string) {
    return ensureConversation(title || '新对话')
  }

  async function sendTextMessage(input: SendTextMessageInput) {
    const text = input.text.trim()
    if (!text && input.attachments.length === 0) return
    if (sending.value) return

    sending.value = true
    errorMessage.value = ''

    const localUser = createLocalMessage('user', text, input.attachments)
    messages.value.push(localUser)

    try {
      const conversationId = await ensureConversation(text || '图片任务')
      const persistedUser = await appendMessage(conversationId, {
        message_type: CHAT_MESSAGE_TYPE_TEXT,
        role: 'user',
        content: text,
        metadata: {
          intent: input.intent,
          model: input.model,
          attachments: input.attachments.map((attachment) => ({
            name: attachment.name,
            type: attachment.type,
            asset_id: attachment.asset?.id,
            url: attachment.url
          }))
        }
      })
      patchPersisted(localUser.id, persistedUser)

      const loadingAssistant = createLocalMessage('assistant', '正在思考...', [])
      loadingAssistant.state = 'loading'
      messages.value.push(loadingAssistant)

      try {
        const completion = await requestChatCompletion(buildCompletionMessages(), input.model)
        const assistantText = extractAssistantText(completion)
        const persistedAssistant = await appendMessage(conversationId, {
          message_type: CHAT_MESSAGE_TYPE_TEXT,
          role: 'assistant',
          content: assistantText,
          metadata: { intent: input.intent, model: input.model }
        })
        patchPersisted(loadingAssistant.id, persistedAssistant)
      } catch (error) {
        const assistantText = normalizeErrorMessage(error, '当前模型服务暂不可用，请稍后重试。')
        const persistedAssistant = await appendMessage(conversationId, {
          message_type: CHAT_MESSAGE_TYPE_ERROR_CARD,
          role: 'assistant',
          content: assistantText,
          metadata: { intent: input.intent, model: input.model }
        })
        patchPersisted(loadingAssistant.id, persistedAssistant, 'error')
      }

      await loadHistory()
    } finally {
      sending.value = false
    }
  }

  async function ensureConversation(title: string) {
    if (activeConversationId.value) return activeConversationId.value
    const conversation = await createConversation({ title: title.slice(0, 80) })
    conversations.value = [conversation, ...conversations.value.filter((item) => item.id !== conversation.id)]
    activeConversationId.value = conversation.id
    return conversation.id
  }

  function patchPersisted(localId: string, persisted: ChatMessage, state?: WorkspaceMessageState) {
    const index = messages.value.findIndex((message) => message.id === localId)
    if (index === -1) return
    messages.value[index] = {
      ...messages.value[index],
      persistedId: persisted.id,
      conversationId: persisted.conversation_id,
      messageType: persisted.message_type,
      role: persisted.role === 'assistant' ? 'assistant' : 'user',
      content: persisted.content,
      state,
      createdAt: persisted.created_at
    }
  }

  function buildCompletionMessages() {
    return messages.value
      .filter((message) => message.messageType === CHAT_MESSAGE_TYPE_TEXT && !message.state && message.content.trim())
      .map((message) => ({
        role: message.role,
        content: message.content.trim()
      }))
      .slice(-40)
  }

  return {
    activeConversation,
    activeConversationId,
    conversations,
    errorMessage,
    loadingHistory,
    loadingMessages,
    messages,
    sending,
    ensureConversationForAssets,
    loadHistory,
    selectConversation,
    sendTextMessage,
    startNewChat
  }
}

function mapPersistedMessage(message: ChatMessage): WorkspaceMessage {
  return {
    id: `persisted-${message.id}`,
    persistedId: message.id,
    conversationId: message.conversation_id,
    messageType: message.message_type,
    role: message.role === 'assistant' ? 'assistant' : 'user',
    content: message.content,
    createdAt: message.created_at,
    state: message.message_type === CHAT_MESSAGE_TYPE_ERROR_CARD ? 'error' : undefined
  }
}

function createLocalMessage(
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

async function requestChatCompletion(payloadMessages: Array<{ role: string; content: string }>, model: string) {
  const { data } = await apiClient.post<ChatStudioResponse>('/chat-studio/complete', {
    model,
    mode: 'general',
    messages: payloadMessages,
    temperature: 0.7
  })
  return data
}

function extractAssistantText(payload: ChatStudioResponse): string {
  const content = payload?.choices?.[0]?.message?.content
  if (typeof content === 'string' && content.trim()) return content.trim()
  if (Array.isArray(content)) {
    const text = content.map((item) => item?.text || item?.content || '').filter(Boolean).join('\n').trim()
    if (text) return text
  }
  return '已收到回复，但当前页面无法展示返回内容。'
}

function normalizeErrorMessage(error: unknown, fallback: string) {
  if (typeof error === 'object' && error !== null) {
    const value = error as { message?: unknown; response?: { data?: { message?: unknown; detail?: unknown } } }
    const message = value.response?.data?.message || value.response?.data?.detail || value.message
    if (typeof message === 'string' && message.trim()) return message
  }
  return fallback
}
