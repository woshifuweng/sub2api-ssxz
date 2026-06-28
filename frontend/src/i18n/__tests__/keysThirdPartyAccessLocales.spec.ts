import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('API Key third-party access locale copy', () => {
  it('positions the zh API Key page as third-party client access', () => {
    const copy = [
      zh.keys.title,
      zh.keys.description,
      zh.keys.clientAccessTitle,
      zh.keys.clientAccessDescription,
      zh.keys.workbenchGuide.title,
      zh.keys.workbenchGuide.description,
      zh.keys.workbenchGuide.ccSwitch,
      zh.keys.workbenchGuide.cherryStudio,
      zh.keys.workbenchGuide.chatbox,
      zh.keys.createdKeyReveal.warningDescription,
      zh.keys.createdKeyReveal.connectionDescription,
      zh.keys.createdKeyReveal.modelHint,
      zh.keys.createFirstKey
    ].join(' ')

    expect(copy).toContain('API Key')
    expect(copy).toContain('第三方')
    expect(copy).toContain('CC Switch')
    expect(copy).toContain('Cherry Studio')
    expect(copy).toContain('Chatbox')
    expect(copy).toContain('脱敏值')
    expect(copy).toContain('后台开放配置')
    expect(copy).not.toContain('开发者 API 平台')
  })

  it('positions the en API Key page as third-party client access', () => {
    const copy = [
      en.keys.title,
      en.keys.description,
      en.keys.clientAccessTitle,
      en.keys.clientAccessDescription,
      en.keys.workbenchGuide.title,
      en.keys.workbenchGuide.description,
      en.keys.workbenchGuide.ccSwitch,
      en.keys.workbenchGuide.cherryStudio,
      en.keys.workbenchGuide.chatbox,
      en.keys.createdKeyReveal.warningDescription,
      en.keys.createdKeyReveal.connectionDescription,
      en.keys.createdKeyReveal.modelHint,
      en.keys.createFirstKey
    ].join(' ')

    expect(copy).toContain('API Key')
    expect(copy).toContain('third-party')
    expect(copy).toContain('CC Switch')
    expect(copy).toContain('Cherry Studio')
    expect(copy).toContain('Chatbox')
    expect(copy).toContain('masked')
    expect(copy).toContain('admin model configuration')
    expect(copy).not.toContain('Developer API Platform')
  })
})
