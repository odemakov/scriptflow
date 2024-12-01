<script setup lang="ts">
const props = defineProps<{
  command: string,
}>()
</script>
<template>
  <pre
    class="text-xs text-left bg-primary-content text-accent-content py-2 px-1 my-1 rounded-md max-w-[500px] overflow-x-auto">{{ props.command }}</pre>
</template>