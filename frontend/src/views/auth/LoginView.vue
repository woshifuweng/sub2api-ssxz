<template>
  <div class="auth-form-stack">
      <header class="auth-form-heading">
        <h1>{{ t('auth.welcomeBack') }}</h1>
        <p>{{ t('auth.signInToAccount') }}</p>
      </header>

      <LinuxDoOAuthSection
        v-if="linuxdoOAuthEnabled && !backendModeEnabled"
        :disabled="authActionDisabled"
      />

      <form class="auth-form" @submit.prevent="handleLogin">
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

        <div class="auth-password-field">
          <FoundationInput
            id="password"
            v-model="formData.password"
            :type="showPassword ? 'text' : 'password'"
            name="password"
            autocomplete="current-password"
            required
            :label="t('auth.passwordLabel')"
            :placeholder="t('auth.passwordPlaceholder')"
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
          <div class="auth-forgot-slot">
            <RouterLink
              to="/forgot-password"
              :class="[
                'auth-form-link auth-forgot-link',
                { 'auth-slot-hidden': !passwordResetEnabled || backendModeEnabled }
              ]"
              :aria-hidden="!passwordResetEnabled || backendModeEnabled ? 'true' : undefined"
              :tabindex="!passwordResetEnabled || backendModeEnabled ? -1 : undefined"
            >
              {{ t('auth.forgotPassword') }}
            </RouterLink>
          </div>
        </div>

        <div class="auth-code-slot auth-code-slot--placeholder" aria-hidden="true"></div>

        <div class="auth-turnstile-slot">
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
        </div>

        <Transition name="fade">
          <div v-if="errorMessage" class="auth-form-status auth-form-status--error" role="alert">
            <CircleAlert aria-hidden="true" />
            <p>{{ errorMessage }}</p>
          </div>
        </Transition>

        <LiquidButton
          type="submit"
          class="w-full"
          size="default"
          :disabled="authActionDisabled || (turnstileEnabled && !turnstileToken)"
        >
          <span class="flex items-center gap-2">
            <LoaderCircle v-if="isLoading" class="auth-spinner" aria-hidden="true" />
            <LogIn v-else aria-hidden="true" />
            {{ isLoading ? t('auth.signingIn') : t('auth.signIn') }}
          </span>
        </LiquidButton>

        <LoginAgreementPrompt
          v-if="loginAgreementEnabled"
          :accepted="agreementAccepted"
          :documents="loginAgreementDocuments"
          :mode="loginAgreementMode"
          :updated-at="loginAgreementUpdatedAt"
          :visible="showAgreementModal"
          @accept="acceptLoginAgreement"
          @reject="rejectLoginAgreement"
          @open="showAgreementModal = true"
        />

        <div v-if="showPasskeyLogin" class="space-y-3 pt-1">
          <div class="flex items-center gap-3">
            <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
            <span class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('auth.oauthOrContinue') }}
            </span>
            <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
          </div>

          <button
            v-if="showPasskeyLogin"
            type="button"
            class="btn btn-secondary w-full"
            :disabled="authActionDisabled"
            @click="handlePasskeyLogin"
          >
            <Icon name="key" size="md" class="mr-2" />
            {{ passkeyLoading ? t('auth.passkeySigningIn') : t('auth.passkeySignIn') }}
          </button>
        </div>
      </form>
  </div>

  <!-- 2FA Modal -->
  <TotpLoginModal
    v-if="show2FAModal"
    ref="totpModalRef"
    :temp-token="totpTempToken"
    :user-email-masked="totpUserEmailMasked"
    @verify="handle2FAVerify"
    @cancel="handle2FACancel"
  />
</template>

<script setup lang="ts">
import { computed, ref, reactive, onMounted } from 'vue'
import { CircleAlert, Eye, EyeOff, LoaderCircle, LockKeyhole, LogIn, Mail } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import LoginAgreementPrompt from '@/components/auth/LoginAgreementPrompt.vue'
import TotpLoginModal from '@/components/auth/TotpLoginModal.vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import Icon from '@/components/icons/Icon.vue'
import { FoundationButton, FoundationInput } from '@/components/foundation'
import LiquidButton from '@/components/common/LiquidButton.vue'
import { useAuthStore, useAppStore } from '@/stores'
import { clearAuthPortalDraft, useAuthPortalDraft } from '@/composables/useAuthPortalDraft'
import { getPublicSettings, isTotp2FARequired } from '@/api/auth'
import { resolveRouteAuthRedirect } from '@/utils/authRedirect'
import { getSafeSessionStorageItem, removeSafeSessionStorageItem } from '@/utils/safeStorage'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { LoginAgreementDocument, TotpLoginResponse } from '@/types'

const LOGIN_AGREEMENT_STORAGE_KEY = 'sub2api_login_agreement_consent'

const { t } = useI18n()

// ==================== Router & Stores ====================

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()

// ==================== State ====================

const isLoading = ref<boolean>(false)
const passkeyLoading = ref<boolean>(false)
const publicSettingsLoaded = ref<boolean>(false)
const errorMessage = ref<string>('')
const showPassword = ref<boolean>(false)

// Public settings
const turnstileEnabled = ref<boolean>(false)
const turnstileSiteKey = ref<string>('')
const linuxdoOAuthEnabled = ref<boolean>(false)
const backendModeEnabled = ref<boolean>(false)
const passwordResetEnabled = ref<boolean>(false)
const passkeyEnabled = ref<boolean>(false)
const loginAgreementEnabled = ref<boolean>(false)
const loginAgreementMode = ref<'modal' | 'checkbox' | string>('modal')
const loginAgreementUpdatedAt = ref<string>('')
const loginAgreementRevision = ref<string>('')
const loginAgreementDocuments = ref<LoginAgreementDocument[]>([])
const agreementAccepted = ref<boolean>(false)
const showAgreementModal = ref<boolean>(false)

// Turnstile
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const turnstileToken = ref<string>('')

// 2FA state
const show2FAModal = ref<boolean>(false)
const totpTempToken = ref<string>('')
const totpUserEmailMasked = ref<string>('')
const totpModalRef = ref<InstanceType<typeof TotpLoginModal> | null>(null)

const formData = useAuthPortalDraft()

const errors = reactive({
  email: '',
  password: '',
  turnstile: ''
})

const agreementGateActive = computed(
  () => loginAgreementEnabled.value && !agreementAccepted.value
)

const authActionDisabled = computed(
  () => isLoading.value || passkeyLoading.value || !publicSettingsLoaded.value || agreementGateActive.value
)

const showPasskeyLogin = computed(
  () => passkeyEnabled.value && typeof window.PublicKeyCredential !== 'undefined'
)
// ==================== Lifecycle ====================

onMounted(async () => {
  const expiredFlag = getSafeSessionStorageItem('auth_expired')
  if (expiredFlag) {
    removeSafeSessionStorageItem('auth_expired')
    const message = t('auth.reloginRequired')
    errorMessage.value = message
    appStore.showWarning(message)
  }

  try {
    const settings = await getPublicSettings()
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled
    backendModeEnabled.value = settings.backend_mode_enabled
    passwordResetEnabled.value = settings.password_reset_enabled
    passkeyEnabled.value = settings.passkey_enabled === true
    applyLoginAgreementSettings(settings)
  } catch (error) {
    console.error('Failed to load public settings:', error)
    loginAgreementEnabled.value = false
    agreementAccepted.value = true
  } finally {
    publicSettingsLoaded.value = true
  }
})

// ==================== Login Agreement ====================

function applyLoginAgreementSettings(settings: {
  login_agreement_enabled?: boolean
  login_agreement_mode?: string
  login_agreement_updated_at?: string
  login_agreement_revision?: string
  login_agreement_documents?: LoginAgreementDocument[]
}): void {
  const documents = Array.isArray(settings.login_agreement_documents)
    ? settings.login_agreement_documents.filter((doc) => doc.title?.trim())
    : []
  loginAgreementDocuments.value = documents
  loginAgreementEnabled.value = settings.login_agreement_enabled === true && documents.length > 0
  loginAgreementMode.value = settings.login_agreement_mode === 'checkbox' ? 'checkbox' : 'modal'
  loginAgreementUpdatedAt.value = settings.login_agreement_updated_at || ''
  loginAgreementRevision.value =
    settings.login_agreement_revision ||
    `${loginAgreementUpdatedAt.value}:${documents.map((doc) => `${doc.id}:${doc.title}`).join('|')}`

  agreementAccepted.value =
    !loginAgreementEnabled.value || hasAcceptedLoginAgreement(loginAgreementRevision.value)
  showAgreementModal.value =
    loginAgreementEnabled.value &&
    !agreementAccepted.value &&
    loginAgreementMode.value !== 'checkbox'
}

function hasAcceptedLoginAgreement(revision: string): boolean {
  if (!revision) return false
  try {
    const raw = localStorage.getItem(LOGIN_AGREEMENT_STORAGE_KEY)
    if (!raw) return false
    const parsed = JSON.parse(raw) as { revision?: string }
    return parsed.revision === revision
  } catch {
    return false
  }
}

function acceptLoginAgreement(): void {
  if (loginAgreementRevision.value) {
    localStorage.setItem(
      LOGIN_AGREEMENT_STORAGE_KEY,
      JSON.stringify({
        revision: loginAgreementRevision.value,
        accepted_at: new Date().toISOString()
      })
    )
  }
  agreementAccepted.value = true
  showAgreementModal.value = false
}

function rejectLoginAgreement(): void {
  localStorage.removeItem(LOGIN_AGREEMENT_STORAGE_KEY)
  agreementAccepted.value = false
  showAgreementModal.value = false
  appStore.showWarning(t('legal.loginAgreementPrompt.loginRejectedWarning'))
}

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
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
    errors.email = t('auth.invalidEmail')
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

async function handleLogin(): Promise<void> {
  // Clear previous error
  errorMessage.value = ''

  if (agreementGateActive.value) {
    appStore.showWarning(t('legal.loginAgreementPrompt.loginRequiredWarning'))
    if (loginAgreementMode.value !== 'checkbox') {
      showAgreementModal.value = true
    }
    return
  }

  // Validate form
  if (!validateForm()) {
    return
  }

  isLoading.value = true

  try {
    // Call auth store login
    const response = await authStore.login({
      email: formData.email,
      password: formData.password,
      turnstile_token: turnstileEnabled.value ? turnstileToken.value : undefined
    })

    // Check if 2FA is required
    if (isTotp2FARequired(response)) {
      const totpResponse = response as TotpLoginResponse
      totpTempToken.value = totpResponse.temp_token || ''
      totpUserEmailMasked.value = totpResponse.user_email_masked || ''
      show2FAModal.value = true
      isLoading.value = false
      return
    }

    clearAuthPortalDraft()

    // Show success toast
    appStore.showSuccess(t('auth.loginSuccess'))

    await router.push(resolveRouteAuthRedirect(route.query))
  } catch (error: unknown) {
    // Reset Turnstile on error
    if (turnstileRef.value) {
      turnstileRef.value.reset()
      turnstileToken.value = ''
    }

    // Handle login error
    const err = error as { message?: string; response?: { data?: { detail?: string } } }

    if (err.response?.data?.detail) {
      errorMessage.value = err.response.data.detail
    } else if (err.message) {
      errorMessage.value = err.message
    } else {
      errorMessage.value = t('auth.loginFailed')
    }

    // Also show error toast
    appStore.showError(errorMessage.value)
  } finally {
    isLoading.value = false
  }
}

async function handlePasskeyLogin(): Promise<void> {
  if (agreementGateActive.value) {
    appStore.showWarning(t('legal.loginAgreementPrompt.loginRequiredWarning'))
    if (loginAgreementMode.value !== 'checkbox') {
      showAgreementModal.value = true
    }
    return
  }

  passkeyLoading.value = true
  try {
    await authStore.loginWithPasskey()
    clearAuthPortalDraft()
    appStore.showSuccess(t('auth.loginSuccess'))
    await router.push(resolveRouteAuthRedirect(route.query))
  } catch (error: unknown) {
    const fallback = error instanceof DOMException && error.name === 'NotAllowedError'
      ? t('auth.passkeyCancelled')
      : t('auth.passkeyFailed')
    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', fallback)
    appStore.showError(errorMessage.value)
  } finally {
    passkeyLoading.value = false
  }
}

// ==================== 2FA Handlers ====================

async function handle2FAVerify(code: string): Promise<void> {
  if (totpModalRef.value) {
    totpModalRef.value.setVerifying(true)
  }

  try {
    await authStore.login2FA(totpTempToken.value, code)

    // Close modal and show success
    show2FAModal.value = false
    clearAuthPortalDraft()
    appStore.showSuccess(t('auth.loginSuccess'))

    await router.push(resolveRouteAuthRedirect(route.query))
  } catch (error: unknown) {
    const err = error as { message?: string; response?: { data?: { message?: string } } }
    const message = err.response?.data?.message || err.message || t('profile.totp.loginFailed')

    if (totpModalRef.value) {
      totpModalRef.value.setError(message)
      totpModalRef.value.setVerifying(false)
    }
  }
}

function handle2FACancel(): void {
  show2FAModal.value = false
  totpTempToken.value = ''
  totpUserEmailMasked.value = ''
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

.auth-form-stack {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.auth-form-heading {
  min-height: 4rem;
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

.auth-password-field {
  display: grid;
  min-height: 5.125rem;
  gap: 0.5rem;
}

.auth-forgot-slot {
  display: flex;
  min-height: 1.125rem;
  justify-content: flex-end;
}

.auth-form-link {
  color: hsl(var(--brand-accent));
  font-size: 0.75rem;
  font-weight: 650;
  text-decoration: none;
}

.auth-form-link:hover {
  color: hsl(var(--foreground));
}

.auth-forgot-link {
  align-self: flex-start;
}

.auth-slot-hidden {
  visibility: hidden;
  pointer-events: none;
}

.auth-code-slot {
  min-height: 4.75rem;
}

.auth-turnstile-slot {
  min-height: 4.0625rem;
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

.auth-spinner {
  animation: auth-spin 800ms linear infinite;
}

@keyframes auth-spin {
  to { transform: rotate(360deg); }
}
</style>
