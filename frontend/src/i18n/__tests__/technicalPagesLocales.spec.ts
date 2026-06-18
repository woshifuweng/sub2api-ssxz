import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

function readPath(messages: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object') return undefined
    return (current as Record<string, unknown>)[segment]
  }, messages)
}

describe('technical user page locale copy', () => {
  const requiredKeys = [
    'availableChannels.searchPlaceholder',
    'availableChannels.columns.name',
    'availableChannels.columns.description',
    'availableChannels.columns.platform',
    'availableChannels.columns.groups',
    'availableChannels.columns.supportedModels',
    'availableChannels.exclusive',
    'availableChannels.public',
    'availableChannels.pricing.billingMode',
    'availableChannels.pricing.billingModeToken',
    'availableChannels.pricing.inputPrice',
    'availableChannels.pricing.outputPrice',
    'availableChannels.pricing.unitPerMillion',
    'common.autoRefresh.title',
    'common.autoRefresh.countdown',
    'common.autoRefresh.enable',
    'common.autoRefresh.seconds',
    'channelStatus.windowTab.7d',
    'channelStatus.windowTab.15d',
    'channelStatus.windowTab.30d',
    'channelStatus.empty.title',
    'channelStatus.empty.description',
    'channelStatus.detailColumns.model',
    'channelStatus.detailColumns.latestStatus',
    'channelStatus.closeDetail',
    'monitorCommon.dialogLatency',
    'monitorCommon.endpointPing',
    'monitorCommon.history60pts',
    'monitorCommon.nextUpdateIn',
    'monitorCommon.status.operational',
    'monitorCommon.providers.openai'
  ]

  it.each([
    ['en', en],
    ['zh', zh]
  ] as const)('defines user technical page strings for %s', (_locale, messages) => {
    for (const key of requiredKeys) {
      const value = readPath(messages, key)
      expect(value, key).toEqual(expect.any(String))
      expect(value).not.toBe(key)
      expect(value).not.toContain('availableChannels.')
      expect(value).not.toContain('common.autoRefresh.')
      expect(value).not.toContain('channelStatus.')
      expect(value).not.toContain('monitorCommon.')
    }
  })
})
