<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed right-4 top-4 z-[9999] space-y-3"
      aria-live="polite"
      aria-atomic="true"
    >
      <TransitionGroup
        enter-active-class="transition ease-out duration-300"
        enter-from-class="opacity-0 translate-x-full"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition ease-in duration-200"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 translate-x-full"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="[
            'ssxz-toast pointer-events-auto min-w-[320px] max-w-md overflow-hidden',
            getToastClass(toast.type)
          ]"
        >
          <div class="p-4">
            <div class="flex items-start gap-3">
              <!-- Icon -->
              <div class="mt-0.5 flex-shrink-0">
                <Icon
                  :name="getToastIconName(toast.type)"
                  size="md"
                  :class="getIconColor(toast.type)"
                  aria-hidden="true"
                />
              </div>

              <!-- Content -->
              <div class="min-w-0 flex-1">
                <p v-if="toast.title" class="ssxz-toast-title text-sm font-semibold">
                  {{ toast.title }}
                </p>
                <p v-if="toast.subtitle" class="ssxz-toast-subtitle mt-1 text-xs">
                  {{ toast.subtitle }}
                </p>
                <p
                  :class="[
                    'text-sm leading-relaxed',
                    toast.title || toast.subtitle
                      ? 'ssxz-toast-message mt-1'
                      : 'ssxz-toast-message'
                  ]"
                >
                  {{ toast.message }}
                </p>
              </div>

              <!-- Close button -->
              <button
                @click="removeToast(toast.id)"
                class="ssxz-toast-close -m-1 flex-shrink-0 rounded p-1 transition-colors"
                aria-label="Close notification"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>

          <!-- Progress bar -->
          <div v-if="typeof toast.progress === 'number'" class="ssxz-toast-track h-px">
            <div
              :class="['h-full transition-all duration-300', getProgressBarColor(toast.type)]"
              :style="{ width: `${Math.min(100, Math.max(0, toast.progress))}%` }"
            ></div>
          </div>
          <div v-else-if="toast.duration" class="ssxz-toast-track h-px">
            <div
              :class="['h-full toast-progress', getProgressBarColor(toast.type)]"
              :style="{ animationDuration: `${toast.duration}ms` }"
            ></div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const toasts = computed(() => appStore.toasts)

const getToastIconName = (type: string): 'checkCircle' | 'xCircle' | 'exclamationTriangle' | 'infoCircle' => {
  switch (type) {
    case 'success':
      return 'checkCircle'
    case 'error':
      return 'xCircle'
    case 'warning':
      return 'exclamationTriangle'
    case 'info':
    default:
      return 'infoCircle'
  }
}

const getIconColor = (type: string): string => {
  const colors: Record<string, string> = {
    success: 'text-green-500',
    error: 'text-red-500',
    warning: 'text-yellow-500',
    info: 'text-sky-400'
  }
  return colors[type] || colors.info
}

const getToastClass = (type: string): string => {
  const colors: Record<string, string> = {
    success: 'ssxz-toast-success',
    error: 'ssxz-toast-error',
    warning: 'ssxz-toast-warning',
    info: 'ssxz-toast-info'
  }
  return colors[type] || colors.info
}

const getProgressBarColor = (type: string): string => {
  const colors: Record<string, string> = {
    success: 'bg-green-500',
    error: 'bg-red-500',
    warning: 'bg-yellow-500',
    info: 'bg-sky-400'
  }
  return colors[type] || colors.info
}

const removeToast = (id: string) => {
  appStore.hideToast(id)
}
</script>

<style scoped>
.ssxz-toast {
  border: 1px solid var(--ssxz-border-strong);
  border-radius: var(--ssxz-radius-card);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 96%, transparent);
  box-shadow: var(--ssxz-shadow);
  color: var(--ssxz-text);
  backdrop-filter: blur(14px);
}

.ssxz-toast-success { border-color: color-mix(in srgb, var(--ssxz-success) 38%, var(--ssxz-border)); }
.ssxz-toast-error { border-color: color-mix(in srgb, var(--ssxz-error) 42%, var(--ssxz-border)); }
.ssxz-toast-warning { border-color: color-mix(in srgb, var(--ssxz-warning) 40%, var(--ssxz-border)); }
.ssxz-toast-info { border-color: color-mix(in srgb, var(--ssxz-accent) 34%, var(--ssxz-border)); }

.ssxz-toast-title,
.ssxz-toast-message { color: var(--ssxz-text); }
.ssxz-toast-subtitle { color: var(--ssxz-text-muted); }
.ssxz-toast-close { color: var(--ssxz-text-subtle); }
.ssxz-toast-close:hover { background: var(--ssxz-surface-muted); color: var(--ssxz-text); }
.ssxz-toast-track { background: var(--ssxz-surface-muted); }

.toast-progress {
  width: 100%;
  animation-name: toast-progress-shrink;
  animation-timing-function: linear;
  animation-fill-mode: forwards;
}

@keyframes toast-progress-shrink {
  from {
    width: 100%;
  }
  to {
    width: 0%;
  }
}
</style>
