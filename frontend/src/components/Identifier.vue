<script setup lang="ts">
const props = defineProps<{
  id: string | undefined;
}>();
</script>
<template>
  <span
    class="bg-slate-200 text-gray-700 font-mono whitespace-nowrap rounded-md py-1 px-2 my-1 text-xs"
  >
    {{ props.id }}
  </span>
</template>
