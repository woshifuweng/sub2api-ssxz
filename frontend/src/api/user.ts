/**
 * User API endpoints
 * Handles user profile management and password changes
 */

import { apiClient } from './client'
import type {
  User,
  UserAvatar,
  ChangePasswordRequest,
  UserAffiliateDetail,
  AffiliateTransferResponse,
  BasePaginationResponse,
  NotifyEmailEntry
} from '@/types'
import type { RedeemHistoryItem } from './redeem'

/**
 * Aggregated balance ledger response:
 * redeem codes + affiliate transfers (type `affiliate_balance`, code `AFF-<id>`).
 */
export interface UserBalanceHistoryResponse extends BasePaginationResponse<RedeemHistoryItem> {
  total_recharged: number
}

/**
 * Get current user profile
 * @returns User profile data
 */
export async function getProfile(): Promise<User> {
  const { data } = await apiClient.get<User>('/user/profile')
  return data
}

/**
 * Update current user profile
 * @param profile - Profile data to update
 * @returns Updated user profile data
 */
export async function updateProfile(profile: {
  username?: string
  balance_notify_enabled?: boolean
  balance_notify_threshold?: number | null
  balance_notify_extra_emails?: NotifyEmailEntry[]
}): Promise<User> {
  const { data } = await apiClient.put<User>('/user', profile)
  return data
}

export async function sendNotifyEmailCode(email: string): Promise<void> {
  await apiClient.post('/user/notify-email/send-code', { email })
}

export async function verifyNotifyEmail(email: string, code: string): Promise<void> {
  await apiClient.post('/user/notify-email/verify', { email, code })
}

export async function removeNotifyEmail(email: string): Promise<void> {
  await apiClient.delete('/user/notify-email', { data: { email } })
}

export async function toggleNotifyEmail(email: string, disabled: boolean): Promise<User> {
  const { data } = await apiClient.put<User>('/user/notify-email/toggle', { email, disabled })
  return data
}

/**
 * Change current user password
 * @param passwords - Old and new password
 * @returns Success message
 */
export async function changePassword(
  oldPassword: string,
  newPassword: string
): Promise<{ message: string }> {
  const payload: ChangePasswordRequest = {
    old_password: oldPassword,
    new_password: newPassword
  }

  const { data } = await apiClient.put<{ message: string }>('/user/password', payload)
  return data
}

export async function getAvatar(): Promise<UserAvatar | null> {
  const { data } = await apiClient.get<UserAvatar | null>('/user/avatar')
  return data
}

export async function updateAvatar(avatar: string): Promise<UserAvatar | null> {
  const { data } = await apiClient.put<UserAvatar | null>('/user/avatar', { avatar })
  return data
}

export async function getAffiliateDetail(): Promise<UserAffiliateDetail> {
  const { data } = await apiClient.get<UserAffiliateDetail>('/user/aff')
  return data
}

export async function transferAffiliateQuota(): Promise<AffiliateTransferResponse> {
  const { data } = await apiClient.post<AffiliateTransferResponse>('/user/aff/transfer')
  return data
}

/**
 * Get current user's aggregated balance ledger (paginated):
 * redeem codes + affiliate transfers.
 */
export async function getBalanceHistory(
  params?: { page?: number; page_size?: number; type?: string }
): Promise<UserBalanceHistoryResponse> {
  const { data } = await apiClient.get<UserBalanceHistoryResponse>('/user/balance-history', { params })
  return data
}

export const userAPI = {
  getProfile,
  updateProfile,
  changePassword,
  sendNotifyEmailCode,
  verifyNotifyEmail,
  removeNotifyEmail,
  toggleNotifyEmail,
  getAvatar,
  updateAvatar,
  getAffiliateDetail,
  transferAffiliateQuota,
  getBalanceHistory
}

export default userAPI
