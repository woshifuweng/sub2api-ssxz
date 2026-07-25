<template>
  <div>
    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ t('payment.paymentMethod') }}
    </label>
    <div class="grid grid-cols-2 gap-3 sm:flex">
      <button
        v-for="method in sortedMethods"
        :key="method.type"
        type="button"
        :disabled="!method.available"
        :class="[
          'payment-method relative flex h-[64px] flex-col items-center justify-center rounded-lg px-3 transition-all sm:flex-1',
          !method.available
            ? 'payment-method-disabled cursor-not-allowed opacity-50'
            : selected === method.type
              ? 'payment-method-selected'
              : 'payment-method-idle',
        ]"
        @click="method.available && emit('select', method.type)"
      >
        <span class="flex items-center gap-2">
          <img :src="methodIcon(method.type)" :alt="t(`payment.methods.${method.type}`)" class="h-7 w-7" />
          <span class="flex flex-col items-start leading-none">
            <span class="text-base font-semibold">{{ t(`payment.methods.${method.type}`) }}</span>
            <span
              v-if="method.fee_rate > 0"
              class="text-[10px] tracking-wide text-gray-500 dark:text-dark-400"
            >
              {{ t('payment.fee') }} {{ method.fee_rate }}%
            </span>
          </span>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_ORDER } from './providerConfig'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'

export interface PaymentMethodOption {
  type: string
  fee_rate: number
  available: boolean
}

const props = defineProps<{
  methods: PaymentMethodOption[]
  selected: string
}>()

const emit = defineEmits<{
  select: [type: string]
}>()

const { t } = useI18n()

const METHOD_ICONS: Record<string, string> = {
  alipay: alipayIcon,
  wxpay: wxpayIcon,
  stripe: stripeIcon,
}

const sortedMethods = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return [...props.methods].sort((a, b) => {
    const ai = order.indexOf(a.type)
    const bi = order.indexOf(b.type)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
})

function methodIcon(type: string): string {
  if (type.includes('alipay')) return METHOD_ICONS.alipay
  if (type.includes('wxpay')) return METHOD_ICONS.wxpay
  return METHOD_ICONS[type] || alipayIcon
}

</script>

<style scoped>
.payment-method {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-text-secondary);
}

.payment-method-idle:hover {
  border-color: var(--ssxz-border-strong);
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text-primary);
}

.payment-method-selected {
  border-color: var(--ssxz-action);
  background: var(--ssxz-action-soft);
  color: var(--ssxz-text-primary);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ssxz-action) 18%, transparent);
}

.payment-method-disabled {
  background: var(--ssxz-surface-muted);
}
</style>
