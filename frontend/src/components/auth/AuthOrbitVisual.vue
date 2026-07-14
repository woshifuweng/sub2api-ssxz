<template>
  <section class="auth-orbit-visual">
    <div class="auth-orbit-copy">
      <span class="auth-orbit-kicker">AI API GATEWAY</span>
      <h2>
        <span>{{ t('auth.portalHeadlineLineOne') }}</span>
        <span>{{ t('auth.portalHeadlineLineTwo') }}</span>
      </h2>
      <p>{{ t('auth.portalDescription') }}</p>
    </div>

    <div class="auth-orbit-scene" aria-hidden="true">
      <span class="auth-orbit-ripple auth-orbit-ripple--one"></span>
      <span class="auth-orbit-ripple auth-orbit-ripple--two"></span>

      <div
        v-for="ring in orbitRings"
        :key="ring.id"
        :class="['auth-orbit-ring', `auth-orbit-ring--${ring.id}`, { 'auth-orbit-ring--reverse': ring.reverse }]"
        :style="ringStyle(ring.duration)"
      >
        <span
          v-for="(model, index) in ring.models"
          :key="model"
          class="auth-orbit-node"
          :style="nodeStyle(index, ring.models.length, ring.radius)"
        >
          <span class="auth-orbit-node-face">
            <ModelIcon :model="model" size="20px" />
          </span>
        </span>
      </div>

      <div class="auth-orbit-core">
        <Waypoints aria-hidden="true" />
        <span>SSXZ</span>
      </div>
    </div>

    <div class="auth-orbit-models" aria-hidden="true">
      <span>OpenAI</span>
      <span>Claude</span>
      <span>Gemini</span>
      <span>DeepSeek</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { CSSProperties } from 'vue'
import { Waypoints } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'

const { t } = useI18n()

const orbitRings = [
  {
    id: 'inner',
    duration: '18s',
    radius: '5.5rem',
    reverse: false,
    models: ['gpt-5.5', 'claude-opus-4-8', 'gemini-2.5-pro', 'grok-4', 'deepseek-chat']
  },
  {
    id: 'middle',
    duration: '27s',
    radius: '8.6rem',
    reverse: true,
    models: ['qwen-max', 'mistral-large', 'llama-4', 'kimi-k2', 'command-r', 'glm-4']
  },
  {
    id: 'outer',
    duration: '36s',
    radius: '11.8rem',
    reverse: false,
    models: ['doubao-pro', 'minimax', 'ernie-4', 'hunyuan', 'perplexity', 'jina-embeddings', 'spark-max']
  }
] as const

function ringStyle(duration: string): CSSProperties {
  return { '--orbit-duration': duration } as CSSProperties
}

function nodeStyle(index: number, total: number, radius: string): CSSProperties {
  return {
    '--orbit-angle': `${(index * 360) / total}deg`,
    '--orbit-radius': radius
  } as CSSProperties
}
</script>

<style scoped>
.auth-orbit-visual {
  display: flex;
  min-height: 46rem;
  flex-direction: column;
  justify-content: space-between;
  overflow: hidden;
  padding: 3rem;
  color: hsl(var(--foreground));
  background: hsl(var(--muted) / 0.62);
}

.auth-orbit-copy {
  position: relative;
  z-index: 2;
  max-width: 31rem;
}

.auth-orbit-kicker {
  color: hsl(var(--brand-accent));
  font-size: 0.6875rem;
  font-weight: 700;
  line-height: 1rem;
}

.auth-orbit-copy h2 {
  margin: 0.85rem 0 0;
  color: hsl(var(--foreground));
  font-size: clamp(2rem, 3vw, 2.5rem);
  font-weight: 680;
  line-height: 1.12;
  letter-spacing: 0;
}

.auth-orbit-copy h2 span {
  display: block;
}

.auth-orbit-copy p {
  max-width: 24rem;
  margin: 1rem 0 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.9375rem;
  line-height: 1.65;
}

.auth-orbit-scene {
  position: relative;
  width: 27rem;
  height: 27rem;
  align-self: center;
  flex: 0 0 auto;
}

.auth-orbit-ring,
.auth-orbit-ripple {
  position: absolute;
  top: 50%;
  left: 50%;
  border: 1px solid hsl(var(--border));
  border-radius: 50%;
  transform: translate(-50%, -50%);
}

.auth-orbit-ring {
  animation: auth-orbit-spin var(--orbit-duration) linear infinite;
}

.auth-orbit-ring--reverse {
  animation-direction: reverse;
}

.auth-orbit-ring--inner {
  width: 11rem;
  height: 11rem;
}

.auth-orbit-ring--middle {
  width: 17.2rem;
  height: 17.2rem;
}

.auth-orbit-ring--outer {
  width: 23.6rem;
  height: 23.6rem;
  border-color: hsl(var(--border) / 0.8);
}

.auth-orbit-node {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 0;
  height: 0;
  transform: rotate(var(--orbit-angle)) translateY(calc(var(--orbit-radius) * -1));
}

.auth-orbit-node-face {
  display: flex;
  width: 2.5rem;
  height: 2.5rem;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: 50%;
  color: hsl(220 10% 9%);
  background: hsl(0 0% 98%);
  box-shadow: 0 5px 16px hsl(var(--shadow));
  transform: translate(-50%, -50%) rotate(calc(var(--orbit-angle) * -1));
  animation: auth-orbit-counter-spin var(--orbit-duration) linear infinite reverse;
}

.auth-orbit-ring--reverse .auth-orbit-node-face {
  animation-direction: normal;
}

.auth-orbit-core {
  position: absolute;
  z-index: 2;
  top: 50%;
  left: 50%;
  display: grid;
  width: 5rem;
  height: 5rem;
  place-items: center;
  border: 1px solid hsl(var(--input));
  border-radius: 50%;
  color: hsl(var(--primary-foreground));
  background: hsl(var(--primary));
  box-shadow:
    0 0 0 0.75rem hsl(var(--brand-accent) / 0.08),
    0 0 2.25rem hsl(var(--brand-accent) / 0.2);
  transform: translate(-50%, -50%);
}

.auth-orbit-core svg {
  width: 1.25rem;
  height: 1.25rem;
  stroke-width: 1.7;
}

.auth-orbit-core span {
  margin-top: -0.75rem;
  font-size: 0.625rem;
  font-weight: 750;
}

.auth-orbit-ripple {
  width: 7rem;
  height: 7rem;
  border-color: hsl(var(--brand-accent) / 0.35);
  opacity: 0;
  animation: auth-orbit-ripple 5s ease-out infinite;
}

.auth-orbit-ripple--two {
  animation-delay: 2.5s;
}

.auth-orbit-models {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.auth-orbit-models span {
  border: 1px solid hsl(var(--border));
  border-radius: 999px;
  padding: 0.35rem 0.625rem;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--card) / 0.72);
  font-size: 0.6875rem;
  font-weight: 600;
}

@keyframes auth-orbit-spin {
  to { transform: translate(-50%, -50%) rotate(360deg); }
}

@keyframes auth-orbit-counter-spin {
  to { transform: translate(-50%, -50%) rotate(calc(var(--orbit-angle) * -1)) rotate(360deg); }
}

@keyframes auth-orbit-ripple {
  0% { opacity: 0.45; transform: translate(-50%, -50%) scale(0.72); }
  75%, 100% { opacity: 0; transform: translate(-50%, -50%) scale(2.35); }
}

@media (prefers-reduced-motion: reduce) {
  .auth-orbit-ring,
  .auth-orbit-node-face,
  .auth-orbit-ripple {
    animation: none;
  }

  .auth-orbit-ripple {
    display: none;
  }
}
</style>
