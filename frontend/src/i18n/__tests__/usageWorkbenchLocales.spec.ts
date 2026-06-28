import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('usage workbench locale copy', () => {
  it('explains real usage, no-charge, and frontend pricing limits in zh', () => {
    const copy = [
      zh.usage.workbench.subtitle,
      zh.usage.workbench.balanceDescription,
      zh.usage.workbench.billingExplanationDescription,
      zh.usage.workbench.billingExplanationItems.failureNoCharge,
      zh.usage.workbench.statsLoadError,
      zh.usage.workbench.trendLoadErrorHint,
      zh.usage.workbench.detailsLoadErrorHint
    ].join(' ')

    expect(copy).toContain('真实用量')
    expect(copy).toContain('第三方')
    expect(copy).toContain('前端不决定价格')
    expect(copy).toContain('未扣费')
    expect(copy).toContain('暂时无法加载')
    expect(copy).toContain('不会把接口失败显示成空趋势')
    expect(copy).toContain('不会自动跳到旧版页面')
  })

  it('explains real usage, no-charge, and frontend pricing limits in en', () => {
    const copy = [
      en.usage.workbench.subtitle,
      en.usage.workbench.balanceDescription,
      en.usage.workbench.billingExplanationDescription,
      en.usage.workbench.billingExplanationItems.failureNoCharge,
      en.usage.workbench.statsLoadError,
      en.usage.workbench.trendLoadErrorHint,
      en.usage.workbench.detailsLoadErrorHint
    ].join(' ')

    expect(copy).toContain('real usage')
    expect(copy).toContain('third-party')
    expect(copy).toContain('frontend does not decide prices')
    expect(copy).toContain('no charge')
    expect(copy).toContain('temporarily unavailable')
    expect(copy).toContain('API failures are not presented as an empty trend')
    expect(copy).toContain('legacy usage page')
  })
})
