<script setup lang="ts">
const props = defineProps<{
  command: string;
}>();
</script>
<template>
  <div class="relative">
    <pre
      class="text-xs text-left bg-primary-content text-accent-content py-2 px-1 my-1 rounded-md max-w-[1000px] overflow-x-auto"
      >{{ props.command }}</pre
    >
  </div>
</template>
