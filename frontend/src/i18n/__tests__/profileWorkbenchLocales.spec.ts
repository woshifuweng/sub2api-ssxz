import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('profile workbench locale copy', () => {
  it('keeps zh profile copy focused on ordinary-user account settings', () => {
    const copy = [
      zh.profile.workbench.introDescription,
      zh.profile.workbench.accountInfoTitle,
      zh.profile.workbench.securityTitle,
      zh.profile.totp.statusLoadFailed,
      zh.profile.totp.statusLoadFailedHint
    ].join(' ')

    expect(copy).toContain('个人资料')
    expect(copy).toContain('API Key')
    expect(copy).toContain('普通用户')
    expect(copy).toContain('暂时无法加载')
    expect(copy).toContain('不会把状态未知显示成未启用')
  })

  it('keeps en profile copy focused on ordinary-user account settings', () => {
    const copy = [
      en.profile.workbench.introDescription,
      en.profile.workbench.accountInfoTitle,
      en.profile.workbench.securityTitle,
      en.profile.totp.statusLoadFailed,
      en.profile.totp.statusLoadFailedHint
    ].join(' ')

    expect(copy).toContain('profile')
    expect(copy).toContain('API Key')
    expect(copy).toContain('ordinary users')
    expect(copy).toContain('temporarily unavailable')
    expect(copy).toContain('unknown status is not shown as not enabled')
  })
})
