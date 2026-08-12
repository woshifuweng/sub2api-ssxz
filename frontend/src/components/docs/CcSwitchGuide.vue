<template>
  <article
    id="cc-switch-guide"
    data-testid="cc-switch-guide"
    class="cc-switch-guide"
    v-html="guideHtml"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
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

const guideHtml = computed(() => {
  const source = Object.entries(guideAssets).reduce(
    (content, [assetPath, assetUrl]) => content.split(assetPath).join(assetUrl),
    guideMarkdown
  )
  return renderRichContent(source)
})
</script>

<style scoped>
.cc-switch-guide {
  min-width: 0;
  color: hsl(var(--foreground));
}

.cc-switch-guide :deep(h1),
.cc-switch-guide :deep(h2) {
  text-wrap: balance;
}

.cc-switch-guide :deep(h1) {
  margin: 0;
  font-size: clamp(1.5rem, 3vw, 2.25rem);
  line-height: 1.2;
}

.cc-switch-guide :deep(h2) {
  margin: 2.25rem 0 0.75rem;
  padding-top: 0.25rem;
  font-size: 1.125rem;
  line-height: 1.6rem;
}

.cc-switch-guide :deep(p),
.cc-switch-guide :deep(li) {
  color: hsl(var(--muted-foreground));
  font-size: 0.875rem;
  line-height: 1.75;
}

.cc-switch-guide :deep(p) {
  margin: 0.75rem 0;
}

.cc-switch-guide :deep(ol),
.cc-switch-guide :deep(ul) {
  display: grid;
  gap: 0.35rem;
  margin: 0.75rem 0;
  padding-left: 1.4rem;
}

.cc-switch-guide :deep(strong) {
  color: hsl(var(--foreground));
}

.cc-switch-guide :deep(a) {
  color: hsl(var(--foreground));
  font-weight: 650;
  text-decoration-thickness: 1px;
  text-underline-offset: 0.2rem;
}

.cc-switch-guide :deep(code) {
  border: 1px solid hsl(var(--border));
  border-radius: 0.25rem;
  padding: 0.1rem 0.35rem;
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
  font-family: var(--font-mono, "Cascadia Code", monospace);
  font-size: 0.8125rem;
}

.cc-switch-guide :deep(blockquote) {
  margin: 1rem 0;
  border-left: 3px solid hsl(var(--border));
  padding: 0.25rem 0 0.25rem 1rem;
  background: hsl(var(--muted) / 0.45);
}

.cc-switch-guide :deep(blockquote p) {
  margin: 0.4rem 0;
}

.cc-switch-guide :deep(img) {
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

@media (max-width: 520px) {
  .cc-switch-guide :deep(h1) {
    font-size: 1.5rem;
  }
}
</style>
