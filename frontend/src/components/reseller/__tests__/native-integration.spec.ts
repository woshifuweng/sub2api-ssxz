import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const resellerRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const files = [
  'components/reseller/RecruitDetailDrawer.vue',
  'components/reseller/WithdrawalStatusBadge.vue',
  'views/reseller/AgentDashboard.vue',
  'views/reseller/AgentWithdrawals.vue',
  'views/reseller/AgentRecruits.vue',
  'views/reseller/AgentCommission.vue',
  'views/reseller/AgentInvite.vue',
  'views/reseller/ManagerDashboard.vue',
  'views/admin/reseller/AdminAgents.vue',
  'views/admin/reseller/AdminWithdrawals.vue',
]

describe('reseller native integration', () => {
  it('does not depend on the retired SSXZ shell or theme variables', () => {
    for (const file of files) {
      const source = readFileSync(resolve(resellerRoot, file), 'utf8')
      expect(source, file).not.toContain('AppSectionShell')
      expect(source, file).not.toContain('--ssxz-')
    }
  })

  it('uses the v183 application layout for user and admin surfaces', () => {
    const userLayout = readFileSync(resolve(resellerRoot, 'components/reseller/ResellerPageLayout.vue'), 'utf8')
    const adminAgents = readFileSync(resolve(resellerRoot, 'views/admin/reseller/AdminAgents.vue'), 'utf8')

    expect(userLayout).toContain('AppLayout')
    expect(adminAgents).toContain('<AppLayout>')
  })
})
