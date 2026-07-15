<template>
  <AppSectionShell
    :class="['docs-shell', `docs-shell--${theme}`]"
    title="文档"
    subtitle="查看客户端接入步骤、真实界面截图和常见问题。"
    eyebrow="接入帮助"
    icon="book"
  >
    <FoundationProvider :theme="theme" class="docs-foundation">
      <div class="docs-layout">
        <aside class="docs-index" aria-label="文档目录">
          <div>
            <span class="docs-index__eyebrow">客户端接入</span>
            <h2>快速开始</h2>
          </div>

          <a href="#cc-switch-guide" class="docs-index__item" aria-current="page">
            <BookOpen aria-hidden="true" />
            <span>
              <strong>CC Switch</strong>
              <small>三步一键接入</small>
            </span>
          </a>

          <RouterLink to="/app/keys" class="docs-index__item docs-index__item--secondary">
            <KeyRound aria-hidden="true" />
            <span>
              <strong>API 密钥</strong>
              <small>创建或管理 Key</small>
            </span>
          </RouterLink>
        </aside>

        <FoundationCard class="docs-card">
          <article
            id="cc-switch-guide"
            data-testid="cc-switch-guide"
            class="docs-article"
            v-html="guideHtml"
          />
        </FoundationCard>
      </div>
    </FoundationProvider>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { BookOpen, KeyRound } from '@lucide/vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import { FoundationCard, FoundationProvider } from '@/components/foundation'
import { renderRichContent } from '@/utils/sanitize'
import guideMarkdown from '../../../../docs/教程/CC-Switch一键接入SSXZ.md?raw'
import downloadImage from '../../../../docs/教程/assets/cc-switch/01-official-download-windows-macos.png'
import mainWindowImage from '../../../../docs/教程/assets/cc-switch/02-cc-switch-main-window.png'
import importButtonImage from '../../../../docs/教程/assets/cc-switch/03-ssxz-import-to-ccs-button.png'
import browserPromptImage from '../../../../docs/教程/assets/cc-switch/04-browser-open-cc-switch-dialog.png'
import importConfirmationImage from '../../../../docs/教程/assets/cc-switch/05-cc-switch-import-confirmation.png'
import selectedProviderImage from '../../../../docs/教程/assets/cc-switch/06-ssxz-selected-redacted.png'
import successImage from '../../../../docs/教程/assets/cc-switch/07-claude-code-success-history.png'

const guideAssets: Record<string, string> = {
  './assets/cc-switch/01-official-download-windows-macos.png': downloadImage,
  './assets/cc-switch/02-cc-switch-main-window.png': mainWindowImage,
  './assets/cc-switch/03-ssxz-import-to-ccs-button.png': importButtonImage,
  './assets/cc-switch/04-browser-open-cc-switch-dialog.png': browserPromptImage,
  './assets/cc-switch/05-cc-switch-import-confirmation.png': importConfirmationImage,
  './assets/cc-switch/06-ssxz-selected-redacted.png': selectedProviderImage,
  './assets/cc-switch/07-claude-code-success-history.png': successImage
}

function getInitialTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'dark'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const theme = ref<'light' | 'dark'>(getInitialTheme())
const guideHtml = computed(() => {
  const source = Object.entries(guideAssets).reduce(
    (content, [assetPath, assetUrl]) => content.split(assetPath).join(assetUrl),
    guideMarkdown
  )
  return renderRichContent(source)
})

let themeObserver: MutationObserver | null = null

function syncTheme(): void {
  theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
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
.docs-foundation {
  min-height: calc(100vh - 12rem);
  background: transparent;
}

.docs-layout {
  display: grid;
  grid-template-columns: minmax(12rem, 15rem) minmax(0, 1fr);
  gap: 1.25rem;
  width: min(100%, 78rem);
  margin: 0 auto;
}

.docs-index {
  position: sticky;
  top: 5.5rem;
  display: grid;
  align-self: start;
  gap: 0.75rem;
  padding: 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--card));
  box-shadow: 0 1px 2px hsl(var(--shadow));
}

.docs-index__eyebrow {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 700;
}

.docs-index h2 {
  margin: 0.2rem 0 0;
  font-size: 1rem;
  line-height: 1.5rem;
}

.docs-index__item {
  display: flex;
  min-width: 0;
  min-height: 3.5rem;
  align-items: center;
  gap: 0.75rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  padding: 0.625rem 0.75rem;
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
  text-decoration: none;
}

.docs-index__item--secondary {
  color: hsl(var(--muted-foreground));
  background: transparent;
}

.docs-index__item:hover,
.docs-index__item:focus-visible {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
}

.docs-index__item:focus-visible {
  outline: 2px solid hsl(var(--ring));
  outline-offset: 2px;
}

.docs-index__item svg {
  width: 1rem;
  height: 1rem;
  flex: 0 0 auto;
}

.docs-index__item span {
  display: grid;
  min-width: 0;
}

.docs-index__item strong {
  font-size: 0.8125rem;
  line-height: 1.25rem;
}

.docs-index__item small {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.docs-card {
  min-width: 0;
}

.docs-card :deep(.f0-card-content) {
  padding: clamp(1rem, 3vw, 2.25rem);
}

.docs-article {
  min-width: 0;
  color: hsl(var(--foreground));
}

.docs-article :deep(h1),
.docs-article :deep(h2) {
  text-wrap: balance;
}

.docs-article :deep(h1) {
  margin: 0;
  font-size: clamp(1.5rem, 3vw, 2.25rem);
  line-height: 1.2;
}

.docs-article :deep(h2) {
  margin: 2.25rem 0 0.75rem;
  padding-top: 0.25rem;
  font-size: 1.125rem;
  line-height: 1.6rem;
}

.docs-article :deep(p),
.docs-article :deep(li) {
  color: hsl(var(--muted-foreground));
  font-size: 0.875rem;
  line-height: 1.75;
}

.docs-article :deep(p) {
  margin: 0.75rem 0;
}

.docs-article :deep(ol),
.docs-article :deep(ul) {
  display: grid;
  gap: 0.35rem;
  margin: 0.75rem 0;
  padding-left: 1.4rem;
}

.docs-article :deep(strong) {
  color: hsl(var(--foreground));
}

.docs-article :deep(a) {
  color: hsl(var(--foreground));
  font-weight: 650;
  text-decoration-thickness: 1px;
  text-underline-offset: 0.2rem;
}

.docs-article :deep(code) {
  border: 1px solid hsl(var(--border));
  border-radius: 0.25rem;
  padding: 0.1rem 0.35rem;
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
  font-family: var(--font-mono, "Cascadia Code", monospace);
  font-size: 0.8125rem;
}

.docs-article :deep(blockquote) {
  margin: 1rem 0;
  border-left: 3px solid hsl(var(--border));
  padding: 0.25rem 0 0.25rem 1rem;
  background: hsl(var(--muted) / 0.45);
}

.docs-article :deep(blockquote p) {
  margin: 0.4rem 0;
}

.docs-article :deep(img) {
  display: block;
  width: auto;
  max-width: 100%;
  max-height: 42rem;
  margin: 1rem auto 1.5rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted));
  box-shadow: 0 8px 24px hsl(var(--shadow));
  object-fit: contain;
}

@media (max-width: 840px) {
  .docs-layout {
    grid-template-columns: 1fr;
  }

  .docs-index {
    position: static;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .docs-index > div {
    grid-column: 1 / -1;
  }
}

@media (max-width: 520px) {
  .docs-index {
    grid-template-columns: 1fr;
  }

  .docs-index > div {
    grid-column: auto;
  }

  .docs-card :deep(.f0-card-content) {
    padding: 1rem;
  }

  .docs-article :deep(h1) {
    font-size: 1.5rem;
  }
}
</style>
