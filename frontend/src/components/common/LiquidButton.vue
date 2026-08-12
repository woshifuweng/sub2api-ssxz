<script setup lang="ts">
import type { HTMLAttributes } from "vue";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const liquidbuttonVariants = cva(
  "isolate inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 overflow-hidden whitespace-nowrap rounded-md text-sm font-medium outline-none transition-[color,background-color,border-color,box-shadow] disabled:pointer-events-none disabled:opacity-50 focus-visible:ring-[3px] focus-visible:ring-ring/50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-transparent text-primary hover:bg-accent/70",
        destructive: "bg-destructive text-white hover:bg-destructive/90",
        outline:
          "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        plain: "",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-9 px-3 text-xs",
        lg: "h-10 rounded-md px-6",
        xl: "h-12 rounded-md px-8",
        xxl: "h-14 rounded-md px-10",
        icon: "size-9 rounded-full p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "xxl",
    },
  },
);

type LiquidButtonVariants = VariantProps<typeof liquidbuttonVariants>;

interface Props {
  variant?: LiquidButtonVariants["variant"];
  size?: LiquidButtonVariants["size"];
  class?: HTMLAttributes["class"];
  as?: string;
}

const props = withDefaults(defineProps<Props>(), {
  as: "button",
  variant: "default",
  size: "xxl",
});
</script>

<template>
  <component
    :is="props.as"
    :class="
      cn(
        'relative',
        liquidbuttonVariants({ variant: props.variant, size: props.size }),
        props.variant === 'plain' &&
          props.size !== 'icon' &&
          'h-auto min-h-0 w-auto p-0',
        props.class,
      )
    "
    v-bind="$attrs"
  >
    <!-- Default actions keep one liquid surface; other variants use their own single shell. -->
    <span
      v-if="props.variant === 'default'"
      aria-hidden="true"
      class="pointer-events-none absolute inset-0 z-0 rounded-[inherit] transition-shadow shadow-[0_0_6px_rgba(0,0,0,0.03),0_2px_6px_rgba(0,0,0,0.08),inset_3px_3px_0.5px_-3px_rgba(0,0,0,0.9),inset_-3px_-3px_0.5px_-3px_rgba(0,0,0,0.85),inset_1px_1px_1px_-0.5px_rgba(0,0,0,0.6),inset_-1px_-1px_1px_-0.5px_rgba(0,0,0,0.6),inset_0_0_6px_6px_rgba(0,0,0,0.12),inset_0_0_2px_2px_rgba(0,0,0,0.06),0_0_12px_rgba(255,255,255,0.15)] dark:shadow-[0_0_8px_rgba(0,0,0,0.03),0_2px_6px_rgba(0,0,0,0.08),inset_3px_3px_0.5px_-3.5px_rgba(255,255,255,0.09),inset_-3px_-3px_0.5px_-3.5px_rgba(255,255,255,0.85),inset_1px_1px_1px_-0.5px_rgba(255,255,255,0.6),inset_-1px_-1px_1px_-0.5px_rgba(255,255,255,0.6),inset_0_0_6px_6px_rgba(255,255,255,0.12),inset_0_0_2px_2px_rgba(255,255,255,0.06),0_0_12px_rgba(0,0,0,0.15)]"
    />
    <slot />
  </component>
</template>
