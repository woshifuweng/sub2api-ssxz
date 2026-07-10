<template>
  <div class="auth-shell relative isolate flex min-h-screen items-center justify-center overflow-hidden p-4">
    <div class="auth-shell-surface pointer-events-none absolute inset-0 z-0"></div>

    <div class="auth-layout relative z-10" :class="{ 'auth-layout--split': split }">
      <aside v-if="split && $slots.visual" class="auth-visual-panel">
        <slot name="visual" />
      </aside>

      <div class="auth-form-column">
        <div class="auth-brand text-center">
          <template v-if="settingsLoaded">
            <div class="auth-brand-lockup inline-flex items-center justify-center">
              <img v-if="approvedSiteLogo" :src="approvedSiteLogo" alt="SSXZ AI Gateway" class="auth-brand-image object-contain" />
              <span v-else>SSXZ AI Gateway</span>
            </div>
            <h1>{{ siteName }}</h1>
            <p class="auth-subtitle">{{ siteSubtitle }}</p>
          </template>
        </div>

        <div class="auth-card">
          <slot />
        </div>

        <div class="auth-footer text-center text-sm">
          <slot name="footer" />
        </div>

        <div class="auth-copyright text-center text-xs">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { DEFAULT_SITE_NAME, normalizeSiteSubtitle } from '@/utils/brand'
import { sanitizeUrl } from '@/utils/url'

withDefaults(defineProps<{ split?: boolean }>(), {
  split: false
})

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || DEFAULT_SITE_NAME)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const approvedSiteLogo = computed(() => {
  const value = siteLogo.value.trim()
  if (!value || /(?:^|\/)logo\.png(?:$|\?)/i.test(value)) return ''
  return value
})
const siteSubtitle = computed(() => normalizeSiteSubtitle(appStore.cachedPublicSettings?.site_subtitle))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell-surface {
  background: var(--ssxz-bg, #070b14);
}

.auth-layout {
  width: min(100%, 30rem);
}

.auth-layout--split {
  display: grid;
  width: min(100%, 70rem);
  grid-template-columns: minmax(0, 1.08fr) minmax(26rem, 0.92fr);
  overflow: hidden;
  border: 1px solid var(--ssxz-border, #1f2937);
  border-radius: 16px;
  background: var(--ssxz-surface, #111827);
  box-shadow: var(--ssxz-shadow-dialog, 0 24px 64px rgb(0 0 0 / 0.42));
}

.auth-visual-panel {
  position: relative;
  min-height: 42rem;
  overflow: hidden;
  border-right: 1px solid var(--ssxz-border, #1f2937);
  background: var(--ssxz-bg-subtle, #0d1420);
}

.auth-form-column {
  width: 100%;
}

.auth-layout:not(.auth-layout--split) .auth-form-column {
  width: min(100%, 30rem);
}

.auth-layout--split .auth-form-column {
  display: flex;
  min-height: 42rem;
  flex-direction: column;
  justify-content: center;
  padding: 2.5rem;
}

.auth-brand {
  margin-bottom: 1.75rem;
}

.auth-card {
  border: 1px solid var(--ssxz-border, #1f2937);
  background: var(--ssxz-surface, #111827);
  border-radius: var(--ssxz-radius-dialog, 12px);
  box-shadow: var(--ssxz-shadow-dialog, 0 24px 64px rgb(0 0 0 / 0.42));
  padding: 2rem;
}

.auth-layout--split .auth-card {
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  padding: 0;
}

.auth-brand-lockup {
  min-height: 2rem;
  color: var(--ssxz-text, #f1f5f9);
  font-size: 0.78rem;
  font-weight: 620;
  letter-spacing: 0.01em;
}

.auth-brand-image {
  max-width: 10rem;
  max-height: 2.5rem;
}

.auth-shell {
  color: var(--ssxz-text, #f3f4f6);
}

.auth-brand h1 {
  margin: 0.75rem 0 0;
  color: var(--ssxz-text, #f3f4f6);
  font-size: 1.35rem;
  font-weight: 620;
}

.auth-subtitle {
  margin: 0.35rem 0 0;
  font-size: 0.78rem;
}

.auth-footer {
  margin-top: 1.5rem;
}

.auth-copyright {
  margin-top: 1.75rem;
}

.auth-subtitle,
.auth-copyright {
  color: var(--ssxz-text-muted, #a1a7b3);
}

.auth-shell :deep(.text-gray-900),
.auth-shell :deep(.text-zinc-950),
.auth-shell :deep(.dark\:text-white),
.auth-shell :deep(label),
.auth-shell :deep(h1),
.auth-shell :deep(h2),
.auth-shell :deep(h3) {
  color: var(--ssxz-text, #f3f4f6);
}

.auth-shell :deep(.text-gray-500),
.auth-shell :deep(.text-gray-600),
.auth-shell :deep(.text-dark-400),
.auth-shell :deep(.dark\:text-dark-400),
.auth-shell :deep(p) {
  color: var(--ssxz-text-muted, #a1a7b3);
}

@media (max-width: 900px) {
  .auth-layout--split {
    width: min(100%, 30rem);
    grid-template-columns: 1fr;
    overflow: visible;
    border: 0;
    background: transparent;
    box-shadow: none;
  }

  .auth-layout--split .auth-visual-panel {
    display: none;
  }

  .auth-layout--split .auth-form-column {
    min-height: auto;
    padding: 0;
  }

  .auth-layout--split .auth-card {
    border: 1px solid var(--ssxz-border, #1f2937);
    border-radius: var(--ssxz-radius-dialog, 12px);
    background: var(--ssxz-surface, #111827);
    box-shadow: var(--ssxz-shadow-card, 0 10px 28px rgb(0 0 0 / 0.28));
    padding: 1.5rem;
  }
}
</style>
