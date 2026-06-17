import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('image generation locale copy', () => {
  it('uses neutral zh product copy for the legacy /sora entry', () => {
    const visibleCopy = [
      zh.nav.sora,
      zh.sora.title,
      zh.sora.description,
      zh.sora.notEnabledDesc,
      zh.sora.welcomeTitle,
      zh.sora.welcomeSubtitle,
      zh.sora.creatorPlaceholder
    ].join(' ')

    expect(visibleCopy).toContain('图片生成')
    expect(visibleCopy).toContain('作品库')
    expect(visibleCopy).not.toContain('Sora')
    expect(visibleCopy).not.toContain('视频')
  })

  it('uses neutral en product copy for the legacy /sora entry', () => {
    const visibleCopy = [
      en.nav.sora,
      en.sora.title,
      en.sora.description,
      en.sora.notEnabledDesc,
      en.sora.welcomeTitle,
      en.sora.welcomeSubtitle,
      en.sora.creatorPlaceholder
    ].join(' ')

    expect(visibleCopy).toContain('Image Generation')
    expect(visibleCopy).toContain('library')
    expect(visibleCopy).not.toContain('Sora')
    expect(visibleCopy).not.toContain('video')
  })
})
