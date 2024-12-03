<script setup lang="ts">
defineProps<{
  title: string
}>()
</script>

<template>
  <div class="text-center pb-4 flex justify-center items-center space-x-4">
    <div class="text-2xl text-primary">
      {{ title }}
    </div>
  </div>
</template>