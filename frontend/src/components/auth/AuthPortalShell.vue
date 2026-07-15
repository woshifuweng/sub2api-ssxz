<template>
  <FoundationProvider :theme="theme">
    <main class="auth-portal-page">
      <section class="auth-portal-frame">
        <AuthOrbitVisual class="auth-portal-visual" :theme="theme" />

        <div class="auth-portal-panel">
          <header class="auth-portal-header">
            <RouterLink to="/home" class="auth-portal-brand" :aria-label="siteName">
              <BrandLogo class="auth-portal-brand-mark" variant="mark" size="2.5rem" :theme="theme" />
              <span class="auth-portal-brand-copy">
                <strong>{{ siteName }}</strong>
                <small>{{ siteSubtitle }}</small>
              </span>
            </RouterLink>

            <FoundationButton
              variant="ghost"
              size="icon"
              :title="themeLabel"
              :aria-label="themeLabel"
              :aria-pressed="theme === 'dark'"
              data-testid="auth-theme-toggle"
              @click="toggleTheme"
            >
              <Sun v-if="theme === 'dark'" aria-hidden="true" />
              <Moon v-else aria-hidden="true" />
            </FoundationButton>
          </header>

          <div class="auth-portal-scroll">
            <nav v-if="showRegisterTab" class="auth-portal-tabs" :aria-label="t('auth.accountAccess')">
              <RouterLink
                :to="loginLink"
                :aria-current="activeTab === 'login' ? 'page' : undefined"
                data-testid="auth-tab-login"
              >
                {{ t('auth.signIn') }}
              </RouterLink>
              <RouterLink
                :to="registerLink"
                :aria-current="activeTab === 'register' ? 'page' : undefined"
                data-testid="auth-tab-register"
              >
                {{ t('auth.signUp') }}
              </RouterLink>
            </nav>

            <div class="auth-portal-content">
              <slot />
            </div>
          </div>

          <footer class="auth-portal-footer">
            &copy; {{ currentYear }} {{ siteName }}
          </footer>
        </div>
      </section>
    </main>
  </FoundationProvider>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Moon, Sun } from '@lucide/vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AuthOrbitVisual from '@/components/auth/AuthOrbitVisual.vue'
import BrandLogo from '@/components/common/BrandLogo.vue'
import { FoundationButton, FoundationProvider } from '@/components/foundation'
import { useAppStore } from '@/stores'
import { DEFAULT_SITE_NAME, normalizeSiteSubtitle } from '@/utils/brand'
import { setSafeLocalStorageItem } from '@/utils/safeStorage'

withDefaults(
  defineProps<{
    activeTab: 'login' | 'register'
    showRegisterTab?: boolean
  }>(),
  {
    showRegisterTab: true
  }
)

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const theme = ref<'light' | 'dark'>('light')
let themeObserver: MutationObserver | null = null

const siteName = computed(() => appStore.siteName || DEFAULT_SITE_NAME)
const siteSubtitle = computed(() => normalizeSiteSubtitle(appStore.cachedPublicSettings?.site_subtitle))
const currentYear = computed(() => new Date().getFullYear())
const themeLabel = computed(() => (theme.value === 'dark' ? t('nav.lightMode') : t('nav.darkMode')))

const preservedQuery = computed(() => {
  const query: Record<string, string> = {}
  for (const key of ['aff', 'affiliate', 'promo', 'returnTo', 'redirect'] as const) {
    const value = route.query[key]
    if (typeof value === 'string' && value) query[key] = value
  }
  return query
})

const loginLink = computed(() => ({ path: '/login', query: preservedQuery.value }))
const registerLink = computed(() => ({ path: '/register', query: preservedQuery.value }))

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
  void appStore.fetchPublicSettings()
})

onUnmounted(() => {
  themeObserver?.disconnect()
})
</script>

<style scoped>
.auth-portal-page {
  display: flex;
  min-height: 100svh;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: hsl(var(--background));
}

.auth-portal-frame {
  display: grid;
  width: min(74rem, 100%);
  min-height: min(46rem, calc(100svh - 3rem));
  grid-template-columns: minmax(0, 1.08fr) minmax(27rem, 0.92fr);
  overflow: hidden;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--card));
  box-shadow: 0 24px 64px hsl(var(--shadow));
}

.auth-portal-visual {
  border-right: 1px solid hsl(var(--border));
}

.auth-portal-panel {
  display: flex;
  min-width: 0;
  min-height: 46rem;
  flex-direction: column;
  background: hsl(var(--card));
}

.auth-portal-header {
  display: flex;
  min-height: 4.25rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid hsl(var(--border) / 0.72);
  padding: 0.75rem 1.5rem;
}

.auth-portal-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
  color: hsl(var(--foreground));
  text-decoration: none;
}

.auth-portal-brand-mark {
  color: hsl(var(--foreground));
}

.auth-portal-brand-copy {
  display: grid;
  min-width: 0;
}

.auth-portal-brand-copy strong,
.auth-portal-brand-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.auth-portal-brand-copy strong {
  font-size: 0.8125rem;
  font-weight: 680;
  line-height: 1.125rem;
}

.auth-portal-brand-copy small {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}

.auth-portal-scroll {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow-y: auto;
  padding: 2.5rem 3rem 2rem;
}

.auth-portal-tabs {
  display: grid;
  width: 100%;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.25rem;
  margin-bottom: 2rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  padding: 0.25rem;
  background: hsl(var(--muted));
}

.auth-portal-tabs a {
  display: inline-flex;
  min-height: 2.25rem;
  align-items: center;
  justify-content: center;
  border-radius: calc(var(--radius) - 2px);
  color: hsl(var(--muted-foreground));
  font-size: 0.8125rem;
  font-weight: 600;
  text-decoration: none;
  transition: background-color 150ms ease, box-shadow 150ms ease, color 150ms ease;
}

.auth-portal-tabs a:hover {
  color: hsl(var(--foreground));
}

.auth-portal-tabs a[aria-current='page'] {
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  box-shadow: 0 1px 2px hsl(var(--shadow));
}

.auth-portal-tabs a:focus-visible {
  outline: 2px solid hsl(var(--ring));
  outline-offset: 2px;
}

.auth-portal-content {
  width: 100%;
  max-width: 24rem;
  margin: auto;
}

.auth-portal-footer {
  min-height: 3rem;
  flex: 0 0 auto;
  border-top: 1px solid hsl(var(--border) / 0.72);
  padding: 0.9rem 1.5rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  text-align: center;
}

@media (max-width: 940px) {
  .auth-portal-page {
    align-items: stretch;
    padding: 0;
  }

  .auth-portal-frame {
    width: 100%;
    min-height: 100svh;
    grid-template-columns: 1fr;
    border: 0;
    border-radius: 0;
    box-shadow: none;
  }

  .auth-portal-visual {
    display: none;
  }

  .auth-portal-panel {
    min-height: 100svh;
  }

  .auth-portal-scroll {
    overflow: visible;
  }

  .auth-portal-content {
    margin: 0 auto;
  }
}

@media (max-width: 560px) {
  .auth-portal-header {
    min-height: 4rem;
    padding: 0.75rem 1rem;
  }

  .auth-portal-scroll {
    padding: 1.5rem 1.25rem 2rem;
  }

  .auth-portal-tabs {
    margin-bottom: 1.5rem;
  }
}
</style>
