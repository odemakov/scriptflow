<script setup lang="ts">
defineProps<{
  id: string | undefined
}>()
</script>
<template>
  <span
    class="bg-slate-300 text-gray-900 font-mono whitespace-nowrap rounded-md py-1 px-2 my-1 text-xs hover:cursor-pointer">
    {{ id }}
  </span>
</template>