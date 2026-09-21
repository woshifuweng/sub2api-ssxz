/**
 * Redeem code API endpoints
 * Handles redeem code redemption for users
 */

import { apiClient } from './client'
import type { PaginatedResponse, RedeemCodeRequest } from '@/types'

export interface RedeemHistoryItem {
  id: number
  code: string
  type: string
  value: number
  status: string
  used_by?: number | null
  used_at: string | null
  created_at: string
  // Notes from admin for admin_balance/admin_concurrency types
  notes?: string
  message?: string
  // Subscription-specific fields
  group_id?: number | null
  validity_days?: number
  group?: {
    id: number
    name: string
  }
}

export type RedeemResult = RedeemHistoryItem

/**
 * Redeem a code
 * @param code - Redeem code string
 * @returns Redeemed code record returned by the backend
 */
export async function redeem(code: string, turnstileToken?: string): Promise<RedeemResult> {
  const payload: RedeemCodeRequest = { code }
  if (turnstileToken) payload.turnstile_token = turnstileToken

  const { data } = await apiClient.post<RedeemResult>('/redeem', payload)

  return data
}

/**
 * Get user's redemption history
 * @returns The requested page of redeemed codes and the total count
 */
export async function getHistory(page = 1, pageSize = 20): Promise<PaginatedResponse<RedeemHistoryItem>> {
  const { data } = await apiClient.get<PaginatedResponse<RedeemHistoryItem>>('/redeem/history', {
    params: { page, page_size: pageSize }
  })
  return data
}

export const redeemAPI = {
  redeem,
  getHistory
}

export default redeemAPI
