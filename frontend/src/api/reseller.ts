import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export type ResellerRole = 'agent' | 'agent_manager'
export type WithdrawStatus = 'pending' | 'approved' | 'rejected' | 'cancelled'
export type ManagedAgentRole = 'agent' | null

export interface ResellerRoleResponse {
  role: ResellerRole | '' | null
}

export interface AgentDashboard {
  user_id: number
  aff_code: string
  aff_quota: number
  aff_frozen_quota: number
  aff_history_quota: number
  recruit_count: number
  rebate_rate: number
  pending_withdraw: number
  commission_earned: number
}

export interface RecruitRecord {
  user_id: number
  email: string
  username: string
  joined_at?: string
  total_rebate: number
  is_active: boolean
}

export interface WithdrawRequest {
  id: number
  user_id: number
  user_email?: string
  username?: string
  amount: number
  method: string
  account_info: Record<string, unknown> | null
  status: WithdrawStatus
  requested_at: string
  reviewed_at?: string
  reviewed_by?: number
  note: string
}

export interface ManagerDashboard {
  total_agents: number
  total_recruits: number
  pending_withdrawals: number
}

export interface AgentSummary {
  user_id: number
  email: string
  username: string
  role: ResellerRole
  commission_rate: number
  aff_code: string
  recruit_count: number
  aff_quota: number
  granted_at: string
  granted_by?: number
}

export interface ReviewWithdrawalInput {
  action: 'approve' | 'reject'
  reason?: string
}

export const resellerAPI = {
  async getRole(): Promise<ResellerRoleResponse> {
    const { data } = await apiClient.get<ResellerRoleResponse>('/user/reseller/role')
    return data
  },

  async getAgentDashboard(): Promise<AgentDashboard> {
    const { data } = await apiClient.get<AgentDashboard>('/user/reseller/dashboard')
    return data
  },

  async listRecruits(page = 1, pageSize = 20): Promise<PaginatedResponse<RecruitRecord>> {
    const { data } = await apiClient.get<PaginatedResponse<RecruitRecord>>('/user/reseller/recruits', {
      params: { page, page_size: pageSize }
    })
    return data
  },

  async listMyWithdrawals(page = 1, pageSize = 20): Promise<PaginatedResponse<WithdrawRequest>> {
    const { data } = await apiClient.get<PaginatedResponse<WithdrawRequest>>('/user/reseller/withdrawals', {
      params: { page, page_size: pageSize }
    })
    return data
  },

  async requestBalanceConversion(amount: number): Promise<WithdrawRequest> {
    const { data } = await apiClient.post<WithdrawRequest>('/user/reseller/withdraw', { amount })
    return data
  },

  async cancelWithdrawal(id: number): Promise<{ id: number; status: 'cancelled' }> {
    const { data } = await apiClient.post<{ id: number; status: 'cancelled' }>(
      `/user/reseller/withdrawals/${id}/cancel`
    )
    return data
  },

  async getManagerDashboard(): Promise<ManagerDashboard> {
    const { data } = await apiClient.get<ManagerDashboard>('/user/reseller/manager/dashboard')
    return data
  },

  async listManagedAgents(page = 1, pageSize = 20, search = ''): Promise<PaginatedResponse<AgentSummary>> {
    const { data } = await apiClient.get<PaginatedResponse<AgentSummary>>('/user/reseller/manager/agents', {
      params: { page, page_size: pageSize, search: search || undefined }
    })
    return data
  },

  async grantManagedAgent(userId: number, notes = ''): Promise<{ user_id: number; role: ResellerRole }> {
    const { data } = await apiClient.post<{ user_id: number; role: ResellerRole }>(
      `/user/reseller/manager/agents/${userId}/grant`,
      { notes }
    )
    return data
  },

  async revokeManagedAgent(userId: number): Promise<{ user_id: number }> {
    const { data } = await apiClient.delete<{ user_id: number }>(`/user/reseller/manager/agents/${userId}/role`)
    return data
  },

  async setManagedAgentRole(
    userId: number,
    role: ManagedAgentRole,
    notes = ''
  ): Promise<{ user_id: number; role?: ResellerRole }> {
    if (role === 'agent') {
      return this.grantManagedAgent(userId, notes)
    }
    return this.revokeManagedAgent(userId)
  },

  async listManagedWithdrawals(
    page = 1,
    pageSize = 20,
    status = ''
  ): Promise<PaginatedResponse<WithdrawRequest>> {
    const { data } = await apiClient.get<PaginatedResponse<WithdrawRequest>>(
      '/user/reseller/manager/withdrawals',
      { params: { page, page_size: pageSize, status: status || undefined } }
    )
    return data
  },

  async listAdminAgents(
    page = 1,
    pageSize = 20,
    search = ''
  ): Promise<PaginatedResponse<AgentSummary>> {
    const { data } = await apiClient.get<PaginatedResponse<AgentSummary>>('/admin/reseller/agents', {
      params: { page, page_size: pageSize, search: search || undefined }
    })
    return data
  },

  async listAdminWithdrawals(
    page = 1,
    pageSize = 20,
    status: WithdrawStatus | '' = ''
  ): Promise<PaginatedResponse<WithdrawRequest>> {
    const { data } = await apiClient.get<PaginatedResponse<WithdrawRequest>>(
      '/admin/reseller/withdrawals',
      { params: { page, page_size: pageSize, status: status || undefined } }
    )
    return data
  },

  async reviewWithdrawal(
    id: number,
    input: ReviewWithdrawalInput
  ): Promise<{ id: number; status: 'approved' | 'rejected' }> {
    const { data } = await apiClient.post<{ id: number; status: 'approved' | 'rejected' }>(
      `/admin/reseller/withdrawals/${id}/review`,
      input
    )
    return data
  }
}

export default resellerAPI
