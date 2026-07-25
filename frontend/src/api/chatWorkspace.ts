/**
 * Chat workspace shell API endpoints.
 * These wrappers only record and retrieve workspace shell state.
 */

import { apiClient } from './client'

export const chatWorkspaceBackendEnabled =
  import.meta.env.VITE_CHAT_WORKSPACE_BACKEND_ENABLED === 'true'

function assertChatWorkspaceBackendEnabled(action: string) {
  if (!chatWorkspaceBackendEnabled) {
    throw new Error(`${action} is unavailable until the chat workspace backend is enabled`)
  }
}

export const CHAT_MESSAGE_TYPE_TEXT = 'text'
export const CHAT_MESSAGE_TYPE_IMAGE = 'image'
export const CHAT_MESSAGE_TYPE_ATTACHMENT = 'attachment'
export const CHAT_MESSAGE_TYPE_IMAGE_TASK = 'image_task'
export const CHAT_MESSAGE_TYPE_ERROR_CARD = 'error_card'

export const CHAT_ASSET_KIND_IMAGE = 'image'
export const CHAT_ASSET_SOURCE_USER_UPLOAD = 'user_upload'
export const CHAT_ASSET_ROLE_ATTACHMENT = 'attachment'

export interface ChatConversation {
  id: number
  title: string
  status: string
  last_message_at?: string
  created_at: string
  updated_at: string
}

export interface ChatMessage {
  id: number
  conversation_id: number
  message_type: string
  role: string
  content: string
  model?: string
  intent?: string
  status?: string
  task_id?: number
  asset_id?: number
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface ChatAsset {
  id: number
  conversation_id?: number
  message_id?: number
  task_id?: number
  asset_kind: string
  source_type: string
  asset_role: string
  storage_provider: string
  storage_key: string
  url: string
  preview_url?: string
  original_name: string
  content_type: string
  byte_size: number
  status: string
  created_at: string
  updated_at: string
}

export interface ChatImageTask {
  id: number
  conversation_id: number
  message_id?: number
  reference_asset_id?: number
  result_asset_id?: number
  request_id: string
  task_status: string
  billing_status: string
  actual_cost?: string
  completed_at?: string
  error_code?: string
  error_message?: string
  prompt: string
  enhanced_prompt: string
  model: string
  ratio: string
  purpose: string
  style: string
  created_at: string
  updated_at: string
}

export interface CreateConversationRequest {
  title?: string
}

export interface AppendMessageRequest {
  message_type: string
  role: string
  content: string
  model: string
  intent: string
  task_id?: number
  asset_id?: number
  metadata?: Record<string, unknown>
}

export interface AppendMessageOptions {
  idempotencyKey?: string
}

export interface RegisterAssetRequest {
  conversation_id?: number
  message_id?: number
  task_id?: number
  asset_kind: string
  source_type?: string
  asset_role?: string
  storage_provider?: string
  storage_key?: string
  url?: string
  preview_url?: string
  original_name?: string
  content_type?: string
  byte_size?: number
}

export interface CreateImageTaskRequest {
  conversation_id: number
  reference_asset_id?: number
  idempotency_key?: string
  prompt?: string
  enhanced_prompt?: string
  model?: string
  ratio?: string
  purpose?: string
  style?: string
}

export interface ChatWorkspaceError {
  status?: number
  code?: string | number
  error?: unknown
  message: string
}

export async function listConversations(): Promise<ChatConversation[]> {
  assertChatWorkspaceBackendEnabled('listConversations')
  const { data } = await apiClient.get<ChatConversation[]>('/chat-workspace/conversations')
  return data
}

export async function createConversation(
  payload: CreateConversationRequest
): Promise<ChatConversation> {
  assertChatWorkspaceBackendEnabled('createConversation')
  const { data } = await apiClient.post<ChatConversation>('/chat-workspace/conversations', payload)
  return data
}

export async function getConversation(id: number): Promise<ChatConversation> {
  assertChatWorkspaceBackendEnabled('getConversation')
  const { data } = await apiClient.get<ChatConversation>(`/chat-workspace/conversations/${id}`)
  return data
}

export async function listMessages(conversationId: number): Promise<ChatMessage[]> {
  assertChatWorkspaceBackendEnabled('listMessages')
  const { data } = await apiClient.get<ChatMessage[]>(
    `/chat-workspace/conversations/${conversationId}/messages`
  )
  return data
}

export async function appendMessage(
  conversationId: number,
  payload: AppendMessageRequest,
  options: AppendMessageOptions = {}
): Promise<ChatMessage> {
  assertChatWorkspaceBackendEnabled('appendMessage')
  const config = options.idempotencyKey
    ? {
        headers: {
          'Idempotency-Key': options.idempotencyKey
        }
      }
    : undefined
  const { data } = await apiClient.post<ChatMessage>(
    `/chat-workspace/conversations/${conversationId}/messages`,
    payload,
    config
  )
  return data
}

export async function registerAsset(payload: RegisterAssetRequest): Promise<ChatAsset> {
  assertChatWorkspaceBackendEnabled('registerAsset')
  const { data } = await apiClient.post<ChatAsset>('/chat-workspace/assets/register', payload)
  return data
}

export async function getAsset(id: number): Promise<ChatAsset> {
  assertChatWorkspaceBackendEnabled('getAsset')
  const { data } = await apiClient.get<ChatAsset>(`/chat-workspace/assets/${id}`)
  return data
}

export async function createImageTask(payload: CreateImageTaskRequest): Promise<ChatImageTask> {
  // The current workspace PR is a frontend shell only. Keep this wrapper guarded
  // until a backend task endpoint validates intent, ownership, billing, and model.
  assertChatWorkspaceBackendEnabled('createImageTask')
  const {
    conversation_id,
    reference_asset_id,
    idempotency_key,
    prompt,
    enhanced_prompt,
    model,
    ratio,
    purpose,
    style
  } = payload
  const { data } = await apiClient.post<ChatImageTask>('/chat-workspace/image-tasks', {
    conversation_id,
    reference_asset_id,
    idempotency_key,
    prompt,
    enhanced_prompt,
    model,
    ratio,
    purpose,
    style
  })
  return data
}

export async function getImageTask(id: number): Promise<ChatImageTask> {
  assertChatWorkspaceBackendEnabled('getImageTask')
  const { data } = await apiClient.get<ChatImageTask>(`/chat-workspace/image-tasks/${id}`)
  return data
}

export const chatWorkspaceAPI = {
  listConversations,
  createConversation,
  getConversation,
  listMessages,
  appendMessage,
  registerAsset,
  getAsset,
  createImageTask,
  getImageTask
}

export default chatWorkspaceAPI
