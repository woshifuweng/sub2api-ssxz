import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export type ResellerRole = 'agent' | 'agent_manager'
export type WithdrawStatus = 'pending' | 'approved' | 'rejected'

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
  amount: number
  method: string
  account_info: Record<string, unknown>
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
  aff_code: string
  recruit_count: number
  aff_quota: number
  granted_at: string
  granted_by?: number
}

export interface WithdrawInput {
  amount: number
  method: 'alipay' | 'wechat' | 'bank' | 'manual'
  account_info: { account: string }
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

  async requestWithdraw(input: WithdrawInput): Promise<WithdrawRequest> {
    const { data } = await apiClient.post<WithdrawRequest>('/user/reseller/withdraw', input)
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
  }
}

export default resellerAPI
