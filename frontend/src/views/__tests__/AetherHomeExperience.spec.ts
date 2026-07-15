import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { ref } from 'vue'
import { describe, expect, it } from 'vitest'
import {
  normalizeApiBaseUrl,
  SECTIONS,
  sections,
  useCliConfigs
} from '@/components/home/aether/home-config'

function readSource(path: string): string {
  return readFileSync(resolve(process.cwd(), path), 'utf8')
}

describe('Aether-derived home structure', () => {
  it('keeps the original three-provider section skeleton and native animated artwork', () => {
    const home = readSource('src/components/home/aether/AetherHomeExperience.vue')
    const cliSection = readSource('src/components/home/aether/CliSection.vue')
    const rippleLogo = readSource('src/components/home/aether/RippleLogo.vue')
    const geminiStars = readSource('src/components/home/aether/GeminiStarCluster.vue')
    const sectionAnimations = readSource('src/components/home/aether/useSectionAnimations.ts')

    expect(home.match(/<CliSection/g)).toHaveLength(3)
    expect(home.match(/content-position="right"/g)).toHaveLength(2)
    expect(home.match(/content-position="left"/g)).toHaveLength(1)
    expect(home).toContain('<RippleLogo')
    expect(home).toContain('<GeminiStarCluster')
    expect(home).toContain('<HomeDocsDialog')
    expect(home).toContain('useSectionAnimations')
    expect(home).toContain('useLogoPosition')
    expect(home).toContain('useLogoTransition')
    expect(home).not.toContain('ModelIcon')
    expect(home).not.toContain('AetherLineByLineLogo')
    expect(home).not.toContain('feature.status')
    expect(home).toContain('接入省事，用量和账单都清楚')
    expect(home).toContain('统一入口，计费透明，稳定可靠')
    expect(home).toContain('<h2>接入省事，用量和账单都清楚</h2>')
    expect(home).not.toContain('<h2 :style="getTitleStyle(SECTIONS.FEATURES)">')
    expect(home).not.toMatch(/class="aether-features__intro"\s+:style=/)
    expect(home).toMatch(
      /\.aether-features h2\s*\{[^}]*font-family: var\(--ssxz-font-sans/s
    )
    expect(home).toMatch(
      /\.aether-features__intro\s*\{[^}]*font-family: var\(--ssxz-font-sans/s
    )
    expect(home).not.toContain('aether-features__eyebrow')
    for (const unfinishedCopy of ['进度', '已完成', '开发中', '载入更多', '敬请期待']) {
      expect(home).not.toContain(unfinishedCopy)
    }
    expect(home.toLowerCase()).not.toContain('roadmap')
    expect(home).not.toMatch(/#cc785c|#d4a27f/i)
    expect(home).not.toContain('backdrop-filter: blur(4px)')
    expect(home).not.toContain('v-if="docUrl"')
    expect(home).toContain('docsOpen = true')
    expect(home).toContain('<RouterLink to="/docs" class="aether-nav__docs">')
    expect(home).toMatch(
      /<RouterLink\s+to="\/docs"\s+class="aether-header__docs-mobile"[\s\S]*?aria-label="打开公开接入文档"/
    )
    expect(home).toContain('linear-gradient(var(--aether-hero-grid) 1px, transparent 1px)')
    expect(home).toMatch(
      /\.aether-home\s*\{[^}]*background-image:\s*linear-gradient\(var\(--aether-hero-grid\)/s
    )
    expect(home).not.toContain('class="aether-hero-background"')
    expect(home).not.toMatch(/\.aether-hero-background\s*\{/)
    expect(home).toContain('--aether-hero-grid: hsl(var(--foreground) / 0.05);')
    expect(home).toMatch(
      /\.aether-brand__mark\s*\{[^}]*width: 2\.875rem;[^}]*border: 0;[^}]*background: transparent;/s
    )
    expect(home).toContain('<BrandLogo variant="mark" size="2.875rem" :theme="theme" />')
    expect(home).not.toContain('.aether-brand__mark img')
    expect(home).not.toContain('filter: brightness(0)')
    expect(home).not.toContain('drop-shadow')
    expect(home).not.toContain('#c9a55b')
    expect(home).toContain("mask: url('/brand/ssxz-cat-dog-static.svg')")
    expect(home).toContain(':theme="theme"')
    expect(home).toContain('--aether-hero-surface: hsl(var(--background));')
    expect(home).not.toContain('--aether-hero-surface: hsl(220 8% 7%);')
    expect(home).not.toContain("'aether-header--hero': currentSection === SECTIONS.HOME")
    expect(home).not.toContain("'scroll-indicator--hero': currentSection === SECTIONS.HOME")
    expect(home).toContain('.aether-home:not(.aether-home--dark) {')
    expect(home).toContain('--background: 44 24% 97%;')
    expect(home).toContain('--card: 42 25% 99%;')
    expect(home).toContain('color: var(--aether-hero-ink);')
    expect(home).not.toContain('--aether-hero-logo-filter')
    expect(home).not.toContain('.logo-container.home-section::after')
    expect(home).not.toMatch(/\.ssxz-brand-logo[^{]*\{[^}]*filter:/s)
    expect(home).toContain('width: min(25rem, 44vh);')
    expect(home).toContain('height: min(46vh, 29rem);')
    expect(home).toContain('font-size: clamp(2.35rem, 5vw, 4.25rem);')
    expect(home).toContain('class="aether-hero__models"')
    expect(home).toContain('<li>OpenAI</li>')
    expect(home).toContain('<li>Claude</li>')
    expect(home).toContain('<li>Gemini</li>')
    expect(home).toContain('<li>Grok</li>')
    expect(home).toContain('class="aether-features__closing"')
    expect(home).toContain('创建 API Key，立即开始')
    expect(cliSection).toContain('class="min-w-0"')
    expect(sectionAnimations).toContain("transform = 'scale(1) translateY(-18vh)'")

    expect(rippleLogo).toContain("type LogoType = 'claude' | 'openai' | 'gemini'")
    expect(rippleLogo).toContain('class="openai-outline"')
    expect(rippleLogo).toContain('class="claude-outline"')
    expect(rippleLogo).toContain('class="gemini-outline"')
    expect(rippleLogo).not.toContain('ModelIcon')
    expect(rippleLogo).not.toContain('logoPaths')
    expect(geminiStars).toContain('twinkleDuration')
    expect(geminiStars).toContain('gemini-star')
  })

  it('draws the SSXZ vector artwork without falling back to the PNG', () => {
    const brandLogo = readSource('src/components/home/aether/SsxzBrandLogo.vue')
    const sharedBrandLogo = readSource('src/components/common/BrandLogo.vue')
    const animatedLogo = readSource('public/brand/ssxz-cat-dog-line-draw.svg')
    const staticLogo = readSource('public/brand/ssxz-cat-dog-static.svg')

    expect(brandLogo).toContain('<BrandLogo variant="animated" size="100%" :theme="theme" />')
    expect(brandLogo).toContain("theme: 'light' | 'dark'")
    expect(sharedBrandLogo).toContain('ssxz-cat-dog-line-draw.svg')
    expect(sharedBrandLogo).toContain('ssxz-cat-dog-static.svg')
    expect(sharedBrandLogo).toContain('brand-logo__artwork--static')
    expect(sharedBrandLogo).toContain("variant?: 'mark' | 'animated'")
    expect(sharedBrandLogo).toContain("mask: url('/brand/ssxz-cat-dog-static.svg')")
    expect(brandLogo).not.toMatch(/\.png/i)
    expect(brandLogo).not.toContain('AETHER_')
    expect(animatedLogo).toContain('created with Arrow by QuiverAI')
    expect(animatedLogo).toContain('ssxz-line-draw')
    expect(animatedLogo).toContain('class="ssxz-outline"')
    expect(animatedLogo).toContain('class="ssxz-color"')
    expect(animatedLogo).toContain('ssxz-color-form')
    expect(animatedLogo).toContain('ssxz-color-breathe')
    expect(animatedLogo).toContain('stroke-dashoffset: 1')
    expect(animatedLogo).toContain('prefers-reduced-motion: reduce')
    expect(animatedLogo.match(/pathLength="1"/g)).toHaveLength(108)
    expect(animatedLogo.match(/<path class="cls-/g)).toHaveLength(108)
    expect(animatedLogo).not.toContain('#030101')
    expect(animatedLogo).toMatch(/\.cls-0\s*\{fill:none;stroke:#FFE1BB;/)
    expect(animatedLogo).toMatch(/\.cls-dog-head\s*\{fill:none;stroke:#FFE1BB;/)
    expect(animatedLogo).toContain('@media (prefers-color-scheme: light)')
    expect(animatedLogo).toMatch(/@media \(prefers-color-scheme: light\)[\s\S]*\.cls-1\s*\{fill:#71491D;/)
    expect(animatedLogo).toMatch(/@media \(prefers-color-scheme: light\)[\s\S]*\.cls-5\s*\{fill:#252A30;/)
    expect(animatedLogo).toMatch(/@media \(prefers-color-scheme: light\)[\s\S]*\.ssxz-draw--warm\s*\{ stroke: #704A1D; \}/)
    expect(animatedLogo).toMatch(/@media \(prefers-color-scheme: light\)[\s\S]*\.ssxz-draw--light\s*\{ stroke: #252A30; \}/)
    expect(animatedLogo).toMatch(/<path class="cls-dog-head" d="m103\.9 62\.6/)
    expect(animatedLogo).not.toMatch(/<path class="cls-0" d="m103\.9 62\.6/)
    expect(animatedLogo).toContain('--path-index:107')
    expect(animatedLogo).not.toContain('<rect')
    expect(Buffer.byteLength(animatedLogo)).toBeLessThan(40_000)
    expect(staticLogo.match(/<path\b/g)).toHaveLength(108)
    expect(staticLogo).not.toContain('#030101')
    expect(staticLogo).toMatch(/\.cls-0\s*\{fill:none;stroke:#FFE1BB;/)
    expect(staticLogo).toMatch(/\.cls-dog-head\s*\{fill:none;stroke:#FFE1BB;/)
    expect(staticLogo).toContain('@media (prefers-color-scheme: light)')
    expect(staticLogo).toMatch(/@media \(prefers-color-scheme: light\)[\s\S]*\.cls-1\s*\{fill:#71491D;/)
    expect(staticLogo).toMatch(/@media \(prefers-color-scheme: light\)[\s\S]*\.cls-5\s*\{fill:#252A30;/)
    expect(staticLogo).toMatch(/<path class="cls-dog-head" d="m103\.9 62\.6/)
    expect(staticLogo).not.toMatch(/<path class="cls-0" d="m103\.9 62\.6/)
    expect(staticLogo).not.toContain('<rect')
    expect(Buffer.byteLength(staticLogo)).toBeLessThan(15_000)
    expect(existsSync(resolve(process.cwd(), 'public/brand/ssxz-double-s-cat-dog-draw.svg'))).toBe(false)
    expect(existsSync(resolve(process.cwd(), 'src/assets/home/ssxz-double-s-cat-dog-logo.png'))).toBe(false)
  })

  it('uses the approved section order and dynamic SSXZ gateway configuration', () => {
    expect(SECTIONS).toEqual({
      HOME: 0,
      CLAUDE: 1,
      CODEX: 2,
      GEMINI: 3,
      FEATURES: 4
    })
    expect(sections.map(section => section.name)).toEqual([
      '首页',
      'Claude Code',
      'Codex CLI',
      'Gemini CLI',
      '平台能力'
    ])

    const baseUrl = ref('https://gateway.example/v1')
    const configs = useCliConfigs(baseUrl)
    const renderedConfigs = Object.values(configs).map(config => config.value).join('\n')

    expect(configs.claudeConfig.value).toContain('https://gateway.example')
    expect(configs.claudeConfig.value).not.toContain('https://gateway.example/v1/v1')
    expect(configs.codexConfig.value).toContain('base_url = "https://gateway.example/v1"')
    expect(configs.codexConfig.value).toContain('model_provider = "ssxz"')
    expect(configs.geminiEnvConfig.value).toContain('GOOGLE_GEMINI_BASE_URL=https://gateway.example')
    expect(renderedConfigs).toContain('your-api-key')
    expect(renderedConfigs).toContain('<当前 Key 可用模型>')
    expect(renderedConfigs).not.toContain('Aether')
    expect(normalizeApiBaseUrl('https://gateway.example/')).toBe('https://gateway.example/v1')
    expect(normalizeApiBaseUrl('https://gateway.example/v1')).toBe('https://gateway.example/v1')
  })
})
