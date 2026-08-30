<template>
  <div class="auth-form-stack auth-forgot-stack">
    <header class="auth-form-heading">
      <h1>{{ t('auth.forgotPasswordTitle') }}</h1>
      <p>{{ t('auth.forgotPasswordHint') }}</p>
    </header>

    <div v-if="isSubmitted" class="auth-success-panel" role="status">
      <div class="auth-success-icon"><CheckCircle2 aria-hidden="true" /></div>
      <div>
        <h2>{{ t('auth.resetEmailSent') }}</h2>
        <p>{{ t('auth.resetEmailSentHint') }}</p>
      </div>

    </div>

    <template v-if="isSubmitted">
      <RouterLink to="/login" class="auth-form-link auth-form-back-link">
        <ArrowLeft aria-hidden="true" />
        {{ t('auth.backToLogin') }}
      </RouterLink>
    </template>

    <form v-else class="auth-form" @submit.prevent="handleSubmit">
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

      <div v-if="captchaEnabled" class="auth-turnstile">
        <TurnstileWidget
          ref="turnstileRef"
          :turnstile-enabled="turnstileEnabled"
          :turnstile-site-key="turnstileSiteKey"
          :tencent-enabled="tencentCaptchaEnabled"
          :tencent-app-id="tencentCaptchaAppId"
          :tencent-region="tencentCaptchaRegion"
          :aliyun-enabled="aliyunCaptchaEnabled"
          :aliyun-scene-id="aliyunCaptchaSceneId"
          :aliyun-prefix="aliyunCaptchaPrefix"
          :aliyun-region="aliyunCaptchaRegion"
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

      <LiquidButton
        class="w-full"
        type="submit"
        size="default"
        :disabled="isLoading || (turnstileEnabled && !turnstileToken)"
      >
        <span class="flex items-center gap-2">
          <LoaderCircle v-if="isLoading" class="auth-spinner" aria-hidden="true" />
          <Mail v-else aria-hidden="true" />
          {{ isLoading ? t('auth.sendingResetLink') : t('auth.sendResetLink') }}
        </span>
      </LiquidButton>
    </form>

    <p v-if="!isSubmitted" class="auth-form-footer">
      {{ t('auth.rememberedPassword') }}
      <RouterLink to="/login" class="auth-form-link">{{ t('auth.signIn') }}</RouterLink>
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, CheckCircle2, CircleAlert, LoaderCircle, Mail } from '@lucide/vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import { FoundationInput } from '@/components/foundation'
import TurnstileWidget from '@/components/CaptchaChallenge.vue'
import { useAppStore } from '@/stores'
import { getPublicSettings, forgotPassword } from '@/api/auth'

const { t } = useI18n()
const appStore = useAppStore()

// ==================== State ====================

const isLoading = ref<boolean>(false)
const isSubmitted = ref<boolean>(false)
const errorMessage = ref<string>('')

// Public settings
const turnstileEnabled = ref<boolean>(false)
const turnstileSiteKey = ref<string>('')
const tencentCaptchaEnabled = ref<boolean>(false)
const tencentCaptchaAppId = ref<string>('')
const tencentCaptchaRegion = ref<string>('cn')
const aliyunCaptchaEnabled = ref<boolean>(false)
const aliyunCaptchaSceneId = ref<string>('')
const aliyunCaptchaPrefix = ref<string>('')
const aliyunCaptchaRegion = ref<string>('cn')

// Turnstile
const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const turnstileToken = ref<string>('')
const tencentCaptchaRandstr = ref<string>('')
const aliyunCaptchaReady = computed(
  () =>
    aliyunCaptchaEnabled.value &&
    Boolean(aliyunCaptchaSceneId.value) &&
    Boolean(aliyunCaptchaPrefix.value)
)
// 鍔ㄤ綔瑙﹀彂寮忛獙璇佺爜锛堣吘璁?闃块噷浜戯級锛氭彁浜ゆ椂寮圭獥楠岃瘉
const actionCaptchaEnabled = computed(
  () =>
    (tencentCaptchaEnabled.value && Boolean(tencentCaptchaAppId.value)) ||
    aliyunCaptchaReady.value
)
const captchaEnabled = computed(
  () =>
    (turnstileEnabled.value && Boolean(turnstileSiteKey.value)) || actionCaptchaEnabled.value
)

const formData = reactive({
  email: ''
})

const errors = reactive({
  email: '',
  turnstile: ''
})

const validationToastMessage = computed(() => errors.email || errors.turnstile || '')

watch(validationToastMessage, (value, previousValue) => {
  if (value && value !== previousValue) {
    appStore.showError(value)
  }
})

// ==================== Lifecycle ====================

onMounted(async () => {
  try {
    const settings = await getPublicSettings()
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    tencentCaptchaEnabled.value = settings.tencent_captcha_enabled === true
    tencentCaptchaAppId.value = settings.tencent_captcha_app_id || ''
    tencentCaptchaRegion.value = settings.tencent_captcha_region || 'cn'
    aliyunCaptchaEnabled.value = settings.aliyun_captcha_enabled === true
    aliyunCaptchaSceneId.value = settings.aliyun_captcha_scene_id || ''
    aliyunCaptchaPrefix.value = settings.aliyun_captcha_prefix || ''
    aliyunCaptchaRegion.value = settings.aliyun_captcha_region || 'cn'
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
})

// ==================== Turnstile Handlers ====================

function onTurnstileVerify(token: string, randstr = ''): void {
  turnstileToken.value = token
  tencentCaptchaRandstr.value = randstr
  errors.turnstile = ''
}

function onTurnstileExpire(): void {
  turnstileToken.value = ''
  tencentCaptchaRandstr.value = ''
  errors.turnstile = t('auth.turnstileExpired')
}

function onTurnstileError(): void {
  turnstileToken.value = ''
  tencentCaptchaRandstr.value = ''
  errors.turnstile = t('auth.turnstileFailed')
}

function resetCaptchaProof(): void {
  turnstileRef.value?.reset()
  turnstileToken.value = ''
  tencentCaptchaRandstr.value = ''
  errors.turnstile = ''
}

async function acquireActionProof(): Promise<boolean> {
  if (!actionCaptchaEnabled.value) return true

  const proof = await turnstileRef.value?.verifyAction()
  if (!proof) return false

  turnstileToken.value = proof.token
  tencentCaptchaRandstr.value = proof.randstr
  return true
}

// ==================== Validation ====================

function validateForm(): boolean {
  errors.email = ''
  errors.turnstile = ''
  let isValid = true

  if (!formData.email.trim()) {
    errors.email = t('auth.emailRequired')
    isValid = false
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
    errors.email = t('auth.invalidEmail')
    isValid = false
  }

  if (turnstileEnabled.value && !turnstileToken.value) {
    errors.turnstile = t('auth.completeVerification')
    isValid = false
  }

  return isValid
}

async function handleSubmit(): Promise<void> {
  errorMessage.value = ''
  if (!validateForm()) return

  if (!(await acquireActionProof())) {
    return
  }

  isLoading.value = true
  try {
    await forgotPassword({
      email: formData.email,
      turnstile_token:
        turnstileEnabled.value || aliyunCaptchaEnabled.value ? turnstileToken.value : undefined,
      tencent_captcha_ticket: tencentCaptchaEnabled.value ? turnstileToken.value : undefined,
      tencent_captcha_randstr: tencentCaptchaEnabled.value ? tencentCaptchaRandstr.value : undefined
    })
    isSubmitted.value = true
    appStore.showSuccess(t('auth.resetEmailSent'))
  } catch (error: unknown) {
    const err = error as { message?: string; response?: { data?: { detail?: string } } }
    errorMessage.value = err.response?.data?.detail || err.message || t('auth.sendResetLinkFailed')
    appStore.showError(errorMessage.value)
  } finally {
    if (captchaEnabled.value) {
      resetCaptchaProof()
    }
    isLoading.value = false
  }
}
</script>

<style scoped>
.auth-form-stack,
.auth-form {
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

.auth-form-heading h1 {
  margin: 0;
  color: hsl(var(--foreground));
  font-size: 1.75rem;
  font-weight: 680;
  line-height: 2.125rem;
}

.auth-form-heading p,
.auth-form-footer,
.auth-success-panel p {
  color: hsl(var(--muted-foreground));
  font-size: 0.875rem;
  line-height: 1.375rem;
}

.auth-form-heading p {
  margin: 0.5rem 0 0;
}

.auth-form-footer {
  margin: 0;
  text-align: center;
}

.auth-form-link {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: hsl(var(--brand-accent));
  font-size: 0.8125rem;
  font-weight: 650;
  text-decoration: none;
}

.auth-form-link:hover {
  color: hsl(var(--foreground));
}

.auth-form-link svg {
  width: 1rem;
  height: 1rem;
}

.auth-success-panel {
  display: flex;
  align-items: flex-start;
  gap: 0.875rem;
  border: 1px solid hsl(var(--success) / 0.28);
  border-radius: var(--radius);
  padding: 1rem;
  background: hsl(var(--success) / 0.08);
}

.auth-success-icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 999px;
  color: hsl(var(--success));
  background: hsl(var(--success) / 0.12);
}

.auth-success-icon svg,
.auth-form-status svg {
  width: 1rem;
  height: 1rem;
  flex: 0 0 auto;
}

.auth-success-panel h2 {
  margin: 0;
  color: hsl(var(--foreground));
  font-size: 0.95rem;
  font-weight: 680;
}

.auth-success-panel p {
  margin: 0.35rem 0 0;
}

.auth-form-back-link {
  justify-content: center;
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
}

.auth-form-status {
  display: flex;
  align-items: flex-start;
  gap: 0.625rem;
  border: 1px solid hsl(var(--destructive) / 0.26);
  border-radius: var(--radius);
  padding: 0.75rem;
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 0.08);
  font-size: 0.8125rem;
  line-height: 1.25rem;
}

.auth-form-status p {
  margin: 0;
}

.auth-spinner {
  animation: auth-spin 800ms linear infinite;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 180ms ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@keyframes auth-spin {
  to { transform: rotate(360deg); }
}
</style>
