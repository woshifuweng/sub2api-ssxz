import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export type ResellerRole = 'agent' | 'agent_manager'
export type ResellerStatus = 'active' | 'disabled' | 'revoked'
export type RebateMode = 'global' | 'disabled' | 'custom'
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
  status: string
  reseller_role: string
  commission_rate: number
  created_at?: string
  joined_at?: string
  total_rebate: number
  is_active: boolean
}

export interface RecruitUsageLog {
  id: number
  created_at: string
  model: string
  request_type: number
  total_tokens: number
  actual_cost: number
}

export interface RecruitRecharge {
  id: number
  event_type: string
  amount: number
  note: string
  created_at: string
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

export interface CommissionRecord {
  time: string
  source_user_masked_email: string
  source_consumption_usd: number
  commission_usd: number
  commission_rate: number
}

export interface CommissionResponse {
  items: CommissionRecord[]
  total: number
  total_commission_usd: number
  page: number
  page_size: number
}

export interface InviteResponse {
  invite_code: string
  invite_link: string
  total_recruited: number
  recruited_this_month: number
}

export interface AgentSummary {
  user_id: number
  email: string
  username: string
  role: ResellerRole
  status: ResellerStatus
  manager_id: number | null
  manager_email: string | null
  effective_rebate_rate_percent: number | null
  rebate_mode: RebateMode
  aff_code: string
  recruit_count: number
  commission_balance: string
  commission_total: string
  notes: string
  granted_at: string
  updated_at: string
  disabled_at: string | null
  disabled_by_email: string | null
  disabled_reason: string
  revoked_at: string | null
  granted_by?: number
}

export interface AgentDetail extends AgentSummary {
  aff_history_quota: number
  pending_redemption_count: number
  recruits?: RecruitRecord[]
}

export interface AdminRecruitRecord {
  user_id: number
  email: string
  username: string
  status: string
  reseller_role: string
  created_at?: string
  joined_at?: string
  is_active: boolean
  total_recharge_usd: number
  total_consumption_usd: number
  current_balance_usd: number
  commission_contributed_usd: number
}

export interface AdminAgentFilters {
  search?: string
  status?: ResellerStatus | 'all' | ''
  role?: ResellerRole | ''
  manager_id?: number
}

export interface AdminWithdrawalsOptions {
  status?: WithdrawStatus | ''
  userId?: number
  page?: number
  pageSize?: number
}

export interface RebatePolicyInput {
  mode: RebateMode
  rate_percent?: number
}

export interface UpdateAdminAgentInput {
  role?: ResellerRole
  manager_id?: number | null
  notes?: string
  rebate_policy?: RebatePolicyInput
  reason?: string
}

export interface ReviewWithdrawalInput {
  action: 'approve' | 'reject'
  reason?: string
}

export interface GrantAdminResellerRoleInput {
  role: ResellerRole
  notes?: string
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

  async getRecruitDetail(userId: number): Promise<RecruitRecord> {
    const { data } = await apiClient.get<RecruitRecord>(`/user/reseller/recruits/${userId}`)
    return data
  },

  async listRecruitLogs(userId: number, page = 1, pageSize = 20): Promise<PaginatedResponse<RecruitUsageLog>> {
    const { data } = await apiClient.get<PaginatedResponse<RecruitUsageLog>>(
      `/user/reseller/recruits/${userId}/logs`,
      { params: { page, page_size: pageSize } }
    )
    return data
  },

  async listRecruitRecharges(userId: number, page = 1, pageSize = 20): Promise<PaginatedResponse<RecruitRecharge>> {
    const { data } = await apiClient.get<PaginatedResponse<RecruitRecharge>>(
      `/user/reseller/recruits/${userId}/recharges`,
      { params: { page, page_size: pageSize } }
    )
    return data
  },

  async listCommission(
    page = 1,
    pageSize = 50,
    startDate = '',
    endDate = ''
  ): Promise<CommissionResponse> {
    const { data } = await apiClient.get<CommissionResponse>('/user/reseller/commission', {
      params: {
        page,
        page_size: pageSize,
        start_date: startDate || undefined,
        end_date: endDate || undefined
      }
    })
    return data
  },

  async getInvite(): Promise<InviteResponse> {
    const { data } = await apiClient.get<InviteResponse>('/user/reseller/invite')
    return data
  },

  async listMyWithdrawals(page = 1, pageSize = 20): Promise<PaginatedResponse<WithdrawRequest>> {
    const { data } = await apiClient.get<PaginatedResponse<WithdrawRequest>>('/user/reseller/withdrawals', {
      params: { page, page_size: pageSize }
    })
    return data
  },

  async requestBalanceConversion(
    amount: number,
    options?: { idempotencyKey?: string }
  ): Promise<WithdrawRequest> {
    const idempotencyKey = options?.idempotencyKey?.trim()
    const { data } = await apiClient.post<WithdrawRequest>(
      '/user/reseller/withdrawals',
      { amount },
      idempotencyKey ? { headers: { 'Idempotency-Key': idempotencyKey } } : undefined
    )
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
    filters: AdminAgentFilters | string = {}
  ): Promise<PaginatedResponse<AgentSummary>> {
    const normalizedFilters = typeof filters === 'string' ? { search: filters } : filters
    const { data } = await apiClient.get<PaginatedResponse<AgentSummary>>('/admin/reseller/agents', {
      params: {
        page,
        page_size: pageSize,
        search: normalizedFilters.search || undefined,
        status: normalizedFilters.status || undefined,
        role: normalizedFilters.role || undefined,
        manager_id: normalizedFilters.manager_id || undefined
      }
    })
    return data
  },

  async getAdminAgent(userId: number): Promise<AgentDetail> {
    const { data } = await apiClient.get<AgentDetail>(`/admin/reseller/agents/${userId}`)
    return data
  },

  async listAdminAgentRecruits(
    agentId: number,
    page = 1,
    pageSize = 20
  ): Promise<PaginatedResponse<AdminRecruitRecord>> {
    const { data } = await apiClient.get<PaginatedResponse<AdminRecruitRecord>>(
      `/admin/reseller/agents/${agentId}/recruits`,
      { params: { page, page_size: pageSize } }
    )
    return data
  },

  async updateAdminAgent(userId: number, input: UpdateAdminAgentInput): Promise<AgentDetail> {
    const { data } = await apiClient.patch<AgentDetail>(
      `/admin/reseller/agents/${userId}`,
      input
    )
    return data
  },

  async disableAdminAgent(userId: number, reason: string): Promise<AgentDetail> {
    const { data } = await apiClient.post<AgentDetail>(
      `/admin/reseller/agents/${userId}/disable`,
      { reason }
    )
    return data
  },

  async enableAdminAgent(userId: number): Promise<AgentDetail> {
    const { data } = await apiClient.post<AgentDetail>(
      `/admin/reseller/agents/${userId}/enable`
    )
    return data
  },

  async grantAdminRole(
    userId: number,
    input: GrantAdminResellerRoleInput
  ): Promise<{ user_id: number; role: ResellerRole }> {
    const { data } = await apiClient.post<{ user_id: number; role: ResellerRole }>(
      `/admin/reseller/agents/${userId}/role`,
      input
    )
    return data
  },

  async revokeAdminRole(userId: number): Promise<{ user_id: number }> {
    const { data } = await apiClient.delete<{ user_id: number }>(
      `/admin/reseller/agents/${userId}/role`
    )
    return data
  },

  async listAdminWithdrawals(
    optionsOrPage: AdminWithdrawalsOptions | number = {},
    legacyPageSize = 20,
    legacyStatus: WithdrawStatus | '' = ''
  ): Promise<PaginatedResponse<WithdrawRequest>> {
    const options: AdminWithdrawalsOptions =
      typeof optionsOrPage === 'number'
        ? { page: optionsOrPage, pageSize: legacyPageSize, status: legacyStatus }
        : optionsOrPage
    const page = options.page ?? 1
    const pageSize = options.pageSize ?? 20
    const { data } = await apiClient.get<PaginatedResponse<WithdrawRequest>>(
      '/admin/reseller/withdrawals',
      {
        params: {
          page,
          page_size: pageSize,
          status: options.status || undefined,
          user_id: options.userId || undefined
        }
      }
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
