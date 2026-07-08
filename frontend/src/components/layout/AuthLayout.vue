<template>
  <div class="auth-shell relative isolate flex min-h-screen items-center justify-center overflow-hidden p-4">
    <div
      class="auth-shell-surface pointer-events-none absolute inset-0 z-0"
    ></div>

    <div class="pointer-events-none absolute inset-0 z-0 overflow-hidden">
      <div class="auth-shell-grid absolute inset-0"></div>
      <div class="auth-shell-rail absolute inset-y-0 left-[12vw] hidden w-px sm:block"></div>
      <div class="auth-shell-rail absolute inset-y-0 right-[12vw] hidden w-px sm:block"></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-2xl shadow-lg shadow-primary-500/30"
          >
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <h1 class="mb-2 text-3xl font-bold tracking-normal text-zinc-950 dark:text-white">
            {{ siteName }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card Container -->
      <div class="auth-card rounded-2xl p-8">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { DEFAULT_SITE_NAME, normalizeSiteSubtitle } from '@/utils/brand'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || DEFAULT_SITE_NAME)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => normalizeSiteSubtitle(appStore.cachedPublicSettings?.site_subtitle))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell-surface {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(244, 244, 245, 0.96)),
    #f4f4f5;
}

:global(.dark) .auth-shell-surface {
  background:
    linear-gradient(180deg, rgba(24, 24, 27, 0.96), rgba(9, 9, 11, 0.98)),
    #09090b;
}

.auth-shell-grid {
  background-image:
    linear-gradient(rgba(39, 39, 42, 0.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(39, 39, 42, 0.07) 1px, transparent 1px);
  background-size: 56px 56px;
  mask-image: linear-gradient(to bottom, transparent, black 15%, black 85%, transparent);
}

:global(.dark) .auth-shell-grid {
  background-image:
    linear-gradient(rgba(244, 244, 245, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(244, 244, 245, 0.06) 1px, transparent 1px);
}

.auth-shell-rail {
  background: linear-gradient(to bottom, transparent, rgba(24, 24, 27, 0.16), transparent);
}

:global(.dark) .auth-shell-rail {
  background: linear-gradient(to bottom, transparent, rgba(244, 244, 245, 0.14), transparent);
}

.auth-card {
  border: 1px solid rgba(39, 39, 42, 0.1);
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 24px 80px rgba(24, 24, 27, 0.12);
  backdrop-filter: blur(18px);
}

:global(.dark) .auth-card {
  border-color: rgba(244, 244, 245, 0.12);
  background: rgba(24, 24, 27, 0.82);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.36);
}
</style>
