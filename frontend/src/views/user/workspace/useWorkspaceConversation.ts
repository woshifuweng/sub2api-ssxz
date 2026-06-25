import { computed, ref } from 'vue'
import {
  CHAT_MESSAGE_TYPE_IMAGE,
  CHAT_MESSAGE_TYPE_TEXT,
  appendMessage,
  chatWorkspaceBackendEnabled,
  createConversation,
  listConversations,
  listMessages,
  type ChatAsset,
  type ChatConversation,
  type ChatWorkspaceError,
  type ChatMessage
} from '@/api/chatWorkspace'

export type WorkspaceIntent = 'home' | 'chat' | 'image'
export type WorkspaceRequestPhase = 'idle' | 'sending' | 'generating' | 'success' | 'failed'
export type WorkspaceMessageState = 'sending' | 'generating' | 'success' | 'failed'

export const WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE =
  '统一工作台后端正在接入，暂不可发送。当前仅展示工作台入口。'
export const WORKSPACE_TEXT_ONLY_MESSAGE =
  '当前对话页是文本 beta，暂不支持发送图片、文件、图片理解或多图分析。请到 AI 作图页生成图片，或先只发送文字。'
export const WORKSPACE_HISTORY_FAILED_MESSAGE = '工作台历史暂时无法加载。'
export const WORKSPACE_MESSAGES_FAILED_MESSAGE = '该对话暂时无法加载。'
export const WORKSPACE_SEND_FAILED_MESSAGE = '发送失败，请稍后重试。'
export const WORKSPACE_MODEL_UNAVAILABLE_MESSAGE =
  '当前模型暂不可用，请切换模型或联系管理员检查模型、API Key、分组和上游账号配置。'
export const WORKSPACE_PROVIDER_FAILED_MESSAGE =
  '消息可能已提交，但 AI 回复失败，请稍后重试或切换模型。'
export const WORKSPACE_REFRESH_AFTER_SEND_FAILED_MESSAGE =
  '消息已提交，但刷新会话失败，请刷新页面后查看。'

export const WORKSPACE_GENERATING_MESSAGE = 'AI response is being generated...'

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
  metadata?: Record<string, unknown>
  createdAt?: string
}

export interface SendTextMessageInput {
  text: string
  model: string
  intent: WorkspaceIntent
  attachments: WorkspaceAttachment[]
  webSearchRequested?: boolean
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
  const requestPhase = ref<WorkspaceRequestPhase>('idle')
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
      requestPhase.value = 'idle'
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
      requestPhase.value = 'failed'
      errorMessage.value = WORKSPACE_HISTORY_FAILED_MESSAGE
    } finally {
      loadingHistory.value = false
      loadingMessages.value = false
    }
  }

  async function selectConversation(id: number) {
    if (!backendEnabled.value) {
      requestPhase.value = 'failed'
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
      requestPhase.value = 'idle'
    } catch {
      activeConversationId.value = null
      messages.value = []
      requestPhase.value = 'failed'
      errorMessage.value = WORKSPACE_MESSAGES_FAILED_MESSAGE
    } finally {
      loadingMessages.value = false
    }
  }

  async function startNewChat() {
    activeConversationId.value = null
    messages.value = []
    errorMessage.value = ''
    requestPhase.value = 'idle'
  }

  async function sendTextMessage(input: SendTextMessageInput) {
    const text = input.text.trim()
    if (!text && input.attachments.length === 0) return false
    if (sending.value) return false

    if (!backendEnabled.value) {
      requestPhase.value = 'failed'
      errorMessage.value = WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE
      return false
    }

    if (!isTextChatIntent(input.intent) || input.attachments.length > 0) {
      requestPhase.value = 'failed'
      errorMessage.value = WORKSPACE_TEXT_ONLY_MESSAGE
      return false
    }

    sending.value = true
    requestPhase.value = 'sending'
    errorMessage.value = ''
    let conversationId = activeConversationId.value
    let localRequestId = ''
    try {
      if (conversationId === null) {
        const conversation = await createConversation({ title: deriveConversationTitle(text) })
        conversationId = conversation.id
        activeConversationId.value = conversation.id
        upsertConversation(conversation)
      }

      localRequestId = createWorkspaceRequestId()
      showLocalRequestProgress(localRequestId, conversationId, text, input.model)
      requestPhase.value = 'generating'

      await appendMessage(conversationId, {
        message_type: CHAT_MESSAGE_TYPE_TEXT,
        role: 'user',
        content: text,
        model: input.model,
        intent: 'chat',
        metadata: input.webSearchRequested ? { web_search_requested: true } : undefined
      })
      let nextMessages: ChatMessage[]
      try {
        nextMessages = await listMessages(conversationId)
      } catch {
        requestPhase.value = 'failed'
        errorMessage.value = WORKSPACE_REFRESH_AFTER_SEND_FAILED_MESSAGE
        if (localRequestId) {
          markLocalRequestFailed(localRequestId, WORKSPACE_REFRESH_AFTER_SEND_FAILED_MESSAGE, 'success')
        }
        return false
      }
      messages.value = nextMessages.map(mapChatMessageToWorkspaceMessage)
      requestPhase.value = 'success'
      await refreshConversationList()
      return true
    } catch (error) {
      const failureMessage = workspaceSendFailureMessage(error)
      requestPhase.value = 'failed'
      errorMessage.value = failureMessage
      if (conversationId !== null) {
        const refreshedMessages = await refreshMessagesAfterFailedSend(conversationId)
        if (refreshedMessages && shouldShowAssistantFailurePlaceholder(error, refreshedMessages)) {
          appendLocalAssistantFailure(
            conversationId,
            localRequestId || createWorkspaceRequestId(),
            failureMessage
          )
        } else if (!refreshedMessages && localRequestId) {
          markLocalRequestFailed(localRequestId, failureMessage)
        }
      } else if (localRequestId) {
        markLocalRequestFailed(localRequestId, failureMessage)
      }
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

  async function refreshMessagesAfterFailedSend(conversationId: number): Promise<WorkspaceMessage[] | null> {
    try {
      const nextMessages = await listMessages(conversationId)
      const workspaceMessages = nextMessages.map(mapChatMessageToWorkspaceMessage)
      messages.value = workspaceMessages
      await refreshConversationList()
      return workspaceMessages
    } catch {
      // Keep the original send error; refresh is only a recovery attempt.
      return null
    }
  }

  function showLocalRequestProgress(
    requestId: string,
    conversationId: number,
    text: string,
    model: string
  ) {
    const createdAt = new Date().toISOString()
    messages.value = [
      ...messages.value,
      {
        id: `local-user-${requestId}`,
        conversationId,
        messageType: CHAT_MESSAGE_TYPE_TEXT,
        role: 'user',
        content: text,
        state: 'sending',
        metadata: {
          local_request_id: requestId,
          request_phase: 'sending',
          model
        },
        createdAt
      },
      {
        id: `local-assistant-${requestId}`,
        conversationId,
        messageType: CHAT_MESSAGE_TYPE_TEXT,
        role: 'assistant',
        content: WORKSPACE_GENERATING_MESSAGE,
        state: 'generating',
        metadata: {
          local_request_id: requestId,
          request_phase: 'generating',
          model
        },
        createdAt
      }
    ]
  }

  function markLocalRequestFailed(
    requestId: string,
    failureMessage: string,
    userState: WorkspaceMessageState = 'failed'
  ) {
    messages.value = messages.value.map((message) => {
      if (message.id === `local-user-${requestId}`) {
        return {
          ...message,
          state: userState,
          metadata: {
            ...message.metadata,
            request_phase: userState
          }
        }
      }
      if (message.id !== `local-assistant-${requestId}`) return message
      return {
        ...message,
        content: failureMessage,
        state: 'failed' as const,
        metadata: {
          ...message.metadata,
          request_phase: 'failed'
        }
      }
    })
  }

  function appendLocalAssistantFailure(
    conversationId: number,
    requestId: string,
    failureMessage: string
  ) {
    messages.value = [
      ...messages.value,
      {
        id: `local-assistant-failed-${requestId}`,
        conversationId,
        messageType: CHAT_MESSAGE_TYPE_TEXT,
        role: 'assistant',
        content: failureMessage,
        state: 'failed',
        metadata: {
          local_request_id: requestId,
          request_phase: 'failed'
        },
        createdAt: new Date().toISOString()
      }
    ]
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
    requestPhase,
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
  const attachments = mapWorkspaceImageAssets(message)
  const state = mapWorkspaceMessageState(message)
  return {
    id: `message-${message.id}`,
    persistedId: message.id,
    conversationId: message.conversation_id,
    messageType: message.message_type,
    role: message.role === 'assistant' ? 'assistant' : 'user',
    content: message.content || mapWorkspaceImageErrorMessage(message),
    state,
    attachments,
    metadata: sanitizeWorkspaceMessageMetadata(message, attachments),
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

function createWorkspaceRequestId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function shouldShowAssistantFailurePlaceholder(error: unknown, messages: WorkspaceMessage[]) {
  if (!isAssistantResponseFailure(error)) return false
  const lastUserIndex = findLastMessageIndex(messages, (message) => message.role === 'user')
  if (lastUserIndex < 0) return false
  return !messages.slice(lastUserIndex + 1).some((message) => message.role === 'assistant')
}

function isAssistantResponseFailure(error: unknown) {
  const code = chatWorkspaceErrorCode(error)
  const message = chatWorkspaceErrorMessage(error).toLowerCase()
  return (
    code === 'WORKSPACE_SERVICE_UNAVAILABLE' ||
    message.includes('workspace service unavailable') ||
    message.includes('provider is not connected') ||
    message.includes('provider unavailable') ||
    message.includes('ai response provider is not connected')
  )
}

function findLastMessageIndex(
  messages: WorkspaceMessage[],
  predicate: (message: WorkspaceMessage) => boolean
) {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    if (predicate(messages[index])) return index
  }
  return -1
}

function workspaceSendFailureMessage(error: unknown) {
  const code = chatWorkspaceErrorCode(error)
  const message = chatWorkspaceErrorMessage(error).toLowerCase()
  if (
    code === 'WORKSPACE_MODEL_UNAVAILABLE' ||
    message.includes('model is not available') ||
    message.includes('model unavailable')
  ) {
    return WORKSPACE_MODEL_UNAVAILABLE_MESSAGE
  }
  if (
    code === 'WORKSPACE_CAPABILITY_UNAVAILABLE' ||
    code === 'WORKSPACE_ATTACHMENTS_DISABLED' ||
    message.includes('capability is not available') ||
    message.includes('capability unavailable') ||
    message.includes('attachments are disabled')
  ) {
    return WORKSPACE_TEXT_ONLY_MESSAGE
  }
  if (
    code === 'WORKSPACE_SERVICE_UNAVAILABLE' ||
    message.includes('workspace service unavailable') ||
    message.includes('provider is not connected') ||
    message.includes('provider unavailable') ||
    message.includes('ai response provider is not connected')
  ) {
    return WORKSPACE_PROVIDER_FAILED_MESSAGE
  }
  return WORKSPACE_SEND_FAILED_MESSAGE
}

function chatWorkspaceErrorCode(error: unknown) {
  if (!error || typeof error !== 'object') return ''
  const workspaceError = error as ChatWorkspaceError
  const code = workspaceError.code ?? workspaceError.error
  return typeof code === 'string' || typeof code === 'number' ? String(code).trim() : ''
}

function chatWorkspaceErrorMessage(error: unknown) {
  if (typeof error === 'string') return error.trim()
  if (error instanceof Error) return error.message.trim()
  if (!error || typeof error !== 'object') return ''
  const workspaceError = error as ChatWorkspaceError
  return typeof workspaceError.message === 'string' || typeof workspaceError.message === 'number'
    ? String(workspaceError.message).trim()
    : ''
}

function mapWorkspaceMessageState(message: ChatMessage): WorkspaceMessageState | undefined {
  const status = metadataString(message.metadata, 'status') || message.status || ''
  if (status === 'pending') return message.role === 'user' ? 'sending' : 'generating'
  if (status === 'failed') return 'failed'
  return undefined
}

function mapWorkspaceImageErrorMessage(message: ChatMessage) {
  if (message.message_type !== CHAT_MESSAGE_TYPE_IMAGE) return ''
  return metadataString(message.metadata, 'error_message') || 'Image generation failed. Please try again.'
}

function mapWorkspaceImageAssets(message: ChatMessage): WorkspaceAttachment[] | undefined {
  if (message.message_type !== CHAT_MESSAGE_TYPE_IMAGE) return undefined
  const rawAssets = Array.isArray(message.metadata?.assets) ? message.metadata.assets : []
  const attachments = rawAssets
    .map((raw, index) => mapWorkspaceImageAsset(raw, index))
    .filter((item): item is WorkspaceAttachment => item !== null)
  return attachments.length > 0 ? attachments : undefined
}

function sanitizeWorkspaceMessageMetadata(
  message: ChatMessage,
  attachments: WorkspaceAttachment[] | undefined
): Record<string, unknown> | undefined {
  if (!message.metadata) return undefined
  if (message.message_type !== CHAT_MESSAGE_TYPE_IMAGE) return message.metadata
  const metadata: Record<string, unknown> = {}
  for (const key of [
    'capability',
    'result_type',
    'status',
    'error_code',
    'error_message',
    'provider',
    'model',
    'latency_ms',
    'usage',
    'enhanced_prompt_present',
    'prompt_present'
  ]) {
    if (message.metadata[key] !== undefined) metadata[key] = message.metadata[key]
  }
  if (attachments?.length) {
    metadata.assets = attachments.map((item) => ({
      id: item.id,
      url: item.url,
      mime_type: item.asset?.content_type,
      provider: item.asset?.storage_provider,
      model: message.model
    }))
  }
  return metadata
}

function mapWorkspaceImageAsset(raw: unknown, index: number): WorkspaceAttachment | null {
  if (!raw || typeof raw !== 'object') return null
  const asset = raw as Record<string, unknown>
  const url = stringValue(asset.url)
  if (!isSafeRemoteImageURL(url)) return null
  const mimeType = stringValue(asset.mime_type) || stringValue(asset.mimeType)
  const name = stringValue(asset.name) || stringValue(asset.id) || `generated-image-${index + 1}`
  return {
    id: stringValue(asset.id) || `generated-image-${index + 1}`,
    name,
    url,
    type: 'image',
    asset: {
      id: numberValue(asset.id) || 0,
      asset_kind: 'image',
      source_type: 'generated',
      asset_role: 'result',
      storage_provider: stringValue(asset.storage_provider),
      storage_key: stringValue(asset.storage_key),
      url,
      preview_url: stringValue(asset.preview_url) || undefined,
      original_name: name,
      content_type: mimeType || 'image/png',
      byte_size: numberValue(asset.byte_size) || 0,
      status: 'completed',
      created_at: '',
      updated_at: ''
    }
  }
}

function isSafeRemoteImageURL(value: string) {
  if (!value) return false
  const lower = value.toLowerCase()
  if (lower.startsWith('data:') || lower.startsWith('blob:') || lower.startsWith('javascript:')) {
    return false
  }
  return lower.startsWith('https://') || lower.startsWith('http://') || lower.startsWith('/')
}

function metadataString(metadata: Record<string, unknown> | undefined, key: string) {
  if (!metadata) return ''
  return stringValue(metadata[key])
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function numberValue(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}
