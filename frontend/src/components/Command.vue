<script setup lang="ts">
const props = defineProps<{
  command: string;
}>();
</script>
<template>
  <div
    class="relative font-mono text-xs text-left bg-primary-content text-accent-content py-2 px-1 my-1 rounded-md w-full overflow-x-auto"
  >
    {{ props.command }}
  </div>
</template>
