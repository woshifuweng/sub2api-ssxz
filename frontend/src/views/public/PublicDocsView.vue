<template>
  <FoundationProvider :theme="theme" class="public-docs-foundation">
    <div class="public-docs" data-testid="public-docs-page">
      <header class="public-docs-header">
        <div class="public-docs-header__inner">
          <RouterLink to="/home" class="public-docs-brand" aria-label="返回 SSXZ 首页">
            <BrandLogo variant="mark" size="2.875rem" :theme="theme" />
            <span>
              <strong>SSXZ</strong>
              <small>AI 开发工具统一接入平台</small>
            </span>
          </RouterLink>

          <nav class="public-docs-nav" aria-label="公共导航">
            <RouterLink to="/home">首页</RouterLink>
            <RouterLink to="/docs" aria-current="page">文档</RouterLink>
          </nav>

          <div class="public-docs-actions">
            <FoundationButton
              variant="ghost"
              size="icon"
              :title="themeLabel"
              :aria-label="themeLabel"
              :aria-pressed="theme === 'dark'"
              @click="toggleTheme"
            >
              <Sun v-if="theme === 'dark'" aria-hidden="true" />
              <Moon v-else aria-hidden="true" />
            </FoundationButton>
            <RouterLink to="/login" class="public-docs-login">登录</RouterLink>
            <RouterLink to="/register" class="f0-button f0-button--default public-docs-cta">
              开始使用
            </RouterLink>
          </div>
        </div>
      </header>

      <main class="public-docs-main">
        <header class="public-docs-intro">
          <span>接入文档</span>
          <h1>用 SSXZ 连接你的 AI 开发工具</h1>
          <p>按真实界面一步步完成配置。本文档公开可访问，不需要先登录。</p>
        </header>

        <div class="public-docs-layout">
          <aside class="public-docs-index" aria-label="文档目录">
            <div>
              <span>快速开始</span>
              <strong>客户端接入</strong>
            </div>
            <a href="#cc-switch-guide" aria-current="page">
              <BookOpen aria-hidden="true" />
              <span>
                <strong>CC Switch</strong>
                <small>三步一键接入</small>
              </span>
            </a>
            <RouterLink to="/login">
              <KeyRound aria-hidden="true" />
              <span>
                <strong>登录控制台</strong>
                <small>创建或管理 API Key</small>
              </span>
            </RouterLink>
          </aside>

          <FoundationCard class="public-docs-card">
            <CcSwitchGuide />
          </FoundationCard>
        </div>
      </main>

      <footer class="public-docs-footer">
        <span>© 2026 SSXZ AI</span>
        <RouterLink to="/home">返回首页</RouterLink>
      </footer>
    </div>
  </FoundationProvider>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { BookOpen, KeyRound, Moon, Sun } from '@lucide/vue'
import BrandLogo from '@/components/common/BrandLogo.vue'
import CcSwitchGuide from '@/components/docs/CcSwitchGuide.vue'
import { FoundationButton, FoundationCard, FoundationProvider } from '@/components/foundation'
import { setSafeLocalStorageItem } from '@/utils/safeStorage'

function getInitialTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'dark'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const theme = ref<'light' | 'dark'>(getInitialTheme())
const themeLabel = computed(() => theme.value === 'dark' ? '切换到亮色模式' : '切换到暗色模式')
let themeObserver: MutationObserver | null = null

function syncTheme(): void {
  theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

function toggleTheme(): void {
  const nextTheme = theme.value === 'dark' ? 'light' : 'dark'
  document.documentElement.classList.toggle('dark', nextTheme === 'dark')
  setSafeLocalStorageItem('theme', nextTheme)
  syncTheme()
}

onMounted(() => {
  syncTheme()
  themeObserver = new MutationObserver(syncTheme)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
})
</script>

<style scoped>
.public-docs-foundation {
  min-height: 100vh;
  background:
    linear-gradient(hsl(var(--border) / 0.22) 1px, transparent 1px),
    linear-gradient(90deg, hsl(var(--border) / 0.22) 1px, transparent 1px),
    hsl(var(--background));
  background-size: 40px 40px;
  color: hsl(var(--foreground));
}

.public-docs {
  min-height: 100vh;
}

.public-docs-header {
  position: sticky;
  z-index: 20;
  top: 0;
  border-bottom: 1px solid hsl(var(--border));
  background: hsl(var(--background) / 0.94);
  backdrop-filter: blur(12px);
}

.public-docs-header__inner {
  display: grid;
  width: min(100% - 2rem, 78rem);
  min-height: 4.5rem;
  grid-template-columns: minmax(14rem, 1fr) auto minmax(14rem, 1fr);
  align-items: center;
  gap: 1rem;
  margin: 0 auto;
}

.public-docs-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.7rem;
  color: hsl(var(--foreground));
  text-decoration: none;
}

.public-docs-brand > span:last-child {
  display: grid;
  min-width: 0;
}

.public-docs-brand strong {
  font-size: 0.875rem;
  line-height: 1.15rem;
}

.public-docs-brand small {
  overflow: hidden;
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.public-docs-nav,
.public-docs-actions {
  display: flex;
  align-items: center;
}

.public-docs-nav {
  gap: 0.25rem;
}

.public-docs-nav a,
.public-docs-login {
  min-height: 2.25rem;
  border-radius: var(--radius);
  padding: 0 0.75rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.8125rem;
  font-weight: 650;
  line-height: 2.25rem;
  text-decoration: none;
}

.public-docs-nav a:hover,
.public-docs-nav a:focus-visible,
.public-docs-nav a[aria-current="page"],
.public-docs-login:hover,
.public-docs-login:focus-visible {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
}

.public-docs-actions {
  justify-content: flex-end;
  gap: 0.35rem;
}

.public-docs-actions svg {
  width: 1rem;
  height: 1rem;
}

.public-docs-cta {
  text-decoration: none;
}

.public-docs-main {
  width: min(100% - 2rem, 78rem);
  margin: 0 auto;
  padding: clamp(2.5rem, 7vw, 5rem) 0 4rem;
}

.public-docs-intro {
  width: min(100%, 46rem);
  margin-bottom: 2rem;
}

.public-docs-intro > span,
.public-docs-index > div span {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 720;
}

.public-docs-intro h1 {
  max-width: 15ch;
  margin-top: 0.75rem;
  font-size: clamp(2rem, 5vw, 3.5rem);
  line-height: 1.08;
  text-wrap: balance;
}

.public-docs-intro p {
  max-width: 42rem;
  margin-top: 1rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.9375rem;
  line-height: 1.75;
}

.public-docs-layout {
  display: grid;
  grid-template-columns: minmax(12rem, 15rem) minmax(0, 1fr);
  gap: 1.5rem;
  align-items: start;
}

.public-docs-index {
  position: sticky;
  top: 6rem;
  display: grid;
  gap: 0.5rem;
}

.public-docs-index > div {
  display: grid;
  gap: 0.2rem;
  padding: 0.5rem 0.75rem 0.75rem;
}

.public-docs-index > a {
  display: grid;
  min-width: 0;
  min-height: 3.5rem;
  grid-template-columns: 1rem minmax(0, 1fr);
  align-items: center;
  gap: 0.75rem;
  border-radius: var(--radius);
  padding: 0.625rem 0.75rem;
  color: hsl(var(--muted-foreground));
  text-decoration: none;
}

.public-docs-index > a:hover,
.public-docs-index > a:focus-visible,
.public-docs-index > a[aria-current="page"] {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
}

.public-docs-index svg {
  width: 1rem;
  height: 1rem;
}

.public-docs-index a span {
  display: grid;
  min-width: 0;
}

.public-docs-index a strong {
  font-size: 0.8125rem;
  line-height: 1.25rem;
}

.public-docs-index a small {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.public-docs-card {
  min-width: 0;
}

.public-docs-card :deep(.f0-card-content) {
  padding: clamp(1rem, 3vw, 2.25rem);
}

.public-docs-footer {
  display: flex;
  width: min(100% - 2rem, 78rem);
  min-height: 4rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid hsl(var(--border));
  margin: 0 auto;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
}

.public-docs-footer a {
  color: hsl(var(--foreground));
  text-decoration: none;
}

@media (max-width: 760px) {
  .public-docs-header__inner {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .public-docs-nav,
  .public-docs-login {
    display: none;
  }

  .public-docs-layout {
    grid-template-columns: 1fr;
  }

  .public-docs-index {
    position: static;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .public-docs-index > div {
    grid-column: 1 / -1;
  }
}

@media (max-width: 480px) {
  .public-docs-header__inner,
  .public-docs-main,
  .public-docs-footer {
    width: min(100% - 1rem, 78rem);
  }

  .public-docs-brand small,
  .public-docs-cta {
    display: none;
  }

  .public-docs-main {
    padding-top: 2rem;
  }

  .public-docs-intro h1 {
    font-size: 2rem;
  }

  .public-docs-index {
    grid-template-columns: 1fr;
  }

  .public-docs-index > div {
    grid-column: auto;
  }

  .public-docs-card :deep(.f0-card-content) {
    padding: 1rem;
  }
}
</style>
