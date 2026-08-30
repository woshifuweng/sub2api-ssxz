<template>
  <ResellerPageLayout
    :title="t('reseller.pages.invite.title')"
    :description="t('reseller.pages.invite.description')"
  >
    <div class="invite-page">
      <div v-if="loadError" class="card invite-empty">{{ loadError }}</div>
      <div v-else-if="loading" class="card invite-empty">正在加载推广信息...</div>
      <template v-else>
        <section class="card invite-grid">
          <div class="invite-copy">
            <span class="invite-label">我的邀请码</span>
            <strong class="invite-code">{{ invite.invite_code || '--' }}</strong>
            <p>用户注册时填写邀请码，或通过邀请链接访问，即可建立推广关系。</p>
            <LiquidButton
              type="button"
              size="default"
              :disabled="!invite.invite_code"
              @click="copyToClipboard(invite.invite_code, '邀请码已复制')"
            >
              <Icon name="copy" size="sm" />
              <span>复制邀请码</span>
            </LiquidButton>
          </div>

          <div class="invite-qr-wrap">
            <canvas ref="qrCanvas" class="invite-qr" aria-label="邀请链接二维码" />
            <span>扫码打开邀请链接</span>
          </div>
        </section>

        <section class="card invite-link-card">
          <div>
            <span class="invite-label">邀请链接</span>
            <p class="invite-url">{{ invite.invite_link || '--' }}</p>
          </div>
          <LiquidButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="!invite.invite_link"
            @click="copyToClipboard(invite.invite_link, '邀请链接已复制')"
          >
            <Icon name="copy" size="sm" />
            <span>复制链接</span>
          </LiquidButton>
        </section>

        <section class="invite-stats">
          <article class="card invite-stat">
            <span>已招募</span>
            <strong>{{ invite.total_recruited }}</strong>
            <small>累计通过邀请建立关系的用户</small>
          </article>
          <article class="card invite-stat">
            <span>本月新增</span>
            <strong>{{ invite.recruited_this_month }}</strong>
            <small>本月新增招募用户</small>
          </article>
        </section>
      </template>
    </div>
  </ResellerPageLayout>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import QRCode from 'qrcode'
import { useI18n } from 'vue-i18n'
import ResellerPageLayout from '@/components/reseller/ResellerPageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import resellerAPI, { type InviteResponse } from '@/api/reseller'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'

const { copyToClipboard } = useClipboard()
const { t } = useI18n()
const loading = ref(true)
const loadError = ref('')
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const invite = ref<InviteResponse>(emptyInvite())

function emptyInvite(): InviteResponse {
  return { invite_code: '', invite_link: '', total_recruited: 0, recruited_this_month: 0 }
}

async function renderQRCode(): Promise<void> {
  await nextTick()
  if (!qrCanvas.value || !invite.value.invite_link) return
  await QRCode.toCanvas(qrCanvas.value, invite.value.invite_link, {
    width: 220,
    margin: 2,
    errorCorrectionLevel: 'M'
  })
}

async function loadInvite(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    invite.value = await resellerAPI.getInvite()
    await renderQRCode()
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, '推广信息加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => void loadInvite())
</script>

<style scoped>
.invite-page { display: grid; gap: 1.25rem; }
.invite-grid { display: grid; gap: 2rem; grid-template-columns: minmax(0, 1fr) 260px; padding: 1.5rem; }
.invite-copy { display: grid; align-content: center; justify-items: start; gap: 0.7rem; }
.invite-label, .invite-copy p, .invite-qr-wrap span, .invite-stat span, .invite-stat small { color: light-dark(#6b7280, #9ca3af); font-size: 0.8rem; }
.invite-copy p { margin: 0; max-width: 34rem; }
.invite-code { color: light-dark(#111827, #f9fafb); font-size: 2.25rem; letter-spacing: 0.08em; }
.invite-qr-wrap { display: grid; justify-items: center; gap: 0.6rem; }
.invite-qr { width: 220px; height: 220px; border-radius: 0.5rem; background: white; }
.invite-link-card { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 1.25rem 1.5rem; }
.invite-url { margin-top: 0.4rem; overflow-wrap: anywhere; color: light-dark(#111827, #f9fafb); font-size: 0.9rem; }
.invite-stats { display: grid; gap: 1rem; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.invite-stat { display: grid; gap: 0.45rem; padding: 1.25rem 1.5rem; }
.invite-stat strong { color: light-dark(#111827, #f9fafb); font-size: 1.75rem; }
.invite-empty { padding: 3rem 1.5rem; color: light-dark(#6b7280, #9ca3af); text-align: center; }
@media (max-width: 767px) { .invite-grid { grid-template-columns: minmax(0, 1fr); } .invite-link-card { align-items: stretch; flex-direction: column; } }
@media (max-width: 479px) { .invite-stats { grid-template-columns: minmax(0, 1fr); } }
</style>
