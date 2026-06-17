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
      zh.keys.createFirstKey
    ].join(' ')

    expect(copy).toContain('API Key')
    expect(copy).toContain('第三方')
    expect(copy).toContain('CC Switch')
    expect(copy).toContain('Cherry Studio')
    expect(copy).toContain('Chatbox')
    expect(copy).not.toContain('开发者 API 平台')
  })

  it('positions the en API Key page as third-party client access', () => {
    const copy = [
      en.keys.title,
      en.keys.description,
      en.keys.clientAccessTitle,
      en.keys.clientAccessDescription,
      en.keys.createFirstKey
    ].join(' ')

    expect(copy).toContain('API Key')
    expect(copy).toContain('third-party')
    expect(copy).toContain('CC Switch')
    expect(copy).toContain('Cherry Studio')
    expect(copy).toContain('Chatbox')
    expect(copy).not.toContain('Developer API Platform')
  })
})
