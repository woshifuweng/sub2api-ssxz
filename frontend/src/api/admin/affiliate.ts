/**
 * Admin affiliate API endpoints.
 * Exposes the existing Sub2API affiliate management backend for operators.
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface AdminAffiliateEntry {
  user_id: number
  email: string
  username: string
  aff_code: string
  aff_code_custom: boolean
  aff_rebate_rate_percent?: number | null
  aff_count: number
}

export interface AffiliateUserSummary {
  id: number
  email: string
  username: string
}

export interface UpdateAffiliateUserRequest {
  aff_code?: string
  aff_rebate_rate_percent?: number | null
  clear_rebate_rate?: boolean
}

export async function listUsers(
  page: number = 1,
  pageSize: number = 20,
  search: string = ''
): Promise<PaginatedResponse<AdminAffiliateEntry>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminAffiliateEntry>>(
    '/admin/affiliates/users',
    {
      params: {
        page,
        page_size: pageSize,
        search: search || undefined
      }
    }
  )
  return data
}

export async function lookupUsers(q: string): Promise<AffiliateUserSummary[]> {
  const keyword = q.trim()
  if (!keyword) {
    return []
  }
  const { data } = await apiClient.get<AffiliateUserSummary[]>('/admin/affiliates/users/lookup', {
    params: { q: keyword }
  })
  return data
}

export async function updateUserSettings(
  userId: number,
  payload: UpdateAffiliateUserRequest
): Promise<{ user_id: number }> {
  const { data } = await apiClient.put<{ user_id: number }>(
    `/admin/affiliates/users/${userId}`,
    payload
  )
  return data
}

export async function clearUserSettings(userId: number): Promise<{ user_id: number }> {
  const { data } = await apiClient.delete<{ user_id: number }>(`/admin/affiliates/users/${userId}`)
  return data
}

export async function batchSetRate(
  userIds: number[],
  ratePercent: number | null,
  clear: boolean = false
): Promise<{ affected: number }> {
  const { data } = await apiClient.post<{ affected: number }>(
    '/admin/affiliates/users/batch-rate',
    {
      user_ids: userIds,
      aff_rebate_rate_percent: clear ? undefined : ratePercent,
      clear
    }
  )
  return data
}

export const affiliateAPI = {
  listUsers,
  lookupUsers,
  updateUserSettings,
  clearUserSettings,
  batchSetRate
}

export default affiliateAPI
