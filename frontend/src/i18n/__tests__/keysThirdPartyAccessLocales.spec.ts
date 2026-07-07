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
      zh.keys.clientReadinessHint,
      zh.keys.clientTroubleshootingHint,
      zh.keys.workbenchGuide.title,
      zh.keys.workbenchGuide.description,
      zh.keys.workbenchGuide.stepCreateTitle,
      zh.keys.workbenchGuide.stepCreateDescription,
      zh.keys.workbenchGuide.stepCopyTitle,
      zh.keys.workbenchGuide.stepCopyDescription,
      zh.keys.workbenchGuide.stepConfigureTitle,
      zh.keys.workbenchGuide.stepConfigureDescription,
      zh.keys.workbenchGuide.ccsImportNote,
      zh.keys.workbenchGuide.ccSwitch,
      zh.keys.workbenchGuide.cherryStudio,
      zh.keys.workbenchGuide.chatbox,
      zh.keys.createdKeyReveal.warningDescription,
      zh.keys.createdKeyReveal.connectionDescription,
      zh.keys.createdKeyReveal.modelHint,
      zh.keys.ccsImportNeedsNewKey,
      zh.keys.fullKeyRequiredForImport,
      zh.keys.useKeyModal.fullKeyMissingDescription,
      zh.keys.useKeyModal.openai.configTomlHint,
      zh.keys.useKeyModal.statusWarning.inactiveDescription,
      zh.keys.useKeyModal.statusWarning.expiredDescription,
      zh.keys.useKeyModal.statusWarning.quotaExhaustedDescription,
      zh.keys.useKeyModal.thirdParty.apiKeyHint,
      zh.keys.useKeyModal.thirdParty.modelsEndpointLabel,
      zh.keys.useKeyModal.thirdParty.connectionChecklistBaseUrl,
      zh.keys.useKeyModal.thirdParty.ccSwitchHomepageLabel,
      zh.keys.useKeyModal.thirdParty.ccSwitchRequestUrlLabel,
      zh.keys.useKeyModal.thirdParty.ccSwitchQuickOfficialSite,
      zh.keys.useKeyModal.thirdParty.ccSwitchQuickRequestUrl,
      zh.keys.useKeyModal.thirdParty.connectionChecklistFullKey,
      zh.keys.useKeyModal.thirdParty.connectionChecklistModels,
      zh.keys.useKeyModal.thirdParty.connectionChecklistBalance,
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
    expect(copy).toContain('新建后可导入')
    expect(copy).toContain('新建')
    expect(copy).toContain('一键导入')
    expect(copy).toContain('可用模型以当前 Key 所属分组')
    expect(copy).toContain('状态')
    expect(copy).toContain('账户余额')
    expect(copy).toContain('额度')
    expect(copy).toContain('其它服务商地址')
    expect(copy).toContain('官网链接')
    expect(copy).toContain('API 请求地址')
    expect(copy).toContain('不带 /v1')
    expect(copy).toContain('必须带 /v1')
    expect(copy).toContain('config.toml')
    expect(copy).toContain('base_url')
    expect(copy).toContain('Codex / CC Switch')
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
      en.keys.clientReadinessHint,
      en.keys.clientTroubleshootingHint,
      en.keys.workbenchGuide.title,
      en.keys.workbenchGuide.description,
      en.keys.workbenchGuide.stepCreateTitle,
      en.keys.workbenchGuide.stepCreateDescription,
      en.keys.workbenchGuide.stepCopyTitle,
      en.keys.workbenchGuide.stepCopyDescription,
      en.keys.workbenchGuide.stepConfigureTitle,
      en.keys.workbenchGuide.stepConfigureDescription,
      en.keys.workbenchGuide.ccsImportNote,
      en.keys.workbenchGuide.ccSwitch,
      en.keys.workbenchGuide.cherryStudio,
      en.keys.workbenchGuide.chatbox,
      en.keys.createdKeyReveal.warningDescription,
      en.keys.createdKeyReveal.connectionDescription,
      en.keys.createdKeyReveal.modelHint,
      en.keys.ccsImportNeedsNewKey,
      en.keys.fullKeyRequiredForImport,
      en.keys.useKeyModal.fullKeyMissingDescription,
      en.keys.useKeyModal.openai.configTomlHint,
      en.keys.useKeyModal.statusWarning.inactiveDescription,
      en.keys.useKeyModal.statusWarning.expiredDescription,
      en.keys.useKeyModal.statusWarning.quotaExhaustedDescription,
      en.keys.useKeyModal.thirdParty.apiKeyHint,
      en.keys.useKeyModal.thirdParty.modelsEndpointLabel,
      en.keys.useKeyModal.thirdParty.connectionChecklistBaseUrl,
      en.keys.useKeyModal.thirdParty.ccSwitchHomepageLabel,
      en.keys.useKeyModal.thirdParty.ccSwitchRequestUrlLabel,
      en.keys.useKeyModal.thirdParty.ccSwitchQuickOfficialSite,
      en.keys.useKeyModal.thirdParty.ccSwitchQuickRequestUrl,
      en.keys.useKeyModal.thirdParty.connectionChecklistFullKey,
      en.keys.useKeyModal.thirdParty.connectionChecklistModels,
      en.keys.useKeyModal.thirdParty.connectionChecklistBalance,
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
    expect(copy).toContain('Import after creating')
    expect(copy).toContain('create a new API Key')
    expect(copy).toContain('One-click CC Switch import')
    expect(copy).toContain('admin model configuration')
    expect(copy).toContain('status')
    expect(copy).toContain('account balance')
    expect(copy).toContain('quota')
    expect(copy).toContain('another service address')
    expect(copy).toContain('Homepage')
    expect(copy).toContain('API request URL')
    expect(copy).toContain('without /v1')
    expect(copy).toContain('including /v1')
    expect(copy).toContain('custom provider')
    expect(copy).toContain('base_url')
    expect(copy).toContain('Codex / CC Switch')
    expect(copy).toContain('Model list check')
    expect(copy).toContain('Provider type')
    expect(copy).toContain('API Host')
    expect(copy).toContain('Authorization: Bearer')
    expect(copy).toContain('x-api-key')
    expect(copy).toContain('query keys')
    expect(copy).not.toContain('Developer API Platform')
  })

  it('keeps customer test setup concise and model availability scoped to key configuration', () => {
    const zhCopy = [
      zh.keys.clientAccessDescription,
      zh.keys.clientReadinessHint,
      zh.keys.clientTroubleshootingHint,
      zh.keys.workbenchGuide.stepConfigureDescription,
      zh.keys.workbenchGuide.ccSwitch,
      zh.keys.createdKeyReveal.modelHint,
      zh.keys.createdKeyReveal.readinessHint,
      zh.keys.useKeyModal.thirdParty.quickStartModelHint,
      zh.keys.useKeyModal.thirdParty.quickStartSpeedHint,
      zh.keys.useKeyModal.thirdParty.ccSwitchQuickDescription,
      zh.keys.useKeyModal.thirdParty.ccSwitchQuickModel,
      zh.keys.useKeyModal.thirdParty.connectionChecklistModels,
      zh.keys.useKeyModal.thirdParty.connectionChecklistBalance,
      zh.keys.useKeyModal.thirdParty.troubleshooting401,
      zh.keys.useKeyModal.thirdParty.troubleshooting403,
      zh.keys.useKeyModal.thirdParty.troubleshooting503
    ].join(' ')

    expect(zhCopy).toContain('Base URL')
    expect(zhCopy).toContain('完整 API Key')
    expect(zhCopy).toContain('gpt-5.5')
    expect(zhCopy).toContain('gpt-5.4-mini')
    expect(zhCopy).toContain('当前 Key 所属分组')
    expect(zhCopy).toContain('后台配置')
    expect(zhCopy).toContain('401')
    expect(zhCopy).toContain('403')
    expect(zhCopy).toContain('503')
    expect(zhCopy).not.toContain('Gemini/Claude/OpenAI 全部稳定')

    const enCopy = [
      en.keys.clientAccessDescription,
      en.keys.clientReadinessHint,
      en.keys.clientTroubleshootingHint,
      en.keys.workbenchGuide.stepConfigureDescription,
      en.keys.workbenchGuide.ccSwitch,
      en.keys.createdKeyReveal.modelHint,
      en.keys.createdKeyReveal.readinessHint,
      en.keys.useKeyModal.thirdParty.quickStartModelHint,
      en.keys.useKeyModal.thirdParty.quickStartSpeedHint,
      en.keys.useKeyModal.thirdParty.ccSwitchQuickDescription,
      en.keys.useKeyModal.thirdParty.ccSwitchQuickModel,
      en.keys.useKeyModal.thirdParty.connectionChecklistModels,
      en.keys.useKeyModal.thirdParty.connectionChecklistBalance,
      en.keys.useKeyModal.thirdParty.troubleshooting401,
      en.keys.useKeyModal.thirdParty.troubleshooting403,
      en.keys.useKeyModal.thirdParty.troubleshooting503
    ].join(' ')

    expect(enCopy).toContain('Base URL')
    expect(enCopy).toContain('full API Key')
    expect(enCopy).toContain('gpt-5.5')
    expect(enCopy).toContain('gpt-5.4-mini')
    expect(enCopy).toContain('key group')
    expect(enCopy).toContain('admin model configuration')
    expect(enCopy).toContain('401')
    expect(enCopy).toContain('403')
    expect(enCopy).toContain('503')
    expect(enCopy).not.toContain('Gemini/Claude/OpenAI are all stable')
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
