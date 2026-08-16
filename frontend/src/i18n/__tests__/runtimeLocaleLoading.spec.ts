import { describe, expect, it } from 'vitest'

import { i18n, loadLocaleMessages } from '../index'

const accountKeys = [
  'admin.accounts.bulkActions.selectAllResults',
  'admin.accounts.columns.id',
  'admin.accounts.columns.upstreamBillingRate',
  'admin.accounts.columns.createdAt',
  'admin.accounts.openai.compactAuto',
  'admin.accounts.upstreamBilling.notProbed',
] as const

describe.each(['zh', 'en'] as const)('runtime locale loader (%s)', (locale) => {
  it('loads the modular admin account messages used by the production page', async () => {
    await loadLocaleMessages(locale)
    const messages = i18n.global.getLocaleMessage(locale) as Record<string, any>

    for (const key of accountKeys) {
      expect(i18n.global.te(key), `${locale} is missing ${key}`).toBe(true)
      const value = key.split('.').reduce<any>((current, part) => current?.[part], messages)
      expect(value, `${locale} has no message value for ${key}`).toBeTypeOf('string')
      expect(value, `${locale} stores the raw key as the message for ${key}`).not.toBe(key)
    }
  })
})
