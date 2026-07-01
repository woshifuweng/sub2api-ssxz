import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('home model availability copy', () => {
  it('does not promise specific models are already supported in Chinese copy', () => {
    const providers = zh.home.providers

    expect(providers.title).toBe('模型能力以后台配置为准')
    expect(providers.description).toContain('实际可用模型以后端配置和账户分组为准')
    expect(providers.supported).toBe('后台配置')
    expect(providers.backendConfigured).toBe('后台配置')
    expect(JSON.stringify(providers)).not.toContain('已支持')
  })

  it('does not promise specific models are already supported in English copy', () => {
    const providers = en.home.providers

    expect(providers.title).toBe('Model availability follows backend configuration')
    expect(providers.description).toContain('Actual availability depends on backend configuration')
    expect(providers.supported).toBe('Backend configured')
    expect(providers.backendConfigured).toBe('Backend configured')
    expect(JSON.stringify(providers)).not.toContain('Supported')
  })
})
