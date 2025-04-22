<script setup lang="ts">
const props = defineProps<{
  command: string;
}>();
</script>
<template>
  <div
    class="relative font-mono text-xs text-left bg-base-200 text-base-content py-2 px-1 my-1 rounded-md w-full overflow-x-auto border border-base-300"
  >
    {{ props.command }}
  </div>
</template>
