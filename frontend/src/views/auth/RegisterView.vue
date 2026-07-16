<template>
  <div class="auth-form-stack">
      <header class="auth-form-heading">
        <h1>{{ t('auth.createAccount') }}</h1>
        <p>{{ t('auth.signUpToStart', { siteName }) }}</p>
      </header>

      <LinuxDoOAuthSection v-if="linuxdoOAuthEnabled" :disabled="isLoading" />

      <div
        v-if="!registrationEnabled && settingsLoaded"
        class="auth-form-status auth-form-status--warning"
        role="status"
      >
        <CircleAlert aria-hidden="true" />
        <p>{{ t('auth.registrationDisabled') }}</p>
      </div>

      <form v-else class="auth-form" @submit.prevent="handleRegister">
        <FoundationInput
          id="email"
          v-model="formData.email"
          type="email"
          name="email"
          autocomplete="email"
          inputmode="email"
          required
          autofocus
          :label="t('auth.emailLabel')"
          :placeholder="t('auth.emailPlaceholder')"
          :error="errors.email"
          :disabled="isLoading"
        >
          <template #leading><Mail aria-hidden="true" /></template>
        </FoundationInput>

        <FoundationInput
          id="password"
          v-model="formData.password"
          :type="showPassword ? 'text' : 'password'"
          name="password"
          autocomplete="new-password"
          required
          :label="t('auth.passwordLabel')"
          :placeholder="t('auth.createPasswordPlaceholder')"
          :help="errors.password ? undefined : t('auth.passwordHint')"
          :error="errors.password"
          :disabled="isLoading"
        >
          <template #leading><LockKeyhole aria-hidden="true" /></template>
          <template #trailing>
            <FoundationButton
              variant="ghost"
              size="icon"
              :aria-label="showPassword ? t('auth.hidePassword') : t('auth.showPassword')"
              @mousedown.prevent
              @click="showPassword = !showPassword"
            >
              <EyeOff v-if="showPassword" aria-hidden="true" />
              <Eye v-else aria-hidden="true" />
            </FoundationButton>
          </template>
        </FoundationInput>

        <Transition name="auth-invite" appear>
          <div class="auth-code-field">
          <FoundationInput
            id="affiliate_code"
            v-model="formData.affiliate_code"
            type="text"
            name="affiliate_code"
            autocomplete="off"
            :label="t('auth.invitationCodeLabel')"
            :placeholder="t('auth.invitationCodePlaceholder')"
            :disabled="isLoading"
          >
            <template #leading><KeyRound aria-hidden="true" /></template>
          </FoundationInput>
          </div>
        </Transition>

        <div v-if="turnstileEnabled && turnstileSiteKey" class="auth-turnstile">
          <TurnstileWidget
            ref="turnstileRef"
            :site-key="turnstileSiteKey"
            @verify="onTurnstileVerify"
            @expire="onTurnstileExpire"
            @error="onTurnstileError"
          />
          <p v-if="errors.turnstile" class="auth-field-error">{{ errors.turnstile }}</p>
        </div>

        <Transition name="fade">
          <div v-if="errorMessage" class="auth-form-status auth-form-status--error" role="alert">
            <CircleAlert aria-hidden="true" />
            <p>{{ errorMessage }}</p>
          </div>
        </Transition>

        <FoundationButton
          type="submit"
          size="lg"
          class="auth-submit"
          :disabled="isLoading || (turnstileEnabled && !turnstileToken)"
        >
          <template #leading>
            <LoaderCircle v-if="isLoading" class="auth-spinner" aria-hidden="true" />
            <UserPlus v-else aria-hidden="true" />
          </template>
          {{
            isLoading
              ? t('auth.processing')
              : emailVerifyEnabled
                ? t('auth.continue')
                : t('auth.createAccount')
          }}
        </FoundationButton>
      </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import {
  CircleAlert,
  Eye,
  EyeOff,
  KeyRound,
  LoaderCircle,
  LockKeyhole,
  Mail,
  UserPlus
} from '@lucide/vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import { FoundationButton, FoundationInput } from '@/components/foundation'
import { useAuthStore, useAppStore } from '@/stores'
import { clearAuthPortalDraft, useAuthPortalDraft } from '@/composables/useAuthPortalDraft'
import { getPublicSettings } from '@/api/auth'
import { buildAuthErrorMessage } from '@/utils/authError'
import { DEFAULT_SITE_NAME, normalizeSiteName } from '@/utils/brand'
import {
  isRegistrationEmailSuffixAllowed,
  normalizeRegistrationEmailSuffixWhitelist
} from '@/utils/registrationEmailPolicy'

const { t, locale } = useI18n()

// ==================== Router & Stores ====================

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()

// ==================== State ====================

const isLoading = ref<boolean>(false)
const settingsLoaded = ref<boolean>(false)
const errorMessage = ref<string>('')
const showPassword = ref<boolean>(false)

// Public settings
const registrationEnabled = ref<boolean>(true)
const emailVerifyEnabled = ref<boolean>(false)
const turnstileEnabled = ref<boolean>(false)
const turnstileSiteKey = ref<string>('')
const siteName = ref<string>(DEFAULT_SITE_NAME)
const linuxdoOAuthEnabled = ref<boolean>(false)
const registrationEmailSuffixWhitelist = ref<string[]>([])

// Turnstile
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const turnstileToken = ref<string>('')

const formData = useAuthPortalDraft()

const errors = reactive({
  email: '',
  password: '',
  turnstile: ''
})

// ==================== Lifecycle ====================

onMounted(async () => {
  try {
    const settings = await getPublicSettings()
    registrationEnabled.value = settings.registration_enabled
    emailVerifyEnabled.value = settings.email_verify_enabled
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    siteName.value = normalizeSiteName(settings.site_name)
    linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled
    registrationEmailSuffixWhitelist.value = normalizeRegistrationEmailSuffixWhitelist(
      settings.registration_email_suffix_whitelist || []
    )

    const affiliateParam = (route.query.aff || route.query.affiliate) as string | undefined
    if (affiliateParam) {
      formData.affiliate_code = affiliateParam
    }
  } catch (error) {
    console.error('Failed to load public settings:', error)
  } finally {
    settingsLoaded.value = true
  }
})

// ==================== Turnstile Handlers ====================

function onTurnstileVerify(token: string): void {
  turnstileToken.value = token
  errors.turnstile = ''
}

function onTurnstileExpire(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileExpired')
}

function onTurnstileError(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileFailed')
}

// ==================== Validation ====================

function validateEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}

function buildEmailSuffixNotAllowedMessage(): string {
  const normalizedWhitelist = normalizeRegistrationEmailSuffixWhitelist(
    registrationEmailSuffixWhitelist.value
  )
  if (normalizedWhitelist.length === 0) {
    return t('auth.emailSuffixNotAllowed')
  }
  const separator = String(locale.value || '').toLowerCase().startsWith('zh') ? '、' : ', '
  return t('auth.emailSuffixNotAllowedWithAllowed', {
    suffixes: normalizedWhitelist.join(separator)
  })
}

function validateForm(): boolean {
  // Reset errors
  errors.email = ''
  errors.password = ''
  errors.turnstile = ''

  let isValid = true

  // Email validation
  if (!formData.email.trim()) {
    errors.email = t('auth.emailRequired')
    isValid = false
  } else if (!validateEmail(formData.email)) {
    errors.email = t('auth.invalidEmail')
    isValid = false
  } else if (
    !isRegistrationEmailSuffixAllowed(formData.email, registrationEmailSuffixWhitelist.value)
  ) {
    errors.email = buildEmailSuffixNotAllowedMessage()
    isValid = false
  }

  // Password validation
  if (!formData.password) {
    errors.password = t('auth.passwordRequired')
    isValid = false
  } else if (formData.password.length < 6) {
    errors.password = t('auth.passwordMinLength')
    isValid = false
  }

  // Turnstile validation
  if (turnstileEnabled.value && !turnstileToken.value) {
    errors.turnstile = t('auth.completeVerification')
    isValid = false
  }

  return isValid
}

// ==================== Form Handlers ====================

async function handleRegister(): Promise<void> {
  // Clear previous error
  errorMessage.value = ''

  // Validate form
  if (!validateForm()) {
    return
  }

  isLoading.value = true

  try {
    // If email verification is enabled, redirect to verification page
    if (emailVerifyEnabled.value) {
      // Store registration data in sessionStorage
      sessionStorage.setItem(
        'register_data',
        JSON.stringify({
          email: formData.email,
          password: formData.password,
          turnstile_token: turnstileToken.value,
          affiliate_code: formData.affiliate_code || undefined
        })
      )

      clearAuthPortalDraft()

      // Navigate to email verification page
      await router.push('/email-verify')
      return
    }

    // Otherwise, directly register
    await authStore.register({
      email: formData.email,
      password: formData.password,
      turnstile_token: turnstileEnabled.value ? turnstileToken.value : undefined,
      affiliate_code: formData.affiliate_code || undefined
    })

    clearAuthPortalDraft()

    // Show success toast
    appStore.showSuccess(t('auth.accountCreatedSuccess', { siteName: siteName.value }))

    // Redirect regular users to the operating dashboard.
    await router.push('/app/dashboard')
  } catch (error: unknown) {
    // Reset Turnstile on error
    if (turnstileRef.value) {
      turnstileRef.value.reset()
      turnstileToken.value = ''
    }

    // Handle registration error
    errorMessage.value = buildAuthErrorMessage(error, {
      fallback: t('auth.registrationFailed')
    })

    // Also show error toast
    appStore.showError(errorMessage.value)
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.auth-form-stack,
.auth-form,
.auth-code-field {
  display: grid;
}

.auth-form-stack {
  gap: 1.5rem;
}

.auth-form {
  gap: 1.25rem;
}

.auth-form-heading {
  min-height: 4rem;
}

.auth-code-field {
  gap: 0.5rem;
}

.auth-invite-enter-active,
.auth-invite-leave-active {
  overflow: hidden;
  transition:
    max-height 220ms ease,
    opacity 180ms ease,
    transform 220ms ease;
}

.auth-invite-enter-from,
.auth-invite-leave-to {
  max-height: 0;
  opacity: 0;
  transform: translateY(-0.5rem);
}

.auth-invite-enter-to,
.auth-invite-leave-from {
  max-height: 6rem;
  opacity: 1;
  transform: translateY(0);
}

.auth-form-heading h1 {
  margin: 0;
  color: hsl(var(--foreground));
  font-size: 1.75rem;
  font-weight: 680;
  line-height: 2.125rem;
  letter-spacing: 0;
}

.auth-form-heading p {
  margin: 0.5rem 0 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.875rem;
  line-height: 1.375rem;
}

.auth-turnstile {
  display: grid;
  gap: 0.5rem;
  justify-items: center;
}

.auth-field-error {
  margin: 0;
  color: hsl(var(--destructive));
  font-size: 0.75rem;
  line-height: 1.125rem;
}

.auth-form-status {
  display: flex;
  align-items: flex-start;
  gap: 0.625rem;
  border: 1px solid;
  border-radius: var(--radius);
  padding: 0.75rem;
  font-size: 0.8125rem;
  line-height: 1.25rem;
}

.auth-form-status svg {
  width: 1rem;
  height: 1rem;
  flex: 0 0 auto;
}

.auth-form-status svg {
  margin-top: 0.125rem;
}

.auth-form-status p {
  margin: 0;
}

.auth-form-status--error {
  border-color: hsl(var(--destructive) / 0.26);
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 0.08);
}

.auth-form-status--warning {
  border-color: hsl(var(--warning) / 0.3);
  color: hsl(var(--warning));
  background: hsl(var(--warning) / 0.1);
}

.auth-submit {
  width: 100%;
  transform: none !important;
}

.auth-submit:hover,
.auth-submit:active {
  transform: none !important;
}

.auth-spinner {
  animation: auth-spin 800ms linear infinite;
}

@keyframes auth-spin {
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .auth-invite-enter-active,
  .auth-invite-leave-active {
    transition: none;
  }
}
</style>
