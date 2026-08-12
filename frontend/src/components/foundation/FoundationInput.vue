<template>
  <div class="f0-input-field">
    <label v-if="label" class="f0-input-label" :for="resolvedId">{{ label }}</label>
    <div class="f0-input-shell">
      <span v-if="$slots.leading" class="f0-input-affix f0-input-affix--leading">
        <slot name="leading" />
      </span>
      <input
        :id="resolvedId"
        :class="[
          'f0-input-control',
          { 'f0-input-control--leading': $slots.leading, 'f0-input-control--trailing': $slots.trailing }
        ]"
        :type="type"
        :name="name"
        :value="modelValue"
        :placeholder="placeholder"
        :autocomplete="autocomplete"
        :inputmode="inputmode"
        :required="required"
        :autofocus="autofocus"
        :disabled="disabled"
        :aria-describedby="descriptionId"
        :aria-invalid="error ? 'true' : undefined"
        @input="handleInput"
        @focus="emit('focus', $event)"
        @blur="emit('blur', $event)"
      />
      <span v-if="$slots.trailing" class="f0-input-affix f0-input-affix--trailing">
        <slot name="trailing" />
      </span>
    </div>
    <p v-if="error" :id="descriptionId" class="f0-input-error">{{ error }}</p>
    <p v-else-if="help" :id="descriptionId" class="f0-input-help">{{ help }}</p>
  </div>
</template>

<script lang="ts">
let inputCounter = 0
</script>

<script setup lang="ts">
import { computed } from 'vue'

const generatedId = `f0-input-${++inputCounter}`

const props = withDefaults(
  defineProps<{
    id?: string
    modelValue?: string
    label?: string
    type?: string
    name?: string
    placeholder?: string
    autocomplete?: string
    inputmode?: 'none' | 'text' | 'decimal' | 'numeric' | 'tel' | 'search' | 'email' | 'url'
    help?: string
    error?: string
    required?: boolean
    autofocus?: boolean
    disabled?: boolean
  }>(),
  {
    modelValue: '',
    type: 'text',
    required: false,
    autofocus: false,
    disabled: false
  }
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'input', value: string): void
  (event: 'focus', value: FocusEvent): void
  (event: 'blur', value: FocusEvent): void
}>()

function handleInput(event: Event): void {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', value)
  emit('input', value)
}

const resolvedId = computed(() => props.id || generatedId)
const descriptionId = computed(() =>
  props.help || props.error ? `${resolvedId.value}-description` : undefined
)
</script>
