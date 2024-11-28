<script setup lang="ts">
defineProps<{
  id: string | undefined
}>()
</script>
<template>
  <span class="bg-slate-200 text-gray-700 font-mono rounded-md py-1 px-2 my-1 text-xs">
    {{ id }}
  </span>
</template>