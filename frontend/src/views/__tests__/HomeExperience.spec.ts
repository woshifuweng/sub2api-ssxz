import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('HomeView visual boundary', () => {
  it('mounts the approved Aether-derived home experience without changing custom home content', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/HomeView.vue'), 'utf8')
    const mainSource = readFileSync(resolve(process.cwd(), 'src/main.ts'), 'utf8')

    expect(source).toContain('AetherHomeExperience')
    expect(source).toContain('v-if="homeContent"')
    expect(source).toContain('renderedHomeContent')
    expect(source).toContain(':base-url="displayedApiBaseUrl"')
    expect(source).toContain(':theme="theme"')
    expect(source).toContain("if (typeof document === 'undefined') return 'dark'")
    expect(source).toContain("const theme = ref<'light' | 'dark'>(getInitialTheme())")
    expect(mainSource).toContain("const shouldUseDark = savedTheme === 'light' ? false : true")
    expect(source).not.toContain('HomeScrollShowcase')
    expect(source).not.toContain('HomeBentoGrid')
    expect(source).not.toContain('ModelIcon')
    expect(source).not.toContain('provider-motion-scan')
    expect(source).not.toContain('gateway-network')
    expect(source).not.toContain("document.documentElement.classList.add('dark')")
    expect(source).not.toContain('REAL PRODUCT')
    expect(source).not.toContain('OPENAI-COMPATIBLE / READY TO CONNECT')
    expect(source).not.toContain('从一个真实请求开始')
    expect(source).not.toContain('READY WHEN YOU ARE')
    expect(source).not.toContain('返利')

    const experience = readFileSync(
      resolve(process.cwd(), 'src/components/home/aether/AetherHomeExperience.vue'),
      'utf8'
    )
    expect(experience.match(/<CliSection/g)).toHaveLength(3)
    expect(experience).toContain('接入省事，用量和账单都清楚')
    expect(experience).toContain('统一入口，计费透明，稳定可靠')
    expect(experience).toContain('<h2>接入省事，用量和账单都清楚</h2>')
    expect(experience).not.toContain('<h2 :style="getTitleStyle(SECTIONS.FEATURES)">')
    for (const unfinishedCopy of ['进度', '已完成', '开发中', '载入更多', '敬请期待']) {
      expect(experience).not.toContain(unfinishedCopy)
    }
    expect(experience.toLowerCase()).not.toContain('roadmap')
    expect(experience).toContain('创建 API Key，立即开始')
    expect(experience).toContain('<HomeDocsDialog')
  })
})
