<template>
  <button
    type="button"
    class="material-switch"
    :class="{
      'material-switch--checked': modelValue,
      'material-switch--small': size === 'sm',
      'material-switch--destructive': variant === 'destructive'
    }"
    role="switch"
    :aria-checked="modelValue"
    :aria-label="ariaLabel || '切换设置'"
    :data-state="modelValue ? 'on' : 'off'"
    :data-size="size"
    :data-variant="variant"
    :disabled="disabled"
    @click="toggle"
  >
    <span class="material-switch__handle" aria-hidden="true">
      <Check
        v-if="modelValue"
        class="material-switch__icon"
        :stroke-width="2.5"
        data-testid="toggle-icon-on"
      />
      <X
        v-else
        class="material-switch__icon"
        :stroke-width="2.5"
        data-testid="toggle-icon-off"
      />
    </span>
  </button>
</template>

<script setup lang="ts">
import { Check, X } from '@lucide/vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    disabled?: boolean
    ariaLabel?: string
    size?: 'default' | 'sm'
    variant?: 'primary' | 'destructive'
  }>(),
  {
    disabled: false,
    ariaLabel: undefined,
    size: 'default',
    variant: 'primary'
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>

<style scoped>
.material-switch {
  --switch-accent: var(--ssxz-primary);
  --switch-handle-off: #59636f;
  --switch-icon-off: #ffffff;
  --switch-handle-on: #ffffff;
  --switch-icon-on: var(--ssxz-primary);
  --switch-width: 52px;
  --switch-height: 32px;
  --switch-handle-size: 24px;
  --switch-travel: 20px;
  --switch-icon-size: 16px;

  position: relative;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  box-sizing: border-box;
  width: var(--switch-width);
  height: var(--switch-height);
  margin: 0;
  padding: 2px;
  overflow: hidden;
  border: 2px solid var(--ssxz-text-muted);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  cursor: pointer;
  vertical-align: middle;
  transition:
    border-color 180ms ease,
    background-color 180ms ease,
    box-shadow 180ms ease;
}

.material-switch--checked {
  border-color: var(--switch-accent);
  background: var(--switch-accent);
}

.material-switch--destructive {
  --switch-accent: var(--ssxz-danger, #b42318);
  --switch-icon-on: var(--ssxz-danger, #b42318);
}

.material-switch--small {
  --switch-width: 40px;
  --switch-height: 24px;
  --switch-handle-size: 16px;
  --switch-travel: 16px;
  --switch-icon-size: 12px;
}

.material-switch__handle {
  pointer-events: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--switch-handle-size);
  height: var(--switch-handle-size);
  border-radius: 50%;
  color: var(--switch-icon-off);
  background: var(--switch-handle-off);
  box-shadow: 0 1px 2px rgb(0 0 0 / 28%);
  transform: translateX(0);
  transition:
    width 180ms ease,
    height 180ms ease,
    color 180ms ease,
    background-color 180ms ease,
    transform 220ms cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.material-switch--checked .material-switch__handle {
  color: var(--switch-icon-on);
  background: var(--switch-handle-on);
  transform: translateX(var(--switch-travel));
}

.material-switch__icon {
  width: var(--switch-icon-size);
  height: var(--switch-icon-size);
}

.material-switch:hover:not(:disabled) {
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--switch-accent) 16%, transparent);
}

.material-switch:active:not(:disabled) .material-switch__handle {
  transform: scale(1.08);
}

.material-switch--checked:active:not(:disabled) .material-switch__handle {
  transform: translateX(var(--switch-travel)) scale(1.08);
}

.material-switch:focus-visible {
  outline: 2px solid var(--switch-accent);
  outline-offset: 3px;
}

.material-switch:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

:global(.dark) .material-switch {
  --switch-handle-off: #a1a1aa;
  --switch-icon-off: #111115;
  --switch-handle-on: #111115;

  border-color: #8b8b96;
  background: #2a2a30;
}

:global(.dark) .material-switch--checked {
  border-color: var(--switch-accent);
  background: var(--switch-accent);
}

@media (prefers-reduced-motion: reduce) {
  .material-switch,
  .material-switch__handle {
    transition: none;
  }
}
</style>
