<template>
  <div class="auth-form-stack">
    <header class="auth-form-heading">
      <h1>{{ t('auth.createAccount') }}</h1>
      <p>{{ t('auth.signUpToStart', { siteName }) }}</p>
    </header>

    <div
      v-if="!registrationEnabled && settingsLoaded"
      class="auth-form-status auth-form-status--warning"
      role="status"
    >
      <Icon name="exclamationCircle" size="md" aria-hidden="true" />
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
        :input-class="errors.email ? 'input-error' : undefined"
        :disabled="registrationActionDisabled"
      >
        <template #leading>
          <Icon name="mail" size="md" aria-hidden="true" />
        </template>
      </FoundationInput>

      <div class="auth-password-field">
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
          :disabled="registrationActionDisabled"
        >
          <template #leading>
            <Icon name="lock" size="md" aria-hidden="true" />
          </template>
          <template #trailing>
            <FoundationButton
              variant="ghost"
              size="icon"
              :disabled="registrationActionDisabled"
              :aria-label="showPassword ? t('auth.hidePassword') : t('auth.showPassword')"
              @mousedown.prevent
              @click="showPassword = !showPassword"
            >
              <Icon :name="showPassword ? 'eyeOff' : 'eye'" size="md" aria-hidden="true" />
            </FoundationButton>
          </template>
        </FoundationInput>
      </div>

      <div v-if="invitationCodeEnabled" class="auth-code-slot auth-code-field">
        <FoundationInput
          id="invitation_code"
          v-model="formData.invitation_code"
          type="text"
          name="invitation_code"
          autocomplete="off"
          :label="t('auth.invitationCodeLabel')"
          :placeholder="t('auth.invitationCodePlaceholder')"
          :error="
            errors.invitation_code ||
            (invitationValidation.invalid ? invitationValidation.message : '')
          "
          :disabled="registrationActionDisabled"
          @input="handleInvitationCodeInput"
        >
          <template #leading>
            <Icon
              name="key"
              size="md"
              :class="{ 'auth-icon--success': invitationValidation.valid }"
              aria-hidden="true"
            />
          </template>
          <template #trailing>
            <span v-if="invitationValidating" class="auth-spinner" aria-hidden="true" />
            <Icon
              v-else-if="invitationValidation.valid"
              name="checkCircle"
              size="md"
              class="auth-icon--success"
              aria-hidden="true"
            />
            <Icon
              v-else-if="invitationValidation.invalid || errors.invitation_code"
              name="exclamationCircle"
              size="md"
              class="auth-icon--error"
              aria-hidden="true"
            />
          </template>
        </FoundationInput>
        <Transition name="fade">
          <p v-if="invitationValidation.valid" class="auth-validation-success">
            <Icon name="checkCircle" size="sm" aria-hidden="true" />
            <span>{{ t('auth.invitationCodeValid') }}</span>
          </p>
        </Transition>
      </div>

      <div
        v-else-if="affiliateEnabled"
        class="auth-code-slot auth-code-field"
        data-testid="affiliate-invitation-field"
      >
        <FoundationInput
          id="affiliate_code"
          v-model="formData.affiliate_code"
          type="text"
          name="affiliate_code"
          autocomplete="off"
          :label="`${t('auth.invitationCodeLabel')} (${t('common.optional')})`"
          :placeholder="t('auth.invitationCodePlaceholder')"
          :disabled="registrationActionDisabled"
        >
          <template #leading>
            <Icon name="key" size="md" aria-hidden="true" />
          </template>
        </FoundationInput>
      </div>

      <div v-if="promoCodeEnabled" class="auth-code-slot auth-code-field">
        <FoundationInput
          id="promo_code"
          v-model="formData.promo_code"
          type="text"
          name="promo_code"
          autocomplete="off"
          :label="`${t('auth.promoCodeLabel')} (${t('common.optional')})`"
          :placeholder="t('auth.promoCodePlaceholder')"
          :error="promoValidation.invalid ? promoValidation.message : ''"
          :disabled="registrationActionDisabled"
          @input="handlePromoCodeInput"
        >
          <template #leading>
            <Icon
              name="gift"
              size="md"
              :class="{ 'auth-icon--success': promoValidation.valid }"
              aria-hidden="true"
            />
          </template>
          <template #trailing>
            <span v-if="promoValidating" class="auth-spinner" aria-hidden="true" />
            <Icon
              v-else-if="promoValidation.valid"
              name="checkCircle"
              size="md"
              class="auth-icon--success"
              aria-hidden="true"
            />
            <Icon
              v-else-if="promoValidation.invalid"
              name="exclamationCircle"
              size="md"
              class="auth-icon--error"
              aria-hidden="true"
            />
          </template>
        </FoundationInput>
        <Transition name="fade">
          <p v-if="promoValidation.valid" class="auth-validation-success">
            <Icon name="gift" size="sm" aria-hidden="true" />
            <span>
              {{ t('auth.promoCodeValid', { amount: promoValidation.bonusAmount?.toFixed(2) }) }}
            </span>
          </p>
        </Transition>
      </div>

      <div v-if="captchaEnabled" class="auth-turnstile-slot" data-testid="registration-turnstile">
        <div class="auth-turnstile">
          <CaptchaChallenge
            ref="captchaRef"
            :turnstile-enabled="turnstileEnabled"
            :turnstile-site-key="turnstileSiteKey"
            :tencent-enabled="tencentCaptchaEnabled"
            :tencent-app-id="tencentCaptchaAppId"
            :aliyun-enabled="aliyunCaptchaEnabled"
            :aliyun-scene-id="aliyunCaptchaSceneId"
            :aliyun-prefix="aliyunCaptchaPrefix"
            :aliyun-region="aliyunCaptchaRegion"
            @verify="onCaptchaVerify"
            @expire="onCaptchaExpire"
            @error="onCaptchaError"
          />
          <p v-if="errors.turnstile" class="auth-field-error">{{ errors.turnstile }}</p>
        </div>
      </div>

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

      <Transition name="fade">
        <div v-if="errorMessage" class="auth-form-status auth-form-status--error" role="alert">
          <Icon name="exclamationCircle" size="md" aria-hidden="true" />
          <p>{{ errorMessage }}</p>
        </div>
      </Transition>

      <FoundationButton
        type="submit"
        size="lg"
        class="auth-submit"
        :disabled="registrationActionDisabled || (turnstileEnabled && !captchaToken)"
      >
        <template #leading>
          <span v-if="isLoading" class="auth-spinner" aria-hidden="true" />
          <Icon v-else name="userPlus" size="md" aria-hidden="true" />
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

    <div v-if="showOAuthLogin" class="auth-oauth-stack">
      <div class="auth-oauth-divider">
        <span aria-hidden="true" />
        <p>{{ t('auth.oauthOrContinue') }}</p>
        <span aria-hidden="true" />
      </div>

      <EmailOAuthButtons
        :disabled="registrationActionDisabled"
        :aff-code="formData.affiliate_code"
        :github-enabled="githubOAuthEnabled"
        :google-enabled="googleOAuthEnabled"
        :show-divider="false"
        @start="handleOAuthStart"
      />
      <LinuxDoOAuthSection
        v-if="linuxdoOAuthEnabled"
        :disabled="registrationActionDisabled"
        :aff-code="formData.affiliate_code"
        :show-divider="false"
        @start="handleOAuthStart"
      />
      <WechatOAuthSection
        v-if="wechatOAuthEnabled"
        :disabled="registrationActionDisabled"
        :aff-code="formData.affiliate_code"
        :show-divider="false"
        @start="handleOAuthStart"
      />
      <OidcOAuthSection
        v-if="oidcOAuthEnabled"
        :disabled="registrationActionDisabled"
        :provider-name="oidcOAuthProviderName"
        :aff-code="formData.affiliate_code"
        :show-divider="false"
        @start="handleOAuthStart"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import EmailOAuthButtons from '@/components/auth/EmailOAuthButtons.vue'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import LoginAgreementPrompt from '@/components/auth/LoginAgreementPrompt.vue'
import OidcOAuthSection from '@/components/auth/OidcOAuthSection.vue'
import WechatOAuthSection from '@/components/auth/WechatOAuthSection.vue'
import CaptchaChallenge from '@/components/CaptchaChallenge.vue'
import { FoundationButton, FoundationInput } from '@/components/foundation'
import Icon from '@/components/icons/Icon.vue'
import { clearAuthPortalDraft, useAuthPortalDraft } from '@/composables/useAuthPortalDraft'
import { useAppStore, useAuthStore } from '@/stores'
import {
  buildOAuthLoginStartURL,
  getPublicSettings,
  isWeChatWebOAuthEnabled,
  startOAuthLogin,
  validateInvitationCode,
  validatePromoCode,
  type OAuthLoginStart
} from '@/api/auth'
import { buildAuthErrorMessage } from '@/utils/authError'
import { extractApiErrorCode, extractI18nErrorMessage } from '@/utils/apiError'
import { DEFAULT_SITE_NAME, normalizeSiteName } from '@/utils/brand'
import {
  formatRegistrationEmailSuffixWhitelistForMessage,
  isRegistrationEmailSuffixAllowed,
  normalizeRegistrationEmailSuffixWhitelist
} from '@/utils/registrationEmailPolicy'
import {
  clearAffiliateReferralCode,
  loadAffiliateReferralCode,
  resolveAffiliateReferralCode
} from '@/utils/oauthAffiliate'
import type { LoginAgreementDocument } from '@/types'

type RegisterPublicSettings = Awaited<ReturnType<typeof getPublicSettings>> & {
  github_oauth_enabled?: boolean
  google_oauth_enabled?: boolean
}

type RegisterFormData = ReturnType<typeof useAuthPortalDraft> & {
  promo_code: string
  invitation_code: string
}

const { t, locale } = useI18n()
const LOGIN_AGREEMENT_STORAGE_KEY = 'sub2api_login_agreement_consent'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()

const isLoading = ref(false)
const settingsLoaded = ref(false)
const errorMessage = ref('')
const showPassword = ref(false)

const registrationEnabled = ref(true)
const emailVerifyEnabled = ref(false)
const promoCodeEnabled = ref(true)
const invitationCodeEnabled = ref(false)
const affiliateEnabled = ref(false)
const turnstileEnabled = ref(false)
const turnstileSiteKey = ref('')
const tencentCaptchaEnabled = ref(false)
const tencentCaptchaAppId = ref('')
const aliyunCaptchaEnabled = ref(false)
const aliyunCaptchaSceneId = ref('')
const aliyunCaptchaPrefix = ref('')
const aliyunCaptchaRegion = ref('cn')
const siteName = ref(DEFAULT_SITE_NAME)
const linuxdoOAuthEnabled = ref(false)
const wechatOAuthEnabled = ref(false)
const oidcOAuthEnabled = ref(false)
const oidcOAuthProviderName = ref('OIDC')
const githubOAuthEnabled = ref(false)
const googleOAuthEnabled = ref(false)
const registrationEmailSuffixWhitelist = ref<string[]>([])
const emailDomainQuotaEnabled = ref(false)
const loginAgreementEnabled = ref(false)
const loginAgreementMode = ref<'modal' | 'checkbox' | string>('modal')
const loginAgreementUpdatedAt = ref('')
const loginAgreementRevision = ref('')
const loginAgreementDocuments = ref<LoginAgreementDocument[]>([])
const agreementAccepted = ref(false)
const showAgreementModal = ref(false)

const captchaRef = ref<InstanceType<typeof CaptchaChallenge> | null>(null)
const captchaToken = ref('')
const tencentCaptchaRandstr = ref('')
const aliyunCaptchaReady = computed(
  () =>
    aliyunCaptchaEnabled.value &&
    Boolean(aliyunCaptchaSceneId.value) &&
    Boolean(aliyunCaptchaPrefix.value)
)
const actionCaptchaEnabled = computed(
  () =>
    (tencentCaptchaEnabled.value && Boolean(tencentCaptchaAppId.value)) ||
    aliyunCaptchaReady.value
)
const captchaEnabled = computed(
  () =>
    (turnstileEnabled.value && Boolean(turnstileSiteKey.value)) || actionCaptchaEnabled.value
)

const promoValidating = ref(false)
const promoValidation = reactive({
  valid: false,
  invalid: false,
  bonusAmount: null as number | null,
  message: ''
})
let promoValidateTimeout: ReturnType<typeof setTimeout> | null = null

const invitationValidating = ref(false)
const invitationValidation = reactive({
  valid: false,
  invalid: false,
  message: ''
})
let invitationValidateTimeout: ReturnType<typeof setTimeout> | null = null

const formData = useAuthPortalDraft() as RegisterFormData
if (typeof formData.promo_code !== 'string') formData.promo_code = ''
if (typeof formData.invitation_code !== 'string') formData.invitation_code = ''

const errors = reactive({
  email: '',
  password: '',
  turnstile: '',
  invitation_code: ''
})

const validationToastMessage = computed(
  () =>
    errors.email ||
    errors.password ||
    (invitationValidation.invalid ? invitationValidation.message : '') ||
    errors.invitation_code ||
    (promoValidation.invalid ? promoValidation.message : '') ||
    errors.turnstile ||
    ''
)

const showOAuthLogin = computed(
  () =>
    linuxdoOAuthEnabled.value ||
    wechatOAuthEnabled.value ||
    oidcOAuthEnabled.value ||
    githubOAuthEnabled.value ||
    googleOAuthEnabled.value
)

const agreementGateActive = computed(
  () => loginAgreementEnabled.value && !agreementAccepted.value
)

const registrationActionDisabled = computed(
  () => isLoading.value || !settingsLoaded.value || agreementGateActive.value
)

watch(validationToastMessage, (value, previousValue) => {
  if (value && value !== previousValue) appStore.showError(value)
})

function syncAffiliateReferralCode(): string {
  const code = resolveAffiliateReferralCode(
    route.query.aff,
    route.query.aff_code,
    route.query.affiliate
  )
  if (code) formData.affiliate_code = code
  return code
}

onMounted(async () => {
  syncAffiliateReferralCode()

  try {
    const settings = (await getPublicSettings()) as RegisterPublicSettings
    registrationEnabled.value = settings.registration_enabled
    emailVerifyEnabled.value = settings.email_verify_enabled
    promoCodeEnabled.value = settings.promo_code_enabled
    invitationCodeEnabled.value = settings.invitation_code_enabled
    affiliateEnabled.value = settings.affiliate_enabled
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    tencentCaptchaEnabled.value = settings.tencent_captcha_enabled === true
    tencentCaptchaAppId.value = settings.tencent_captcha_app_id || ''
    aliyunCaptchaEnabled.value = settings.aliyun_captcha_enabled === true
    aliyunCaptchaSceneId.value = settings.aliyun_captcha_scene_id || ''
    aliyunCaptchaPrefix.value = settings.aliyun_captcha_prefix || ''
    aliyunCaptchaRegion.value = settings.aliyun_captcha_region || 'cn'
    siteName.value = normalizeSiteName(settings.site_name)
    linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled
    wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
    oidcOAuthEnabled.value = settings.oidc_oauth_enabled
    oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
    githubOAuthEnabled.value = settings.github_oauth_enabled === true
    googleOAuthEnabled.value = settings.google_oauth_enabled === true
    registrationEmailSuffixWhitelist.value = normalizeRegistrationEmailSuffixWhitelist(
      settings.registration_email_suffix_whitelist || []
    )
    emailDomainQuotaEnabled.value = settings.registration_email_domain_quota_enabled === true
    applyLoginAgreementSettings(settings)

    if (promoCodeEnabled.value) {
      const promoParam = typeof route.query.promo === 'string' ? route.query.promo : ''
      if (promoParam) {
        formData.promo_code = promoParam
        await validatePromoCodeValue(promoParam)
      }
    }
    syncAffiliateReferralCode()
  } catch (error) {
    console.error('Failed to load public settings:', error)
    loginAgreementEnabled.value = false
    agreementAccepted.value = true
  } finally {
    settingsLoaded.value = true
  }
})

watch(
  () => [route.query.aff, route.query.aff_code, route.query.affiliate],
  () => syncAffiliateReferralCode()
)

onUnmounted(() => {
  if (promoValidateTimeout) clearTimeout(promoValidateTimeout)
  if (invitationValidateTimeout) clearTimeout(invitationValidateTimeout)
})

function applyLoginAgreementSettings(settings: {
  login_agreement_enabled?: boolean
  login_agreement_mode?: string
  login_agreement_updated_at?: string
  login_agreement_revision?: string
  login_agreement_documents?: LoginAgreementDocument[]
}): void {
  const documents = Array.isArray(settings.login_agreement_documents)
    ? settings.login_agreement_documents.filter((document) => document.title?.trim())
    : []
  loginAgreementDocuments.value = documents
  loginAgreementEnabled.value = settings.login_agreement_enabled === true && documents.length > 0
  loginAgreementMode.value = settings.login_agreement_mode === 'checkbox' ? 'checkbox' : 'modal'
  loginAgreementUpdatedAt.value = settings.login_agreement_updated_at || ''
  loginAgreementRevision.value =
    settings.login_agreement_revision ||
    `${loginAgreementUpdatedAt.value}:${documents
      .map((document) => `${document.id}:${document.title}`)
      .join('|')}`

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
    return (JSON.parse(raw) as { revision?: string }).revision === revision
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
  appStore.showWarning(t('legal.loginAgreementPrompt.registerRejectedWarning'))
}

function handlePromoCodeInput(): void {
  const code = formData.promo_code.trim()
  promoValidation.valid = false
  promoValidation.invalid = false
  promoValidation.bonusAmount = null
  promoValidation.message = ''

  if (promoValidateTimeout) clearTimeout(promoValidateTimeout)
  if (!code) {
    promoValidating.value = false
    return
  }

  promoValidateTimeout = setTimeout(() => void validatePromoCodeValue(code), 500)
}

async function validatePromoCodeValue(code: string): Promise<void> {
  if (!code.trim()) return
  promoValidating.value = true
  try {
    const result = await validatePromoCode(code)
    promoValidation.valid = result.valid
    promoValidation.invalid = !result.valid
    promoValidation.bonusAmount = result.valid ? result.bonus_amount || 0 : null
    promoValidation.message = result.valid ? '' : getPromoErrorMessage(result.error_code)
  } catch (error) {
    console.error('Failed to validate promo code:', error)
    promoValidation.valid = false
    promoValidation.invalid = true
    promoValidation.bonusAmount = null
    promoValidation.message = t('auth.promoCodeInvalid')
  } finally {
    promoValidating.value = false
  }
}

function getPromoErrorMessage(errorCode?: string): string {
  switch (errorCode) {
    case 'PROMO_CODE_NOT_FOUND':
      return t('auth.promoCodeNotFound')
    case 'PROMO_CODE_EXPIRED':
      return t('auth.promoCodeExpired')
    case 'PROMO_CODE_DISABLED':
      return t('auth.promoCodeDisabled')
    case 'PROMO_CODE_MAX_USED':
      return t('auth.promoCodeMaxUsed')
    case 'PROMO_CODE_ALREADY_USED':
      return t('auth.promoCodeAlreadyUsed')
    default:
      return t('auth.promoCodeInvalid')
  }
}

function handleInvitationCodeInput(): void {
  const code = formData.invitation_code.trim()
  invitationValidation.valid = false
  invitationValidation.invalid = false
  invitationValidation.message = ''
  errors.invitation_code = ''

  if (invitationValidateTimeout) clearTimeout(invitationValidateTimeout)
  if (!code) {
    invitationValidating.value = false
    return
  }

  invitationValidateTimeout = setTimeout(() => void validateInvitationCodeValue(code), 500)
}

async function validateInvitationCodeValue(code: string): Promise<void> {
  invitationValidating.value = true
  try {
    const result = await validateInvitationCode(code)
    invitationValidation.valid = result.valid
    invitationValidation.invalid = !result.valid
    invitationValidation.message = result.valid
      ? ''
      : getInvitationErrorMessage(result.error_code)
  } catch {
    invitationValidation.valid = false
    invitationValidation.invalid = true
    invitationValidation.message = t('auth.invitationCodeInvalid')
  } finally {
    invitationValidating.value = false
  }
}

function getInvitationErrorMessage(_errorCode?: string): string {
  return t('auth.invitationCodeInvalid')
}

function onCaptchaVerify(token: string, randstr = ''): void {
  captchaToken.value = token
  tencentCaptchaRandstr.value = randstr
  errors.turnstile = ''
}

function onCaptchaExpire(): void {
  captchaToken.value = ''
  tencentCaptchaRandstr.value = ''
  errors.turnstile = t('auth.turnstileExpired')
}

function onCaptchaError(): void {
  captchaToken.value = ''
  tencentCaptchaRandstr.value = ''
  errors.turnstile = t('auth.turnstileFailed')
}

function resetCaptchaProof(): void {
  captchaRef.value?.reset()
  captchaToken.value = ''
  tencentCaptchaRandstr.value = ''
  errors.turnstile = ''
}

async function acquireActionProof(): Promise<boolean> {
  if (!actionCaptchaEnabled.value) return true
  const proof = await captchaRef.value?.verifyAction()
  if (!proof) return false
  captchaToken.value = proof.token
  tencentCaptchaRandstr.value = proof.randstr
  return true
}

function withSSXZRedirect(request: OAuthLoginStart): OAuthLoginStart {
  const redirect = request.params.redirect === '/dashboard' ? '/app/dashboard' : request.params.redirect
  return { ...request, params: { ...request.params, redirect } }
}

async function handleOAuthStart(originalRequest: OAuthLoginStart): Promise<void> {
  if (registrationActionDisabled.value) return
  const request = withSSXZRedirect(originalRequest)

  if (!actionCaptchaEnabled.value) {
    window.location.href = buildOAuthLoginStartURL(request)
    return
  }

  isLoading.value = true
  try {
    const proof = await captchaRef.value?.verifyAction()
    if (!proof) return
    const result = await startOAuthLogin(
      request,
      tencentCaptchaEnabled.value
        ? {
            tencent_captcha_ticket: proof.token,
            tencent_captcha_randstr: proof.randstr
          }
        : { turnstile_token: proof.token }
    )
    window.location.href = result.authorize_url
  } catch (error: unknown) {
    errorMessage.value = extractI18nErrorMessage(
      error,
      t,
      'auth.errors',
      t('auth.turnstileFailed')
    )
    appStore.showError(errorMessage.value)
  } finally {
    resetCaptchaProof()
    isLoading.value = false
  }
}

function validateEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

function buildEmailSuffixNotAllowedMessage(): string {
  const normalizedWhitelist = normalizeRegistrationEmailSuffixWhitelist(
    registrationEmailSuffixWhitelist.value
  )
  if (normalizedWhitelist.length === 0) return t('auth.emailSuffixNotAllowed')

  const separator = String(locale.value || '')
    .toLowerCase()
    .startsWith('zh')
    ? '、'
    : ', '
  return t('auth.emailSuffixNotAllowedWithAllowed', {
    suffixes: formatRegistrationEmailSuffixWhitelistForMessage(normalizedWhitelist, {
      separator,
      more: (count) => t('auth.emailSuffixAllowedMore', { count })
    })
  })
}

function validateForm(): boolean {
  errors.email = ''
  errors.password = ''
  errors.turnstile = ''
  errors.invitation_code = ''
  let isValid = true

  if (agreementGateActive.value) {
    appStore.showWarning(t('legal.loginAgreementPrompt.registerRequiredWarning'))
    if (loginAgreementMode.value !== 'checkbox') showAgreementModal.value = true
    return false
  }

  if (!formData.email.trim()) {
    errors.email = t('auth.emailRequired')
    isValid = false
  } else if (!validateEmail(formData.email)) {
    errors.email = t('auth.invalidEmail')
    isValid = false
  } else if (
    !emailDomainQuotaEnabled.value &&
    !isRegistrationEmailSuffixAllowed(formData.email, registrationEmailSuffixWhitelist.value)
  ) {
    errors.email = buildEmailSuffixNotAllowedMessage()
    isValid = false
  }

  if (!formData.password) {
    errors.password = t('auth.passwordRequired')
    isValid = false
  } else if (formData.password.length < 6) {
    errors.password = t('auth.passwordMinLength')
    isValid = false
  }

  if (invitationCodeEnabled.value && !formData.invitation_code.trim()) {
    errors.invitation_code = t('auth.invitationCodeRequired')
    isValid = false
  }

  if (turnstileEnabled.value && !captchaToken.value) {
    errors.turnstile = t('auth.completeVerification')
    isValid = false
  }

  return isValid
}

function clearRegisterDraft(): void {
  clearAuthPortalDraft()
  formData.promo_code = ''
  formData.invitation_code = ''
}

async function handleRegister(): Promise<void> {
  errorMessage.value = ''
  if (!validateForm()) return

  if (formData.promo_code.trim()) {
    if (promoValidating.value) {
      errorMessage.value = t('auth.promoCodeValidating')
      return
    }
    if (promoValidation.invalid) {
      errorMessage.value = t('auth.promoCodeInvalidCannotRegister')
      return
    }
  }

  if (invitationCodeEnabled.value) {
    if (invitationValidating.value) {
      errorMessage.value = t('auth.invitationCodeValidating')
      return
    }
    if (invitationValidation.invalid) {
      errorMessage.value = t('auth.invitationCodeInvalidCannotRegister')
      return
    }
    if (formData.invitation_code.trim() && !invitationValidation.valid) {
      errorMessage.value = t('auth.invitationCodeValidating')
      await validateInvitationCodeValue(formData.invitation_code.trim())
      if (!invitationValidation.valid) {
        errorMessage.value = t('auth.invitationCodeInvalidCannotRegister')
        return
      }
    }
  }

  if (!(await acquireActionProof())) return
  isLoading.value = true

  try {
    const affiliateCode = formData.affiliate_code.trim() || loadAffiliateReferralCode()
    if (affiliateCode) formData.affiliate_code = affiliateCode

    const registrationPayload = {
      email: formData.email,
      password: formData.password,
      turnstile_token:
        turnstileEnabled.value || aliyunCaptchaEnabled.value ? captchaToken.value : undefined,
      tencent_captcha_ticket: tencentCaptchaEnabled.value ? captchaToken.value : undefined,
      tencent_captcha_randstr: tencentCaptchaEnabled.value
        ? tencentCaptchaRandstr.value
        : undefined,
      promo_code: formData.promo_code || undefined,
      invitation_code: formData.invitation_code || undefined,
      affiliate_code: affiliateCode || undefined
    }

    if (emailVerifyEnabled.value) {
      sessionStorage.setItem('register_data', JSON.stringify(registrationPayload))
      clearRegisterDraft()
      await router.push('/email-verify')
      return
    }

    await authStore.register(registrationPayload)
    clearAffiliateReferralCode()
    clearRegisterDraft()
    appStore.showSuccess(t('auth.accountCreatedSuccess', { siteName: siteName.value }))
    await router.push('/app/dashboard')
  } catch (error: unknown) {
    errorMessage.value = buildRegistrationErrorMessage(error, {
      fallback: t('auth.registrationFailed')
    })
    appStore.showError(errorMessage.value)
  } finally {
    if (captchaEnabled.value) resetCaptchaProof()
    isLoading.value = false
  }
}

function buildRegistrationErrorMessage(error: unknown, options: { fallback: string }): string {
  if (extractApiErrorCode(error) === 'EMAIL_DOMAIN_REGISTRATION_LIMIT') {
    return t('auth.emailDomainRegistrationLimit')
  }
  return buildAuthErrorMessage(error, options)
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

.auth-code-field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.auth-password-field {
  min-height: 5.125rem;
}

.auth-code-slot {
  min-height: 4.75rem;
}

.auth-turnstile-slot {
  min-height: 4.0625rem;
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

.auth-validation-success {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  margin: 0;
  color: hsl(142 72% 36%);
  font-size: 0.75rem;
  line-height: 1.125rem;
}

.auth-icon--success {
  color: hsl(142 72% 36%);
}

.auth-icon--error {
  color: hsl(var(--destructive));
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
  display: inline-block;
  width: 1rem;
  height: 1rem;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 999px;
  animation: auth-spin 800ms linear infinite;
}

.auth-oauth-stack {
  display: grid;
  gap: 0.75rem;
}

.auth-oauth-divider {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.auth-oauth-divider span {
  height: 1px;
  flex: 1;
  background: hsl(var(--border));
}

.auth-oauth-divider p {
  margin: 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
}

@keyframes auth-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
