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
          <CcSwitchGuide />
        </FoundationCard>
      </div>
    </FoundationProvider>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { BookOpen, KeyRound } from '@lucide/vue'
import CcSwitchGuide from '@/components/docs/CcSwitchGuide.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import { FoundationCard, FoundationProvider } from '@/components/foundation'

function getInitialTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'dark'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

const theme = ref<'light' | 'dark'>(getInitialTheme())

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

}
</style>
