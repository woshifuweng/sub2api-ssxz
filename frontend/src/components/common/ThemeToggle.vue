<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { setSafeLocalStorageItem } from '@/utils/safeStorage'

const { t } = useI18n()
const isDark = ref(
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
)
const label = computed(() => (isDark.value ? t('nav.lightMode') : t('nav.darkMode')))

function syncThemeState(): void {
  isDark.value = document.documentElement.classList.contains('dark')
}

function toggle(): void {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  setSafeLocalStorageItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(syncThemeState)
</script>

<template>
  <div
    class="sky-toggle"
    :class="{ 'is-dark': isDark }"
    role="button"
    :aria-label="label"
    :aria-pressed="isDark"
    :title="label"
    :tabindex="0"
    @click="toggle"
    @keydown.enter="toggle"
    @keydown.space.prevent="toggle"
  >
    <!-- Sky / night background -->
    <div class="sky-bg">
      <!-- Clouds (light mode) -->
      <div class="cloud cloud-lg" />
      <div class="cloud cloud-sm" />
      <!-- Stars (dark mode) -->
      <span class="star star-1" />
      <span class="star star-2" />
      <span class="star star-3" />
    </div>

    <!-- Knob: sun (light) / moon (dark) -->
    <div class="knob">
      <!-- Moon crater marks -->
      <span class="crater crater-a" />
      <span class="crater crater-b" />
    </div>
  </div>
</template>

<style scoped>
/* ─── Outer shell ──────────────────────────────────────────────── */
.sky-toggle {
  position: relative;
  width: 64px;
  height: 32px;
  border-radius: 99px;
  cursor: pointer;
  outline: none;
  border: 2.5px solid rgba(255, 255, 255, 0.85);
  background: linear-gradient(160deg, #60b8e0 0%, #a8d8ea 60%, #c9e9f5 100%);
  box-shadow:
    0 3px 10px rgba(90, 160, 210, 0.45),
    inset 0 1px 4px rgba(255, 255, 255, 0.7),
    inset 0 -2px 4px rgba(80, 140, 190, 0.3);
  transition:
    background 0.45s ease,
    border-color 0.45s ease,
    box-shadow 0.45s ease;
  overflow: hidden;
  user-select: none;
}

.sky-toggle:focus-visible {
  box-shadow:
    0 3px 10px rgba(90, 160, 210, 0.45),
    0 0 0 3px rgba(96, 184, 224, 0.5);
}

/* Dark mode shell */
.sky-toggle.is-dark {
  background: linear-gradient(160deg, #0f1b35 0%, #1a2a50 60%, #243060 100%);
  border-color: rgba(140, 160, 220, 0.5);
  box-shadow:
    0 3px 10px rgba(10, 20, 60, 0.6),
    inset 0 1px 4px rgba(255, 255, 255, 0.08),
    inset 0 -2px 4px rgba(0, 0, 0, 0.4);
}

/* ─── Sky background layer ─────────────────────────────────────── */
.sky-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
  border-radius: 99px;
}

/* ─── Clouds ────────────────────────────────────────────────────── */
.cloud {
  position: absolute;
  background: rgba(255, 255, 255, 0.92);
  border-radius: 99px;
  transition: opacity 0.35s ease, transform 0.35s ease;
}

.cloud-lg {
  width: 20px;
  height: 8px;
  bottom: 7px;
  right: 9px;
  box-shadow: 0 -5px 0 3px rgba(255, 255, 255, 0.92), 7px -3px 0 1px rgba(255, 255, 255, 0.85);
}

.cloud-sm {
  width: 13px;
  height: 6px;
  top: 6px;
  right: 13px;
  box-shadow: 0 -4px 0 2px rgba(255, 255, 255, 0.9), 5px -2px 0 1px rgba(255, 255, 255, 0.8);
}

/* Hide clouds in dark mode */
.is-dark .cloud {
  opacity: 0;
  transform: translateX(8px);
}

/* ─── Stars ─────────────────────────────────────────────────────── */
.star {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.85);
  opacity: 0;
  transition: opacity 0.45s ease;
}

.is-dark .star {
  opacity: 1;
}

.star-1 { width: 2px; height: 2px; top: 7px;  right: 12px; }
.star-2 { width: 2px; height: 2px; top: 14px; right: 8px;  }
.star-3 { width: 3px; height: 3px; top: 5px;  right: 20px; }

/* ─── Knob (sun / moon) ─────────────────────────────────────────── */
.knob {
  position: absolute;
  top: 50%;
  left: 4px;
  transform: translateY(-50%);
  width: 22px;
  height: 22px;
  border-radius: 50%;
  /* Sun: bright yellow-gold */
  background: radial-gradient(circle at 35% 35%, #ffe066, #f5a623 60%, #e08000);
  box-shadow:
    0 2px 6px rgba(240, 140, 0, 0.55),
    inset 0 -2px 3px rgba(200, 90, 0, 0.25),
    inset 2px 2px 4px rgba(255, 230, 120, 0.6);
  transition:
    transform 0.42s cubic-bezier(0.34, 1.56, 0.64, 1),
    background 0.42s ease,
    box-shadow 0.42s ease;
  z-index: 3;
}

/* Dark mode: knob slides right + becomes moon */
.is-dark .knob {
  transform: translateY(-50%) translateX(32px);
  background: radial-gradient(circle at 40% 35%, #e8eaf6, #b0b8d8 65%, #8090c0);
  box-shadow:
    0 2px 8px rgba(0, 0, 20, 0.55),
    inset -5px -2px 0 rgba(70, 90, 140, 0.35);
}

/* ─── Crater marks on moon ──────────────────────────────────────── */
.crater {
  position: absolute;
  border-radius: 50%;
  background: rgba(100, 120, 170, 0.35);
  opacity: 0;
  transition: opacity 0.3s ease 0.15s;
}

.is-dark .crater { opacity: 1; }

.crater-a { width: 5px; height: 5px; top: 5px;  right: 4px; }
.crater-b { width: 3px; height: 3px; bottom: 6px; right: 8px; }
</style>
