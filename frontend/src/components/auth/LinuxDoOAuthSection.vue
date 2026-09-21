<template>
  <div class="auth-oauth-section">
    <FoundationButton variant="outline" class="auth-oauth-button" :disabled="disabled" @click="startLogin">
      <template #leading><ScanLine aria-hidden="true" /></template>
      {{ t('auth.linuxdo.signIn') }}
    </FoundationButton>

    <div v-if="showDivider" class="auth-oauth-divider">
      <span></span>
      <small>{{ t('auth.linuxdo.orContinue') }}</small>
      <span></span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ScanLine } from '@lucide/vue'
import FoundationButton from '@/components/foundation/FoundationButton.vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { OAuthLoginStart } from '@/api/auth'
import { resolveAffiliateReferralCode, storeOAuthAffiliateCode } from '@/utils/oauthAffiliate'

const props = withDefaults(defineProps<{
  disabled?: boolean
  affCode?: string
  promoCode?: string
  showDivider?: boolean
}>(), {
  showDivider: true
})
const emit = defineEmits<{
  start: [request: OAuthLoginStart]
}>()

const route = useRoute()
const { t } = useI18n()

function startLogin(): void {
  if (props.disabled) return
  const redirectTo = (route.query.redirect as string) || '/dashboard'
  storeOAuthAffiliateCode(resolveAffiliateReferralCode(props.affCode, route.query.aff, route.query.aff_code))
  const params: Record<string, string> = { redirect: redirectTo }
  const promoCode = props.promoCode?.trim()
  if (promoCode) {
    params.promo_code = promoCode
  }
  emit('start', { provider: 'linuxdo', params })
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
