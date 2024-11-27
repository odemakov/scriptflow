<script setup lang="ts">
import { IBack } from '@/types';
import { onMounted, onUnmounted } from 'vue';
const props = defineProps<{
  title: string
  back?: IBack
}>()

const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === 'b' || event.key === 'B') {
    props.back?.to();
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyPress);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyPress);
});

</script>

<template>
  <div class="text-center pb-4 flex justify-center items-center space-x-4">
    <div class="text-xl text-primary">
      {{ props.title }}
    </div>
    <div v-if="props.back?.label" class="text-sm">
      Press
      <kbd class="kbd kbd-xs">B</kbd>
      to go
      <span @click="props.back?.to" class="cursor-pointer bg-slate-100 p-1 rounded-lg">
        {{ props.back?.label }}
      </span>
    </div>
  </div>
</template>