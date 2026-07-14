/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey, GroupPlatform } from '@/types'

export type AdminAPIKeyStatus = 'active' | 'inactive' | 'expired'
export type AdminAPIKeySort =
  | 'created_at'
  | 'last_used_at'
  | 'today_actual_cost'
  | 'last_30_days_actual_cost'
  | 'total_actual_cost'

export interface AdminAPIKeyListUser {
  id: number
  email: string
  username: string
  balance: number
}

export interface AdminAPIKeyListGroup {
  id: number
  name: string
  platform: GroupPlatform
  rate_multiplier: number
}

export interface AdminAPIKeyListItem {
  id: number
  user: AdminAPIKeyListUser
  key: string
  name: string
  group?: AdminAPIKeyListGroup | null
  status: AdminAPIKeyStatus
  quota: number
  quota_used: number
  last_used_at?: string | null
  expires_at?: string | null
  created_at: string
  today_actual_cost: number
  last_30_days_actual_cost: number
  total_actual_cost: number
}

export interface AdminAPIKeyListSummary {
  total: number
  active: number
  inactive: number
  expired: number
  last_30_days_actual_cost: number
}

export interface AdminAPIKeyListResponse {
  items: AdminAPIKeyListItem[]
  total: number
  page: number
  page_size: number
  pages: number
  summary: AdminAPIKeyListSummary
}

export interface AdminAPIKeyListParams {
  page?: number
  page_size?: number
  search?: string
  user_id?: number
  group_id?: number
  status?: AdminAPIKeyStatus
  sort_by?: AdminAPIKeySort
  sort_order?: 'asc' | 'desc'
}

export async function list(
  params: AdminAPIKeyListParams,
  options?: { signal?: AbortSignal }
): Promise<AdminAPIKeyListResponse> {
  const { data } = await apiClient.get<AdminAPIKeyListResponse>('/admin/api-keys', {
    params,
    signal: options?.signal
  })
  return data
}

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_id: groupId === null ? 0 : groupId
  })
  return data
}

export async function setEnabled(id: number, enabled: boolean): Promise<{ api_key: ApiKey }> {
  const { data } = await apiClient.patch<{ api_key: ApiKey }>(`/admin/api-keys/${id}/status`, {
    enabled
  })
  return data
}

export async function deleteApiKey(id: number): Promise<{ deleted: boolean }> {
  const { data } = await apiClient.delete<{ deleted: boolean }>(`/admin/api-keys/${id}`)
  return data
}

export const apiKeysAPI = {
  list,
  updateApiKeyGroup,
  setEnabled,
  deleteApiKey
}

export default apiKeysAPI
