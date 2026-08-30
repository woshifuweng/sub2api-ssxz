export default {
  reseller: {
    eyebrow: 'Reseller',
    pages: {
      dashboard: { title: 'My rebates', description: 'Review available balance, commission, and referral progress' },
      withdrawals: { title: 'Transfers', description: 'Move rebate credit into your account balance and track reviews' },
      recruits: { title: 'Recruited users', description: 'Review users connected through your invitations' },
      commission: { title: 'Commission', description: 'Review commission earned from recruited-user usage' },
      invite: { title: 'Promotion tools', description: 'Share your invite code and referral link' },
      manager: { title: 'Manage agents', description: 'Review direct agents and maintain their roles' }
    },
    nav: {
      dashboard: 'Overview',
      withdrawals: 'Transfers',
      recruits: 'Recruits',
      commission: 'Commission',
      invite: 'Promotion tools',
      manager: 'Manage agents'
    },
    status: {
      pending: 'Pending',
      approved: 'Approved',
      rejected: 'Rejected',
      cancelled: 'Cancelled'
    },
    admin: {
      agents: { title: 'Reseller management', description: 'Manage agent relationships, rebate policies, and lifecycle' },
      withdrawals: { title: 'Transfer reviews', description: 'Review agent requests to move credit into account balance' },
      nav: { agents: 'Agents', withdrawals: 'Transfer reviews' }
    }
  }
}
