export default {
  reseller: {
    eyebrow: '渠道合作',
    pages: {
      dashboard: { title: '我的返利', description: '查看可兑换余额、累计佣金与推广进展' },
      withdrawals: { title: '兑换记录', description: '将返利转入账户余额，并查看处理进度' },
      recruits: { title: '招募用户', description: '查看通过邀请加入的用户及其返利概况' },
      commission: { title: '佣金明细', description: '查看下线消费带来的逐笔佣金记录' },
      invite: { title: '推广工具', description: '分享邀请码和邀请链接，查看招募进展' },
      manager: { title: '管理 Agent', description: '查看直属 Agent 数据并维护角色' }
    },
    nav: {
      dashboard: '返利概览',
      withdrawals: '兑换记录',
      recruits: '招募用户',
      commission: '佣金明细',
      invite: '推广工具',
      manager: '管理 Agent'
    },
    status: {
      pending: '待审核',
      approved: '已批准',
      rejected: '已拒绝',
      cancelled: '已取消'
    },
    admin: {
      agents: { title: 'Reseller 管理', description: '管理 Agent 合作关系、返利策略与生命周期' },
      withdrawals: { title: '兑换审批', description: '审核 Agent 转入账户余额的申请' },
      nav: { agents: 'Agent 列表', withdrawals: '兑换审批' }
    }
  }
}
