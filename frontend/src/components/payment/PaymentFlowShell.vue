<template>
  <main class="payment-flow-shell">
    <header class="payment-flow-header">
      <RouterLink to="/app/purchase" class="payment-flow-brand" aria-label="返回充值中心">
        <span class="payment-flow-brand-mark">S</span>
        <span>
          <strong>SSXZ AI</strong>
          <small>支付处理中</small>
        </span>
      </RouterLink>
      <RouterLink to="/app/purchase" class="payment-flow-return">
        返回充值中心
      </RouterLink>
    </header>

    <section class="payment-flow-content">
      <slot />
    </section>
  </main>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<style scoped>
.payment-flow-shell {
  min-height: 100vh;
  background:
    radial-gradient(circle at 50% 0%, rgba(20, 184, 166, 0.12), transparent 34rem),
    #f7f4ee;
  color: #111827;
}

:root.dark .payment-flow-shell {
  background:
    radial-gradient(circle at 50% 0%, rgba(20, 184, 166, 0.16), transparent 34rem),
    #0f1110;
  color: #f8fafc;
}

.payment-flow-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  padding: 1rem clamp(1rem, 4vw, 2rem);
}

:root.dark .payment-flow-header {
  border-color: rgba(255, 255, 255, 0.08);
}

.payment-flow-brand,
.payment-flow-return {
  display: inline-flex;
  align-items: center;
  text-decoration: none;
}

.payment-flow-brand {
  gap: 0.75rem;
  color: inherit;
}

.payment-flow-brand-mark {
  display: grid;
  width: 2rem;
  height: 2rem;
  place-items: center;
  border-radius: 0.7rem;
  background: #111827;
  color: #ffffff;
  font-weight: 800;
}

:root.dark .payment-flow-brand-mark {
  background: #f8fafc;
  color: #0f1110;
}

.payment-flow-brand strong,
.payment-flow-brand small {
  display: block;
  line-height: 1.15;
}

.payment-flow-brand small {
  margin-top: 0.15rem;
  color: #64748b;
  font-size: 0.75rem;
}

:root.dark .payment-flow-brand small {
  color: #94a3b8;
}

.payment-flow-return {
  min-height: 2.25rem;
  justify-content: center;
  border: 1px solid rgba(15, 23, 42, 0.12);
  border-radius: 999px;
  color: #0f766e;
  font-size: 0.85rem;
  font-weight: 700;
  padding: 0 0.9rem;
}

:root.dark .payment-flow-return {
  border-color: rgba(255, 255, 255, 0.12);
  color: #5eead4;
}

.payment-flow-content {
  display: flex;
  min-height: calc(100vh - 4.1rem);
  align-items: center;
  justify-content: center;
  padding: clamp(1rem, 4vw, 2rem);
}
</style>
