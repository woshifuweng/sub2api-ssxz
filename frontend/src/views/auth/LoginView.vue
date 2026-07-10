<template>
  <AuthLayout split>
    <template #visual>
      <div class="login-visual">
        <div class="login-visual-copy">
          <span>Secure access</span>
          <h2>回到你的 AI Gateway 控制台</h2>
          <p>密钥、用量、模型和账单都在同一个工作区里管理。</p>
        </div>

        <div class="character-stage" :class="`is-${characterState}`" aria-hidden="true">
          <div class="character character--back">
            <div class="character-head">
              <span class="character-eye character-eye--left"></span>
              <span class="character-eye character-eye--right"></span>
              <span class="character-hand character-hand--left"></span>
              <span class="character-hand character-hand--right"></span>
            </div>
            <div class="character-body"></div>
          </div>
          <div class="character character--front">
            <div class="character-head">
              <span class="character-eye character-eye--left"></span>
              <span class="character-eye character-eye--right"></span>
              <span class="character-hand character-hand--left"></span>
              <span class="character-hand character-hand--right"></span>
            </div>
            <div class="character-body"></div>
          </div>
          <div class="character-input-line">
            <span></span>
          </div>
        </div>

        <div class="login-visual-meta">
          <span>API Keys</span>
          <span>Usage</span>
          <span>Billing</span>
        </div>
      </div>
    </template>

    <div class="space-y-6">
      <!-- Title -->
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('auth.welcomeBack') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('auth.signInToAccount') }}
        </p>
      </div>

      <!-- LinuxDo Connect OAuth 登录 -->
      <LinuxDoOAuthSection v-if="linuxdoOAuthEnabled && !backendModeEnabled" :disabled="isLoading" />

      <!-- Login Form -->
      <form @submit.prevent="handleLogin" class="space-y-5">
        <!-- Email Input -->
        <div>
          <label for="email" class="input-label">
            {{ t('auth.emailLabel') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="mail" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="email"
              v-model="formData.email"
              type="email"
              required
              autofocus
              autocomplete="email"
              :disabled="isLoading"
              class="input pl-11"
              :class="{ 'input-error': errors.email }"
              :placeholder="t('auth.emailPlaceholder')"
              @focus="activeField = 'email'"
              @blur="activeField = null"
            />
          </div>
          <p v-if="errors.email" class="input-error-text">
            {{ errors.email }}
          </p>
        </div>

        <!-- Password Input -->
        <div>
          <label for="password" class="input-label">
            {{ t('auth.passwordLabel') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="lock" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="password"
              v-model="formData.password"
              :type="showPassword ? 'text' : 'password'"
              required
              autocomplete="current-password"
              :disabled="isLoading"
              class="input pl-11 pr-11"
              :class="{ 'input-error': errors.password }"
              :placeholder="t('auth.passwordPlaceholder')"
              @focus="activeField = 'password'"
              @blur="activeField = null"
            />
            <button
              type="button"
              @click="showPassword = !showPassword"
              @mousedown.prevent
              class="absolute inset-y-0 right-0 flex items-center pr-3.5 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-dark-300"
              :aria-label="showPassword ? '隐藏密码' : '显示密码'"
            >
              <Icon v-if="showPassword" name="eyeOff" size="md" />
              <Icon v-else name="eye" size="md" />
            </button>
          </div>
          <div class="mt-1 flex items-center justify-between">
            <p v-if="errors.password" class="input-error-text">
              {{ errors.password }}
            </p>
            <span v-else></span>
            <router-link
              v-if="passwordResetEnabled && !backendModeEnabled"
              to="/forgot-password"
              class="text-sm font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
            >
              {{ t('auth.forgotPassword') }}
            </router-link>
          </div>
        </div>

        <!-- Turnstile Widget -->
        <div v-if="turnstileEnabled && turnstileSiteKey">
          <TurnstileWidget
            ref="turnstileRef"
            :site-key="turnstileSiteKey"
            @verify="onTurnstileVerify"
            @expire="onTurnstileExpire"
            @error="onTurnstileError"
          />
          <p v-if="errors.turnstile" class="input-error-text mt-2 text-center">
            {{ errors.turnstile }}
          </p>
        </div>

        <!-- Error Message -->
        <transition name="fade">
          <div
            v-if="errorMessage"
            class="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
          >
            <div class="flex items-start gap-3">
              <div class="flex-shrink-0">
                <Icon name="exclamationCircle" size="md" class="text-red-500" />
              </div>
              <p class="text-sm text-red-700 dark:text-red-400">
                {{ errorMessage }}
              </p>
            </div>
          </div>
        </transition>

        <!-- Submit Button -->
        <button
          type="submit"
          :disabled="isLoading || (turnstileEnabled && !turnstileToken)"
          class="btn btn-primary w-full"
        >
          <svg
            v-if="isLoading"
            class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          <Icon v-else name="login" size="md" class="mr-2" />
          {{ isLoading ? t('auth.signingIn') : t('auth.signIn') }}
        </button>
      </form>
    </div>

    <!-- Footer -->
    <template v-if="!backendModeEnabled" #footer>
      <p class="text-gray-500 dark:text-dark-400">
        {{ t('auth.dontHaveAccount') }}
        <router-link
          :to="registerInAppLink"
          class="font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
        >
          {{ t('auth.signUp') }}
        </router-link>
      </p>
    </template>
  </AuthLayout>

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
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import TotpLoginModal from '@/components/auth/TotpLoginModal.vue'
import Icon from '@/components/icons/Icon.vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import { useAuthStore, useAppStore } from '@/stores'
import { getPublicSettings, isTotp2FARequired } from '@/api/auth'
import { resolveRouteAuthRedirect } from '@/utils/authRedirect'
import { getSafeSessionStorageItem, removeSafeSessionStorageItem } from '@/utils/safeStorage'
import type { TotpLoginResponse } from '@/types'

const { t } = useI18n()

// ==================== Router & Stores ====================

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()

// ==================== State ====================

const isLoading = ref<boolean>(false)
const errorMessage = ref<string>('')
const showPassword = ref<boolean>(false)
const activeField = ref<'email' | 'password' | null>(null)

// Public settings
const turnstileEnabled = ref<boolean>(false)
const turnstileSiteKey = ref<string>('')
const linuxdoOAuthEnabled = ref<boolean>(false)
const backendModeEnabled = ref<boolean>(false)
const passwordResetEnabled = ref<boolean>(false)

// Turnstile
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const turnstileToken = ref<string>('')

// 2FA state
const show2FAModal = ref<boolean>(false)
const totpTempToken = ref<string>('')
const totpUserEmailMasked = ref<string>('')
const totpModalRef = ref<InstanceType<typeof TotpLoginModal> | null>(null)

const formData = reactive({
  email: '',
  password: ''
})

const errors = reactive({
  email: '',
  password: '',
  turnstile: ''
})

const characterState = computed(() => {
  if (activeField.value === 'password') return showPassword.value ? 'peek' : 'cover'
  if (activeField.value === 'email' || formData.email) return 'follow'
  return 'idle'
})

const registerInAppLink = computed(() => {
  const hasReturnTarget = route.query.returnTo !== undefined || route.query.redirect !== undefined
  const returnTo = hasReturnTarget ? resolveRouteAuthRedirect(route.query) : undefined

  return {
    path: '/register',
    query: {
      ...(returnTo ? { returnTo } : {})
    }
  }
})

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
  } catch (error) {
    console.error('Failed to load public settings:', error)
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

// ==================== 2FA Handlers ====================

async function handle2FAVerify(code: string): Promise<void> {
  if (totpModalRef.value) {
    totpModalRef.value.setVerifying(true)
  }

  try {
    await authStore.login2FA(totpTempToken.value, code)

    // Close modal and show success
    show2FAModal.value = false
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

.login-visual {
  display: flex;
  min-height: 42rem;
  flex-direction: column;
  justify-content: space-between;
  padding: 3rem;
}

.login-visual-copy {
  max-width: 28rem;
}

.login-visual-copy > span {
  color: var(--ssxz-accent);
  font-family: var(--ssxz-font-mono);
  font-size: 0.7rem;
}

.login-visual-copy h2 {
  max-width: 25rem;
  margin: 0.9rem 0 0;
  color: var(--ssxz-text);
  font-size: clamp(1.8rem, 3vw, 2.7rem);
  font-weight: 620;
  line-height: 1.18;
}

.login-visual-copy p {
  max-width: 25rem;
  margin: 1rem 0 0;
  color: var(--ssxz-text-muted);
  font-size: 0.9rem;
  line-height: 1.7;
}

.character-stage {
  position: relative;
  width: min(100%, 31rem);
  height: 17rem;
  align-self: center;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: 14px;
  background: var(--ssxz-surface);
}

.character-stage::before,
.character-stage::after {
  content: "";
  position: absolute;
  background: var(--ssxz-border);
}

.character-stage::before {
  top: 0;
  bottom: 0;
  left: 50%;
  width: 1px;
}

.character-stage::after {
  right: 0;
  bottom: 3.4rem;
  left: 0;
  height: 1px;
}

.character {
  position: absolute;
  bottom: 3.4rem;
  width: 8rem;
  transition: transform 480ms cubic-bezier(0.22, 1, 0.36, 1);
}

.character--back {
  left: 16%;
  transform: translateY(1.9rem) scale(0.9);
}

.character--front {
  right: 16%;
}

.character-head {
  position: relative;
  z-index: 2;
  width: 6.3rem;
  height: 6.7rem;
  margin: 0 auto -1rem;
  border: 1px solid color-mix(in srgb, var(--ssxz-primary) 34%, var(--ssxz-border));
  border-radius: 46% 46% 42% 42%;
  background: var(--ssxz-surface-raised);
}

.character--back .character-head {
  border-color: var(--ssxz-border-strong);
  border-radius: 42% 48% 44% 46%;
  background: var(--ssxz-bg-subtle);
}

.character-body {
  width: 8rem;
  height: 5rem;
  border: 1px solid var(--ssxz-border-strong);
  border-radius: 3.5rem 3.5rem 0 0;
  background: var(--ssxz-bg-subtle);
}

.character--front .character-body {
  background: color-mix(in srgb, var(--ssxz-primary) 10%, var(--ssxz-bg-subtle));
}

.character-eye {
  position: absolute;
  top: 2.45rem;
  width: 0.48rem;
  height: 0.58rem;
  border-radius: 50%;
  background: var(--ssxz-text-secondary);
  transform: translateX(0);
  transition: transform 220ms ease, height 180ms ease;
}

.character-eye--left { left: 1.85rem; }
.character-eye--right { right: 1.85rem; }

.character-hand {
  position: absolute;
  z-index: 4;
  top: 5rem;
  width: 2.2rem;
  height: 1.1rem;
  border: 1px solid var(--ssxz-border-strong);
  border-radius: 999px;
  background: var(--ssxz-surface-raised);
  opacity: 0;
  transition:
    top 360ms cubic-bezier(0.22, 1, 0.36, 1),
    opacity 180ms ease,
    transform 360ms cubic-bezier(0.22, 1, 0.36, 1);
}

.character-hand--left {
  left: 0.55rem;
  transform: rotate(-18deg);
}

.character-hand--right {
  right: 0.55rem;
  transform: rotate(18deg);
}

.is-follow .character-eye {
  transform: translateX(0.28rem);
}

.is-follow .character--front {
  transform: translateY(-0.2rem);
}

.is-cover .character-hand {
  top: 2.25rem;
  opacity: 1;
}

.is-cover .character-hand--left {
  transform: rotate(8deg);
}

.is-cover .character-hand--right {
  transform: rotate(-8deg);
}

.is-peek .character-hand {
  top: 2.55rem;
  opacity: 1;
}

.is-peek .character-hand--left {
  transform: translateX(-0.42rem) rotate(-8deg);
}

.is-peek .character-hand--right {
  transform: translateX(0.42rem) rotate(8deg);
}

.is-peek .character-eye--right {
  height: 0.18rem;
}

.character-input-line {
  position: absolute;
  right: 1.25rem;
  bottom: 1.2rem;
  left: 1.25rem;
  height: 1rem;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-bg-subtle);
}

.character-input-line span {
  display: block;
  width: 22%;
  height: 100%;
  border-radius: inherit;
  background: color-mix(in srgb, var(--ssxz-primary) 42%, transparent);
  transition: width 420ms ease;
}

.is-follow .character-input-line span { width: 58%; }
.is-cover .character-input-line span,
.is-peek .character-input-line span { width: 82%; }

.login-visual-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.login-visual-meta span {
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  padding: 0.35rem 0.58rem;
  color: var(--ssxz-text-subtle);
  font-family: var(--ssxz-font-mono);
  font-size: 0.64rem;
}

@media (prefers-reduced-motion: reduce) {
  .character,
  .character-eye,
  .character-hand,
  .character-input-line span {
    transition: none;
  }
}
</style>
