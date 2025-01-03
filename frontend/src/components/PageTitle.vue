<script setup lang="ts">
const props = defineProps<{
  title: string;
}>();
</script>

<template>
  <div
    class="text-center pb-1 md:pb-2 flex flex-col md:flex-row justify-center items-center space-y-2 md:space-y-0 md:space-x-4"
  >
    <div class="text-xl md:text-2xl text-primary">
      {{ props.title }}
    </div>
  </div>
</template>
