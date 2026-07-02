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
      zh.keys.workbenchGuide.stepCreateTitle,
      zh.keys.workbenchGuide.stepCreateDescription,
      zh.keys.workbenchGuide.stepCopyTitle,
      zh.keys.workbenchGuide.stepCopyDescription,
      zh.keys.workbenchGuide.stepConfigureTitle,
      zh.keys.workbenchGuide.stepConfigureDescription,
      zh.keys.workbenchGuide.ccSwitch,
      zh.keys.workbenchGuide.cherryStudio,
      zh.keys.workbenchGuide.chatbox,
      zh.keys.createdKeyReveal.warningDescription,
      zh.keys.createdKeyReveal.connectionDescription,
      zh.keys.createdKeyReveal.modelHint,
      zh.keys.fullKeyRequiredForImport,
      zh.keys.useKeyModal.fullKeyMissingDescription,
      zh.keys.useKeyModal.thirdParty.apiKeyHint,
      zh.keys.useKeyModal.thirdParty.modelsEndpointLabel,
      zh.keys.useKeyModal.thirdParty.securityNote,
      zh.keys.useKeyModal.thirdParty.ccSwitchFields.baseUrl,
      zh.keys.useKeyModal.thirdParty.ccSwitchFields.apiKey,
      zh.keys.useKeyModal.thirdParty.cherryStudioFields.provider,
      zh.keys.useKeyModal.thirdParty.cherryStudioFields.apiKey,
      zh.keys.useKeyModal.thirdParty.chatboxFields.apiHost,
      zh.keys.useKeyModal.thirdParty.chatboxFields.apiKey,
      zh.keys.createFirstKey
    ].join(' ')

    expect(copy).toContain('API Key')
    expect(copy).toContain('第三方')
    expect(copy).toContain('CC Switch')
    expect(copy).toContain('Cherry Studio')
    expect(copy).toContain('Chatbox')
    expect(copy).toContain('脱敏值')
    expect(copy).toContain('新建')
    expect(copy).toContain('后台开放配置')
    expect(copy).toContain('服务商类型')
    expect(copy).toContain('API Host')
    expect(copy).toContain('Authorization: Bearer')
    expect(copy).toContain('x-api-key')
    expect(copy).toContain('URL 参数')
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
      en.keys.workbenchGuide.stepCreateTitle,
      en.keys.workbenchGuide.stepCreateDescription,
      en.keys.workbenchGuide.stepCopyTitle,
      en.keys.workbenchGuide.stepCopyDescription,
      en.keys.workbenchGuide.stepConfigureTitle,
      en.keys.workbenchGuide.stepConfigureDescription,
      en.keys.workbenchGuide.ccSwitch,
      en.keys.workbenchGuide.cherryStudio,
      en.keys.workbenchGuide.chatbox,
      en.keys.createdKeyReveal.warningDescription,
      en.keys.createdKeyReveal.connectionDescription,
      en.keys.createdKeyReveal.modelHint,
      en.keys.fullKeyRequiredForImport,
      en.keys.useKeyModal.fullKeyMissingDescription,
      en.keys.useKeyModal.thirdParty.apiKeyHint,
      en.keys.useKeyModal.thirdParty.modelsEndpointLabel,
      en.keys.useKeyModal.thirdParty.securityNote,
      en.keys.useKeyModal.thirdParty.ccSwitchFields.baseUrl,
      en.keys.useKeyModal.thirdParty.ccSwitchFields.apiKey,
      en.keys.useKeyModal.thirdParty.cherryStudioFields.provider,
      en.keys.useKeyModal.thirdParty.cherryStudioFields.apiKey,
      en.keys.useKeyModal.thirdParty.chatboxFields.apiHost,
      en.keys.useKeyModal.thirdParty.chatboxFields.apiKey,
      en.keys.createFirstKey
    ].join(' ')

    expect(copy).toContain('API Key')
    expect(copy).toContain('third-party')
    expect(copy).toContain('CC Switch')
    expect(copy).toContain('Cherry Studio')
    expect(copy).toContain('Chatbox')
    expect(copy).toContain('masked')
    expect(copy).toContain('create a new API Key')
    expect(copy).toContain('admin model configuration')
    expect(copy).toContain('Model list check')
    expect(copy).toContain('Provider type')
    expect(copy).toContain('API Host')
    expect(copy).toContain('Authorization: Bearer')
    expect(copy).toContain('x-api-key')
    expect(copy).toContain('query keys')
    expect(copy).not.toContain('Developer API Platform')
  })

  it('explains API Key quota resets without implying billing or balance changes', () => {
    const zhCopy = [
      zh.keys.quotaAmountHint,
      zh.keys.resetQuotaConfirmMessage,
      zh.keys.rateLimitHint,
      zh.keys.resetRateLimitConfirmMessage
    ].join(' ')

    expect(zhCopy).toContain('自限额')
    expect(zhCopy).toContain('不会删除历史用量')
    expect(zhCopy).toContain('不会改账单')
    expect(zhCopy).toContain('不会返还余额')

    const enCopy = [
      en.keys.quotaAmountHint,
      en.keys.resetQuotaConfirmMessage,
      en.keys.rateLimitHint,
      en.keys.resetRateLimitConfirmMessage
    ].join(' ')

    expect(enCopy).toContain('self-limit')
    expect(enCopy).toContain('does not delete usage history')
    expect(enCopy).toContain('change bills')
    expect(enCopy).toContain('refund balance')
  })

  it('does not ask users to paste full API keys into list search', () => {
    expect(zh.keys.searchPlaceholder).toBe('搜索名称...')
    expect(zh.keys.searchPlaceholder).not.toContain('Key')
    expect(en.keys.searchPlaceholder).toBe('Search name...')
    expect(en.keys.searchPlaceholder.toLowerCase()).not.toContain('key')
  })
})
