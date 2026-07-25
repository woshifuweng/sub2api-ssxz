<template>
  <div class="auth-oauth-section">
    <FoundationButton variant="outline" class="auth-oauth-button" :disabled="disabled" @click="startLogin">
      <template #leading><ScanLine aria-hidden="true" /></template>
      {{ t('auth.linuxdo.signIn') }}
    </FoundationButton>

    <div class="auth-oauth-divider">
      <span></span>
      <small>{{ t('auth.linuxdo.orContinue') }}</small>
      <span></span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ScanLine } from '@lucide/vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { FoundationButton } from '@/components/foundation'

defineProps<{
  disabled?: boolean
}>()

const route = useRoute()
const { t } = useI18n()

function startLogin(): void {
  const redirectTo = (route.query.redirect as string) || '/app/dashboard'
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
  const normalized = apiBase.replace(/\/$/, '')
  const startURL = `${normalized}/auth/oauth/linuxdo/start?redirect=${encodeURIComponent(redirectTo)}`
  window.location.href = startURL
}
</script>

<style scoped>
.auth-oauth-section {
  display: grid;
  gap: 1rem;
}

.auth-oauth-button {
  width: 100%;
}

.auth-oauth-divider {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: center;
  gap: 0.75rem;
}

.auth-oauth-divider span {
  height: 1px;
  background: hsl(var(--border));
}

.auth-oauth-divider small {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  line-height: 1rem;
}
</style>
