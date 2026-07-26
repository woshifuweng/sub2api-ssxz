/**
 * Redeem code API endpoints
 * Handles redeem code redemption for users
 */

import { apiClient } from './client'
import type { BasePaginationResponse, RedeemCodeRequest } from '@/types'

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
 * Get user's redemption history (paginated)
 * @returns Paginated redeemed code records
 */
export async function getHistory(
  params?: { page?: number; page_size?: number }
): Promise<BasePaginationResponse<RedeemHistoryItem>> {
  const { data } = await apiClient.get<BasePaginationResponse<RedeemHistoryItem>>('/redeem/history', { params })
  return data
}

export const redeemAPI = {
  redeem,
  getHistory
}

export default redeemAPI
