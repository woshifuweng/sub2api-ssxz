<template>
  <div ref="rootRef" class="model-selector">
    <LiquidButton
      type="button"
      class="model-trigger"
      data-testid="workspace-model-trigger"
      :disabled="!models.length"
      :aria-expanded="open"
      aria-controls="workspace-model-menu"
      @click.stop="toggle"
      variant="plain"
      size="sm"
    >
      <span>{{ selectedLabel }}</span>
      <Icon name="chevronDown" size="xs" />
    </LiquidButton>
    <div
      v-if="open"
      id="workspace-model-menu"
      class="model-menu"
      data-testid="workspace-model-menu"
    >
      <LiquidButton
        v-for="model in models"
        :key="model.id"
        type="button"
        class="model-option"
        data-testid="workspace-model-option"
        :class="{ 'is-selected': model.id === selectedModel }"
        @click.stop="select(model.id)"
        variant="plain"
        size="sm"
      >
        <span>{{ model.name || model.id }}</span>
      </LiquidButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import Icon from "@/components/icons/Icon.vue";
import type { ChatModelOption } from "@/composables/useUserCapabilities";

const props = defineProps<{
  models: ChatModelOption[];
  selectedModel: string;
}>();

const emit = defineEmits<{
  (event: "update:selectedModel", value: string): void;
}>();

const open = ref(false);
const rootRef = ref<HTMLElement | null>(null);

const selectedLabel = computed(() => {
  const model = props.models.find((item) => item.id === props.selectedModel);
  return model?.name || props.selectedModel || "暂无可用模型";
});

function toggle() {
  if (!props.models.length) return;
  open.value = !open.value;
}

function select(modelId: string) {
  emit("update:selectedModel", modelId);
  open.value = false;
}

function handlePointerDown(event: PointerEvent) {
  if (!open.value) return;
  if (event.target instanceof Node && rootRef.value?.contains(event.target))
    return;
  open.value = false;
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") open.value = false;
}

onMounted(() => {
  document.addEventListener("pointerdown", handlePointerDown);
  document.addEventListener("keydown", handleKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", handlePointerDown);
  document.removeEventListener("keydown", handleKeydown);
});
</script>
