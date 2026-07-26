<template>
  <div
    ref="scrollContainer"
    class="aether-home"
    :class="{ 'aether-home--dark': theme === 'dark' }"
  >
    <nav class="scroll-indicator" aria-label="首页章节">
      <button
        v-for="(section, index) in sections"
        :key="section.name"
        type="button"
        class="scroll-indicator__button"
        :aria-label="`前往${section.name}`"
        :aria-current="currentSection === index ? 'step' : undefined"
        @click="scrollToSection(index)"
      >
        <span class="scroll-indicator__label">{{ section.name }}</span>
        <span class="scroll-indicator__dot" :class="{ 'is-active': currentSection === index }" />
      </button>
    </nav>

    <header class="aether-header">
      <div class="aether-header__inner">
        <button
          type="button"
          class="aether-brand"
          aria-label="返回首页首屏"
          @click="scrollToSection(SECTIONS.HOME)"
        >
          <span class="aether-brand__mark">
            <BrandLogo variant="mark" size="2.875rem" :theme="theme" />
          </span>
          <span class="aether-brand__copy">
            <strong>SSXZ</strong>
            <small>AI 开发工具统一接入平台</small>
          </span>
        </button>

        <nav class="aether-nav" aria-label="首页导航">
          <button
            v-for="(section, index) in sections"
            :key="section.name"
            type="button"
            :class="{ 'is-active': currentSection === index }"
            @click="scrollToSection(index)"
          >
            {{ section.name }}
          </button>
          <RouterLink to="/docs" class="aether-nav__docs">
            文档
          </RouterLink>
        </nav>

        <div class="aether-header__actions">
          <RouterLink
            to="/docs"
            class="aether-header__docs-mobile"
            title="公开接入文档"
            aria-label="打开公开接入文档"
          >
            <BookOpen aria-hidden="true" />
          </RouterLink>
          <div class="aether-locale">
            <LocaleSwitcher />
          </div>
          <FoundationButton
            variant="ghost"
            size="icon"
            :title="themeLabel"
            :aria-label="themeLabel"
            :aria-pressed="theme === 'dark'"
            @click="emit('toggle-theme')"
          >
            <Sun v-if="theme === 'dark'" aria-hidden="true" />
            <Moon v-else aria-hidden="true" />
          </FoundationButton>
          <RouterLink
            v-if="!isAuthenticated"
            to="/login"
            class="aether-header__login"
          >
            登录
          </RouterLink>
          <RouterLink :to="primaryCtaPath" class="f0-button f0-button--default aether-header__cta">
            {{ isAuthenticated ? '进入控制台' : '开始使用' }}
          </RouterLink>
        </div>
      </div>
    </header>

    <main class="aether-main">
      <div class="fixed-logo-stage" aria-hidden="true">
        <Transition name="fade">
          <GeminiStarCluster
            v-if="currentSection === SECTIONS.GEMINI"
            :is-visible="sectionVisibility[SECTIONS.GEMINI] > 0.05"
            class="gemini-stars"
            :class="{ 'is-mobile': windowWidth < 768 }"
            :style="fixedLogoStyle"
          />
        </Transition>

        <div
          class="logo-container"
          :class="[
            currentSection === SECTIONS.HOME ? 'home-section' : '',
            `logo-transition-${scrollDirection}`
          ]"
          :style="fixedLogoStyle"
        >
          <Transition :name="logoTransitionName">
            <SsxzBrandLogo
              v-if="currentSection === SECTIONS.HOME"
              :key="`ssxz-${currentSection}-${theme}`"
              :theme="theme"
            />
            <div
              v-else-if="currentSection !== SECTIONS.FEATURES"
              :key="`ripple-wrapper-${currentLogoType}`"
              :class="{ 'heartbeat-wrapper': currentSection === SECTIONS.GEMINI && geminiFillComplete }"
            >
              <RippleLogo
                ref="rippleLogoRef"
                :type="currentLogoType"
                :size="windowWidth < 768 ? 200 : 320"
                :disable-ripple="currentSection === SECTIONS.GEMINI"
                :anim-delay="logoTransitionDelay"
                class="logo-active"
                :class="[currentLogoClass]"
              />
            </div>
          </Transition>
        </div>
      </div>

      <section ref="section0" class="aether-section aether-hero">
        <div class="aether-hero__content">
          <div class="aether-hero__logo-space" />
          <p class="aether-hero__brand" :style="getTitleStyle(SECTIONS.HOME)">
            <span class="typewriter">
              {{ brandText }}<span class="cursor" :class="{ 'is-hidden': !showCursor }">_</span>
            </span>
          </p>
          <h1 :style="getTitleStyle(SECTIONS.HOME)">一个 API Key，调用 Claude 与 GPT 系列官方模型</h1>
          <p :style="getDescStyle(SECTIONS.HOME)">按量计费，失败不扣费，每一笔消耗都有明细可查。</p>
          <div class="aether-hero__actions" :style="getButtonsStyle(SECTIONS.HOME)">
            <RouterLink :to="createKeyPath" class="f0-button f0-button--default f0-button--lg">
              创建 API Key
              <ArrowRight aria-hidden="true" />
            </RouterLink>
            <button
              type="button"
              class="f0-button f0-button--outline f0-button--lg"
              @click="scrollToSection(SECTIONS.CLAUDE)"
            >
              查看接入方式
            </button>
          </div>
          <ul class="aether-hero__models" aria-label="支持的模型提供商">
            <li>OpenAI</li>
            <li>Claude</li>
            <li>Gemini</li>
          </ul>
          <button
            type="button"
            class="aether-hero__scroll"
            aria-label="滚动到 Claude Code 接入方式"
            :style="getScrollIndicatorStyle(SECTIONS.HOME)"
            @click="scrollToSection(SECTIONS.CLAUDE)"
          >
            <ChevronDown aria-hidden="true" />
          </button>
        </div>
      </section>

      <CliSection
        ref="section1"
        v-model:platform-value="claudePlatform"
        title="Claude Code"
        description="安装 Claude Code 后，将认证令牌与 Base URL 指向 SSXZ，即可使用当前 Key 所属分组开放的 Claude 模型。"
        :badge-icon="Code2"
        badge-text="AI 编程助手"
        badge-class="border border-[hsl(var(--border))] bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]"
        :platform-options="platformPresets.claude.options"
        :install-command="claudeInstallCommand"
        :config-files="[{ path: '~/.claude/settings.json', content: claudeConfig, language: 'json' }]"
        :badge-style="getBadgeStyle(SECTIONS.CLAUDE)"
        :title-style="getTitleStyle(SECTIONS.CLAUDE)"
        :desc-style="getDescStyle(SECTIONS.CLAUDE)"
        :card-style-fn="idx => getCardStyle(SECTIONS.CLAUDE, idx)"
        content-position="right"
        @copy="copyText"
      />

      <CliSection
        ref="section2"
        v-model:platform-value="codexPlatform"
        title="Codex CLI"
        description="保留 Codex CLI 的 Responses 协议，只需新增 SSXZ provider 并填入当前 Key，无需改动日常命令习惯。"
        :badge-icon="Terminal"
        badge-text="命令行接入"
        badge-class="border border-[hsl(var(--border))] bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]"
        :platform-options="platformPresets.codex.options"
        :install-command="codexInstallCommand"
        :config-files="[
          { path: '~/.codex/config.toml', content: codexConfig, language: 'toml' },
          { path: '~/.codex/auth.json', content: codexAuthConfig, language: 'json' }
        ]"
        :badge-style="getBadgeStyle(SECTIONS.CODEX)"
        :title-style="getTitleStyle(SECTIONS.CODEX)"
        :desc-style="getDescStyle(SECTIONS.CODEX)"
        :card-style-fn="idx => getCardStyle(SECTIONS.CODEX, idx)"
        content-position="left"
        @copy="copyText"
      />

      <CliSection
        ref="section3"
        v-model:platform-value="geminiPlatform"
        title="Gemini CLI"
        description="通过环境变量接入 SSXZ，Base URL 随当前站点自动生成；具体模型以当前 Key 的实际可用目录为准。"
        :badge-icon="Sparkles"
        badge-text="多模态 AI"
        badge-class="border border-[hsl(var(--border))] bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]"
        :platform-options="platformPresets.gemini.options"
        :install-command="geminiInstallCommand"
        :config-files="[
          { path: '~/.gemini/.env', content: geminiEnvConfig, language: 'dotenv' },
          { path: '~/.gemini/settings.json', content: geminiSettingsConfig, language: 'json' }
        ]"
        :badge-style="getBadgeStyle(SECTIONS.GEMINI)"
        :title-style="getTitleStyle(SECTIONS.GEMINI)"
        :desc-style="getDescStyle(SECTIONS.GEMINI)"
        :card-style-fn="idx => getCardStyle(SECTIONS.GEMINI, idx)"
        content-position="right"
        @copy="copyText"
      />

      <section ref="section4" class="aether-section aether-features">
        <div class="aether-features__content">
          <h2>接入省事，用量和账单都清楚</h2>
          <p class="aether-features__intro">
            统一入口，计费透明，稳定可靠
          </p>

          <div class="aether-features__grid">
            <article
              v-for="(feature, index) in featureCards"
              :key="feature.title"
              class="aether-feature-card"
              :style="getFeatureCardStyle(SECTIONS.FEATURES, index)"
            >
              <span class="aether-feature-card__icon">
                <component :is="feature.icon" aria-hidden="true" />
              </span>
              <h3>{{ feature.title }}</h3>
              <p>{{ feature.desc }}</p>
            </article>
          </div>

          <div class="aether-features__closing" :style="getButtonsStyle(SECTIONS.FEATURES)">
            <h3>创建 API Key，立即开始</h3>
            <p>选择可用分组后，将 Base URL 和 API Key 填入客户端即可。</p>
            <div class="aether-features__actions">
              <RouterLink :to="createKeyPath" class="f0-button f0-button--default f0-button--lg">
                创建 API Key
                <ArrowRight aria-hidden="true" />
              </RouterLink>
              <button
                type="button"
                class="f0-button f0-button--outline f0-button--lg"
                @click="docsOpen = true"
              >
                查看接入文档
              </button>
            </div>
          </div>
        </div>
      </section>
    </main>

    <HomeDocsDialog
      v-model:open="docsOpen"
      :base-url="baseUrl"
      :doc-url="docUrl"
      :create-key-path="createKeyPath"
      @copy="copyText"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, toRef, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { ArrowRight, BookOpen, ChevronDown, Code2, Moon, Sparkles, Sun, Terminal } from '@lucide/vue'
import BrandLogo from '@/components/common/BrandLogo.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { FoundationButton } from '@/components/foundation'
import CliSection from './CliSection.vue'
import GeminiStarCluster from './GeminiStarCluster.vue'
import HomeDocsDialog from './HomeDocsDialog.vue'
import RippleLogo from './RippleLogo.vue'
import SsxzBrandLogo from './SsxzBrandLogo.vue'
import {
  featureCards,
  getLogoClass,
  getLogoType,
  SECTIONS,
  sections,
  useCliConfigs
} from './home-config'
import { getInstallCommand, platformPresets } from './platform-presets'
import {
  useLogoPosition,
  useLogoTransition,
  useSectionAnimations
} from './useSectionAnimations'

const props = defineProps<{
  theme: 'light' | 'dark'
  siteName: string
  docUrl: string
  baseUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  primaryCtaPath: string
  createKeyPath: string
}>()

const emit = defineEmits<{
  copy: [text: string]
  'toggle-theme': []
}>()

const scrollContainer = ref<HTMLElement | null>(null)
const docsOpen = ref(false)
const currentSection = ref(0)
const previousSection = ref(0)
const scrollDirection = ref<'up' | 'down'>('down')
const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1024)
const sectionVisibility = ref<number[]>([1, 0, 0, 0, 0])
const section0 = ref<HTMLElement | null>(null)
const section1 = ref<InstanceType<typeof CliSection> | null>(null)
const section2 = ref<InstanceType<typeof CliSection> | null>(null)
const section3 = ref<InstanceType<typeof CliSection> | null>(null)
const section4 = ref<HTMLElement | null>(null)
const rippleLogoRef = ref<InstanceType<typeof RippleLogo> | null>(null)
const geminiFillComplete = ref(false)
let lastScrollY = 0
let scrollEndTimer: ReturnType<typeof setTimeout> | null = null
let typewriterTimer: ReturnType<typeof setTimeout> | null = null

const {
  getBadgeStyle,
  getTitleStyle,
  getDescStyle,
  getButtonsStyle,
  getScrollIndicatorStyle,
  getCardStyle,
  getFeatureCardStyle
} = useSectionAnimations(sectionVisibility)
const { fixedLogoStyle } = useLogoPosition(currentSection, windowWidth)
const { logoTransitionName } = useLogoTransition(currentSection, previousSection)
const currentLogoType = computed(() => getLogoType(currentSection.value))
const currentLogoClass = computed(() => getLogoClass(currentSection.value))
const logoTransitionDelay = computed(() => previousSection.value === SECTIONS.FEATURES ? 200 : 500)
const themeLabel = computed(() => props.theme === 'dark' ? '切换到亮色模式' : '切换到暗色模式')

const claudePlatform = ref<string>(platformPresets.claude.defaultValue)
const codexPlatform = ref<string>(platformPresets.codex.defaultValue)
const geminiPlatform = ref<string>(platformPresets.gemini.defaultValue)
const claudeInstallCommand = computed(() => getInstallCommand('claude', claudePlatform.value))
const codexInstallCommand = computed(() => getInstallCommand('codex', codexPlatform.value))
const geminiInstallCommand = computed(() => getInstallCommand('gemini', geminiPlatform.value))
const {
  claudeConfig,
  codexConfig,
  codexAuthConfig,
  geminiEnvConfig,
  geminiSettingsConfig
} = useCliConfigs(toRef(props, 'baseUrl'))

const brandText = ref('')
const showCursor = ref(true)

function getSectionElement(index: number): HTMLElement | null {
  switch (index) {
    case SECTIONS.HOME: return section0.value
    case SECTIONS.CLAUDE: return section1.value?.sectionEl ?? null
    case SECTIONS.CODEX: return section2.value?.sectionEl ?? null
    case SECTIONS.GEMINI: return section3.value?.sectionEl ?? null
    case SECTIONS.FEATURES: return section4.value
    default: return null
  }
}

function copyText(text: string): void {
  emit('copy', text)
}

function startTypewriter(): void {
  if (typewriterTimer) clearTimeout(typewriterTimer)
  const fullText = props.siteName || 'SSXZ'
  let index = 0
  let deleting = false

  const step = () => {
    if (!deleting) {
      index = Math.min(index + 1, fullText.length)
      brandText.value = fullText.slice(0, index)
      showCursor.value = true
      if (index === fullText.length) {
        deleting = true
        typewriterTimer = setTimeout(step, 3500)
        return
      }
      typewriterTimer = setTimeout(step, 180)
      return
    }

    index = Math.max(index - 1, 0)
    brandText.value = fullText.slice(0, index)
    if (index === 0) {
      deleting = false
      typewriterTimer = setTimeout(step, 900)
      return
    }
    typewriterTimer = setTimeout(step, 110)
  }

  step()
}

function calculateVisibility(element: HTMLElement | null): number {
  if (!element) return 0
  const rect = element.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const center = rect.top + rect.height / 2
  const distance = Math.abs(center - viewportHeight / 2)
  const maxDistance = viewportHeight / 2 + rect.height / 2
  return Math.max(0, Math.min(1, 1 - distance / maxDistance))
}

function handleScroll(): void {
  if (!scrollContainer.value) return
  const scrollY = scrollContainer.value.scrollTop
  scrollDirection.value = scrollY >= lastScrollY ? 'down' : 'up'
  lastScrollY = scrollY

  for (let index = 0; index < sections.length; index += 1) {
    sectionVisibility.value[index] = calculateVisibility(getSectionElement(index))
  }

  const scrollMiddle = scrollY + window.innerHeight / 2
  for (let index = 0; index < sections.length; index += 1) {
    const section = getSectionElement(index)
    if (!section) continue
    const top = section.offsetTop
    if (scrollMiddle >= top && scrollMiddle < top + section.offsetHeight) {
      if (currentSection.value !== index) {
        previousSection.value = currentSection.value
        currentSection.value = index
      }
      break
    }
  }

  if (scrollEndTimer) clearTimeout(scrollEndTimer)
  scrollEndTimer = setTimeout(() => {
    if (currentSection.value === SECTIONS.HOME && brandText.value === '') startTypewriter()
  }, 150)
}

function scrollToSection(index: number): void {
  getSectionElement(index)?.scrollIntoView({ behavior: 'smooth' })
}

function handleResize(): void {
  windowWidth.value = window.innerWidth
  handleScroll()
}

watch(
  () => rippleLogoRef.value?.fillComplete,
  value => {
    if (currentSection.value === SECTIONS.GEMINI && value) geminiFillComplete.value = true
  }
)

watch(currentSection, (_, previous) => {
  if (previous === SECTIONS.GEMINI) geminiFillComplete.value = false
})

onMounted(() => {
  scrollContainer.value?.addEventListener('scroll', handleScroll, { passive: true })
  window.addEventListener('resize', handleResize, { passive: true })
  handleScroll()
  typewriterTimer = setTimeout(startTypewriter, 450)
})

onUnmounted(() => {
  scrollContainer.value?.removeEventListener('scroll', handleScroll)
  window.removeEventListener('resize', handleResize)
  if (scrollEndTimer) clearTimeout(scrollEndTimer)
  if (typewriterTimer) clearTimeout(typewriterTimer)
})
</script>

<style scoped>
.aether-home {
  --color-background: hsl(var(--background));
  --color-border: hsl(var(--border));
  --color-text: hsl(var(--foreground));
  --color-code-background: hsl(var(--muted) / 0.72);
  --color-code-text: hsl(var(--foreground));
  --aether-hero-surface: hsl(var(--background));
  --aether-hero-ink: hsl(var(--foreground));
  --aether-hero-muted: hsl(var(--muted-foreground));
  --aether-hero-grid: hsl(var(--foreground) / 0.05);
  --aether-hero-glow: hsl(var(--foreground));
  --aether-hero-primary-surface: hsl(var(--primary));
  --aether-hero-primary-ink: hsl(var(--primary-foreground));
  --aether-hero-outline-surface: hsl(var(--background));
  --aether-hero-outline-ink: hsl(var(--foreground));
  --aether-hero-outline-border: hsl(var(--border));
  --aether-hero-scroll-ink: hsl(var(--muted-foreground));

  position: relative;
  height: 100vh;
  height: 100dvh;
  overflow-x: hidden;
  overflow-y: auto;
  scroll-behavior: smooth;
  scroll-snap-type: y mandatory;
  color: hsl(var(--foreground));
  background-color: hsl(var(--background));
  background-image:
    linear-gradient(var(--aether-hero-grid) 1px, transparent 1px),
    linear-gradient(90deg, var(--aether-hero-grid) 1px, transparent 1px);
  background-position: center;
  background-size: 2rem 2rem;
}

.aether-home:not(.aether-home--dark) {
  --background: 44 24% 97%;
  --foreground: 220 10% 10%;
  --card: 42 25% 99%;
  --card-foreground: 220 10% 10%;
  --popover: 42 25% 99%;
  --popover-foreground: 220 10% 10%;
  --secondary: 42 18% 93%;
  --secondary-foreground: 220 10% 16%;
  --muted: 42 16% 94%;
  --muted-foreground: 36 6% 42%;
  --accent: 40 14% 91%;
  --accent-foreground: 220 10% 16%;
  --border: 38 14% 84%;
  --input: 38 12% 78%;
  --ring: 220 8% 38%;
  --shadow: 24 12% 18% / 0.07;
  --button-shadow: 24 12% 18% / 0.16;
}

.aether-header {
  position: sticky;
  z-index: 50;
  top: 0;
  border-bottom: 1px solid hsl(var(--border) / 0.82);
  background: hsl(var(--background) / 0.97);
  transition: color 180ms ease, background-color 180ms ease, border-color 180ms ease;
}

.aether-header__inner {
  display: grid;
  width: min(82rem, calc(100% - 3rem));
  min-height: 4rem;
  grid-template-columns: minmax(12rem, 1fr) auto minmax(12rem, 1fr);
  align-items: center;
  gap: 1.5rem;
  margin: 0 auto;
}

.aether-brand {
  display: inline-flex;
  width: fit-content;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
  border: 0;
  padding: 0;
  color: inherit;
  background: transparent;
  cursor: pointer;
}

.aether-brand__mark {
  width: 2.875rem;
  height: 2.875rem;
  overflow: visible;
  flex: 0 0 auto;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.aether-brand__copy {
  display: grid;
  min-width: 0;
  text-align: left;
}

.aether-brand__copy strong {
  font-size: 0.875rem;
  font-weight: 720;
  line-height: 1rem;
}

.aether-brand__copy small {
  overflow: hidden;
  color: hsl(var(--muted-foreground));
  font-size: 0.625rem;
  line-height: 0.875rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.aether-nav,
.aether-header__actions {
  display: flex;
  align-items: center;
}

.aether-nav {
  gap: 0.25rem;
}

.aether-nav button,
.aether-nav a,
.aether-header__login {
  position: relative;
  min-height: 2.25rem;
  border: 0;
  border-radius: var(--radius);
  padding: 0 0.75rem;
  color: hsl(var(--muted-foreground));
  background: transparent;
  font-size: 0.75rem;
  font-weight: 620;
  line-height: 2.25rem;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
  transition: color 160ms ease, background-color 160ms ease;
}

.aether-header__docs-mobile {
  display: none;
  width: 2.25rem;
  height: 2.25rem;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius);
  color: hsl(var(--muted-foreground));
  text-decoration: none;
  transition: color 160ms ease, background-color 160ms ease;
}

.aether-header__docs-mobile:hover,
.aether-header__docs-mobile:focus-visible {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
}

.aether-nav button:hover,
.aether-nav button.is-active,
.aether-nav a:hover,
.aether-header__login:hover {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
}

.aether-header__actions {
  justify-content: flex-end;
  gap: 0.5rem;
}

.aether-locale :deep(button) {
  min-height: 2.25rem;
  border-radius: var(--radius);
}

.aether-header__cta {
  height: 2.25rem;
  min-height: 2.25rem;
  padding: 0 0.875rem;
  text-decoration: none;
}

.aether-main {
  position: relative;
  z-index: 10;
}

.fixed-logo-stage {
  position: fixed;
  z-index: 20;
  inset: 0;
  display: flex;
  overflow: hidden;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.logo-container {
  position: relative;
  display: flex;
  width: 20rem;
  height: 20rem;
  align-items: center;
  justify-content: center;
  transform: translateZ(0);
}

.logo-container.home-section {
  width: min(25rem, 44vh);
  height: min(25rem, 44vh);
}

.logo-container.home-section::before {
  position: absolute;
  z-index: 0;
  inset: 7%;
  background: var(--aether-hero-glow);
  content: '';
  filter: blur(1.75rem);
  mask: url('/brand/ssxz-cat-dog-static.svg') center / contain no-repeat;
  opacity: 0.1;
  transform: scale(1.06);
  -webkit-mask: url('/brand/ssxz-cat-dog-static.svg') center / contain no-repeat;
}

.logo-container > * {
  position: absolute;
  z-index: 1;
  inset: 0;
  display: flex;
  width: 100%;
  height: 100%;
  align-items: center;
  justify-content: center;
}

.gemini-stars {
  position: absolute;
  z-index: -1;
}

.gemini-stars.is-mobile {
  opacity: 0.6;
  transform: scale(0.75);
}

.aether-section {
  position: relative;
  z-index: 30;
  display: flex;
  min-height: 100vh;
  min-height: 100dvh;
  scroll-snap-align: start;
  align-items: center;
  justify-content: center;
  padding: 5rem 1rem;
}

.aether-hero {
  color: var(--aether-hero-ink);
  background: transparent;
}

.aether-hero__content {
  width: min(56rem, 100%);
  margin: 0 auto;
  text-align: center;
}

.aether-hero__logo-space {
  width: 100%;
  height: min(46vh, 29rem);
}

.aether-hero h1,
.aether-features h2,
.aether-feature-card h3 {
  letter-spacing: 0;
}

.aether-hero h1 {
  margin: 0;
  font-size: clamp(2.35rem, 5vw, 4.25rem);
  font-weight: 730;
  line-height: 1.04;
  text-wrap: balance;
  transition: opacity 700ms ease, transform 700ms ease;
}

.aether-hero .typewriter {
  color: var(--aether-hero-ink);
}

.aether-hero .cursor {
  margin-left: 0.0625rem;
  font-weight: 400;
  animation: cursor-blink 1s ease-in-out infinite;
}

.aether-hero .cursor.is-hidden {
  opacity: 0;
  animation: none;
}

.aether-hero p {
  max-width: 48rem;
  margin: 1.5rem auto 0;
  color: var(--aether-hero-muted);
  font-size: 1rem;
  line-height: 1.8;
  text-wrap: balance;
  transition: opacity 700ms ease, transform 700ms ease;
}

.aether-hero p.aether-hero__brand {
  margin: 0 auto 0.9rem;
  color: var(--aether-hero-ink);
  font-size: clamp(0.95rem, 1.4vw, 1.15rem);
  font-weight: 720;
  letter-spacing: 0.14em;
  line-height: 1.2;
  text-transform: uppercase;
}

.aether-hero__actions,
.aether-features__actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  transition: opacity 700ms ease, transform 700ms ease;
}

.aether-hero__actions {
  margin-top: 1.75rem;
}

.aether-hero__models {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0;
  margin: 1rem 0 0;
  padding: 0;
  color: var(--aether-hero-muted);
  font-size: 0.6875rem;
  font-weight: 620;
  line-height: 1rem;
  list-style: none;
}

.aether-hero__models li {
  display: inline-flex;
  align-items: center;
}

.aether-hero__models li + li::before {
  width: 0.1875rem;
  height: 0.1875rem;
  margin: 0 0.75rem;
  border-radius: 999px;
  background: currentColor;
  content: '';
  opacity: 0.5;
}

.aether-hero__actions a,
.aether-features__actions a {
  text-decoration: none;
}

.aether-hero__actions svg,
.aether-features__actions svg {
  width: 1rem;
  height: 1rem;
}

.aether-hero .f0-button--default {
  border-color: var(--aether-hero-primary-surface);
  color: var(--aether-hero-primary-ink);
  background: var(--aether-hero-primary-surface);
}

.aether-hero .f0-button--outline {
  border-color: var(--aether-hero-outline-border);
  color: var(--aether-hero-outline-ink);
  background: var(--aether-hero-outline-surface);
}

.aether-hero__scroll {
  display: inline-flex;
  width: 2.5rem;
  height: 2.5rem;
  align-items: center;
  justify-content: center;
  margin-top: 1rem;
  border: 0;
  color: var(--aether-hero-scroll-ink);
  background: transparent;
  cursor: pointer;
  transition: color 160ms ease, opacity 500ms ease, transform 500ms ease;
}

.aether-hero__scroll:hover {
  color: var(--aether-hero-ink);
}

.aether-hero__scroll svg {
  width: 1.75rem;
  height: 1.75rem;
  animation: scroll-cue 1.8s ease-in-out infinite;
}

.aether-features__content {
  width: min(72rem, 100%);
  margin: 0 auto;
  text-align: center;
}

.aether-features h2 {
  max-width: 48rem;
  margin: 0 auto;
  font-family: var(--ssxz-font-sans, Inter, ui-sans-serif, system-ui, sans-serif);
  font-size: clamp(2rem, 3.5vw, 2.75rem);
  font-weight: 700;
  line-height: 1.2;
  text-wrap: balance;
}

.aether-features__intro {
  max-width: 42rem;
  margin: 1.5rem auto 0;
  color: hsl(var(--muted-foreground));
  font-family: var(--ssxz-font-sans, Inter, ui-sans-serif, system-ui, sans-serif);
  font-size: 1rem;
  letter-spacing: 0;
  line-height: 1.6;
}

.aether-features__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.875rem;
  margin-top: 2rem;
}

.aether-feature-card {
  min-width: 0;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  padding: 1.5rem;
  text-align: left;
  background: hsl(var(--card));
  box-shadow: 0 12px 32px hsl(var(--shadow));
  transition:
    border-color 180ms ease,
    box-shadow 180ms ease,
    opacity 700ms ease,
    transform 700ms ease;
}

.aether-feature-card:hover {
  border-color: hsl(var(--foreground) / 0.22);
  box-shadow: 0 16px 40px hsl(var(--shadow));
}

.aether-feature-card__icon {
  display: inline-flex;
  width: 2.5rem;
  height: 2.5rem;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
}

.aether-feature-card__icon svg {
  width: 1.125rem;
  height: 1.125rem;
}

.aether-feature-card h3 {
  margin: 1rem 0 0;
  font-size: 0.9375rem;
  font-weight: 680;
  line-height: 1.25rem;
}

.aether-feature-card p {
  margin: 0.625rem 0 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  line-height: 1.625;
}

.aether-features__closing {
  margin-top: 2rem;
  padding-top: 1.5rem;
  border-top: 1px solid hsl(var(--border) / 0.72);
  text-align: center;
  transition: opacity 700ms ease, transform 700ms ease;
}

.aether-features__closing h3 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 2rem;
}

.aether-features__closing p {
  margin: 0.5rem auto 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.8125rem;
  line-height: 1.4rem;
}

.aether-features__actions {
  margin-top: 1.25rem;
}

.scroll-indicator {
  position: fixed;
  z-index: 60;
  top: 50%;
  right: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
  transform: translateY(-50%);
}

.scroll-indicator__button {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  border: 0;
  padding: 0.25rem;
  background: transparent;
  cursor: pointer;
}

.scroll-indicator__label {
  position: absolute;
  right: 1.5rem;
  opacity: 0;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--popover));
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
  line-height: 1rem;
  white-space: nowrap;
  pointer-events: none;
  transition: opacity 160ms ease;
}

.scroll-indicator__button:hover .scroll-indicator__label {
  opacity: 1;
}

.scroll-indicator__dot {
  width: 0.625rem;
  height: 0.625rem;
  border: 2px solid hsl(var(--border));
  border-radius: 50%;
  background: hsl(var(--background));
  transition: border-color 200ms ease, background-color 200ms ease, transform 200ms ease;
}

.scroll-indicator__dot.is-active {
  border-color: hsl(var(--foreground));
  background: hsl(var(--foreground));
  transform: scale(1.25);
}

.heartbeat-wrapper {
  animation: heartbeat 1.5s ease-in-out infinite;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 600ms ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.logo-scale-enter-active {
  transition: opacity 500ms ease-out, transform 500ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

.logo-scale-leave-active {
  transition: opacity 300ms ease-in, transform 300ms ease-in;
}

.logo-scale-enter-from {
  opacity: 0;
  transform: scale(0.6) rotate(-8deg);
}

.logo-scale-leave-to {
  opacity: 0;
  transform: scale(1.2) rotate(8deg);
}

.logo-slide-left-enter-active,
.logo-slide-right-enter-active {
  transition: opacity 400ms ease-out, transform 500ms cubic-bezier(0.25, 0.46, 0.45, 0.94);
}

.logo-slide-left-leave-active,
.logo-slide-right-leave-active {
  transition: opacity 250ms ease-in, transform 300ms ease-in;
}

.logo-slide-left-enter-from {
  opacity: 0;
  transform: translateX(3.75rem) scale(0.9);
}

.logo-slide-left-leave-to {
  opacity: 0;
  transform: translateX(-3.75rem) scale(0.9);
}

.logo-slide-right-enter-from {
  opacity: 0;
  transform: translateX(-3.75rem) scale(0.9);
}

.logo-slide-right-leave-to {
  opacity: 0;
  transform: translateX(3.75rem) scale(0.9);
}

@keyframes heartbeat {
  0%,
  70%,
  100% { transform: scale(1); }
  14% { transform: scale(1.06); }
  28% { transform: scale(1); }
  42% { transform: scale(1.1); }
}

@keyframes cursor-blink {
  0%,
  45% { opacity: 1; }
  50%,
  100% { opacity: 0; }
}

@keyframes scroll-cue {
  0%,
  100% { transform: translateY(0); }
  50% { transform: translateY(0.375rem); }
}

@media (max-width: 1100px) {
  .aether-header__inner {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .aether-nav {
    display: none;
  }

  .aether-header__docs-mobile {
    display: inline-flex;
  }

  .aether-features__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .aether-header__inner {
    width: calc(100% - 2rem);
    min-height: 3.75rem;
    gap: 0.75rem;
  }

  .aether-brand__copy small,
  .aether-locale,
  .aether-header__login,
  .aether-header__cta {
    display: none;
  }

  .logo-container {
    width: 15rem;
    height: 15rem;
  }

  .logo-container.home-section {
    width: min(16.5rem, 68vw);
    height: min(16.5rem, 68vw);
  }

  .aether-section {
    padding: 5rem 1rem 3rem;
  }

  .aether-hero__logo-space {
    height: min(38vh, 20rem);
  }

  .aether-hero h1 {
    font-size: 2.375rem;
  }

  .aether-hero p {
    font-size: 0.875rem;
    line-height: 1.7;
  }

  .aether-hero__actions,
  .aether-features__actions {
    display: grid;
    grid-template-columns: 1fr;
  }

  .aether-hero__actions > *,
  .aether-features__actions > * {
    width: 100%;
  }

  .aether-features__grid {
    grid-template-columns: 1fr;
  }

  .aether-feature-card {
    padding: 1.125rem;
  }

  .aether-hero__models {
    flex-wrap: wrap;
    row-gap: 0.375rem;
  }

  .aether-features__closing {
    margin-top: 2rem;
    padding-top: 1.5rem;
  }

  .scroll-indicator {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .aether-home {
    scroll-behavior: auto;
  }

  .aether-hero__scroll svg,
  .heartbeat-wrapper,
  .aether-hero .cursor {
    animation: none;
  }
}
</style>
