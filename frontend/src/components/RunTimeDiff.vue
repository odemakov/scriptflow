<script setup lang="ts">
import { computed } from 'vue';
import { TimeDiff } from '@/lib/helpers';

const props = defineProps<{
  run: IRun,
}>()

// computed running time property
const runningTime = computed(() => {
  if (props.run.status == CRunStatus.started) {
    return TimeDiff(props.run.created, new Date().toISOString())
  } else {
    return TimeDiff(props.run.created, props.run.updated)
  }
})

</script>

<template>
  <span>
    <span class="text-brown">{{ runningTime }}</span>
  </span>
</template>