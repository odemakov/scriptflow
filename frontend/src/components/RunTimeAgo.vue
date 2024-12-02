<script setup lang="ts">
import { computed } from 'vue';
import { TimeAgo } from '@/lib/helpers';

const props = defineProps<{
  datetime: string,
}>()

// Reactive computation of `timeAgo`
// TODO: it actually doesn't update data-tip prop
const timeAgo = computed(() => {
  return TimeAgo(props.datetime);
});

</script>

<template>
  <div class="tooltip" :data-tip="`${timeAgo} ago`">
    <span class="text-brown">{{ props.datetime }}</span>
  </div>
</template>
