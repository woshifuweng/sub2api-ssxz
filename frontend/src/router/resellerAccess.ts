import type { ResellerRole } from '@/api/reseller'

export function hasRequiredResellerRole(
  requiredRole: 'agent' | 'agent_manager',
  actualRole: ResellerRole | null,
): boolean {
  if (requiredRole === 'agent_manager') return actualRole === 'agent_manager'
  return actualRole === 'agent' || actualRole === 'agent_manager'
}

export function resellerAccessFallback(actualRole: ResellerRole | null): string {
  return actualRole === 'agent' || actualRole === 'agent_manager'
    ? '/app/reseller'
    : '/app/dashboard'
}
