import { describe, expect, it } from 'vitest'

import { resolveGroupDisplayPlatform } from '../groupPlatform'

describe('resolveGroupDisplayPlatform', () => {
  it('uses the Claude brand for Claude-named groups even when routing uses OpenAI compatibility', () => {
    expect(resolveGroupDisplayPlatform('Claude Kiro高缓池', 'openai')).toBe('anthropic')
    expect(resolveGroupDisplayPlatform('Claude 满血池(CCMAX)', 'openai')).toBe('anthropic')
  })

  it('preserves the technical platform for other groups', () => {
    expect(resolveGroupDisplayPlatform('满血 GPT号池', 'openai')).toBe('openai')
    expect(resolveGroupDisplayPlatform('Claude Kiro高缓池', 'kiro')).toBe('anthropic')
    expect(resolveGroupDisplayPlatform('Kiro高缓池', 'kiro')).toBe('kiro')
  })
})
