import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

describe('affiliate real locale messages', () => {
  it('renders user and native admin affiliate labels in Chinese without raw keys', () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'zh',
      fallbackLocale: 'en',
      messages: { en, zh },
    })

    const expectedLabels = new Map([
      ['nav.affiliate', '邀请返利'],
      ['nav.affiliateInviteRecords', '邀请记录'],
      ['nav.affiliateRebateRecords', '返利记录'],
      ['nav.affiliateTransferRecords', '提取记录'],
      ['affiliate.title', '邀请返利'],
      ['affiliate.stats.invitedUsers', '邀请人数'],
      ['affiliate.stats.availableQuota', '可转返利额度'],
      ['admin.affiliates.records.inviter', '邀请人'],
      ['admin.affiliates.records.invitee', '被邀请人'],
      ['admin.affiliates.records.rebateAmount', '返利金额'],
      ['admin.affiliates.records.transferAmount', '提取金额'],
    ])

    for (const [key, label] of expectedLabels) {
      expect(i18n.global.t(key)).toBe(label)
      expect(i18n.global.t(key)).not.toBe(key)
    }
  })
})
