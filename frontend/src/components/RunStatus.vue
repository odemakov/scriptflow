<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  run: IRun,
  title?: string
}>()

// color class based on run status
const colorClass = computed(() => {
  switch (props.run.status) {
    case CRunStatus.completed:
      return 'badge-success bg-opacity-60'
    case CRunStatus.started:
      return 'badge-info bg-opacity-60'
    case CRunStatus.error:
    case CRunStatus.internal_error:
      return 'badge-error bg-opacity-60'
    case CRunStatus.interrupted:
      return 'badge-warning bg-opacity-60'
    default:
      return ''
  }
})
const runStatus = computed(() => {
  if (props.run.status == CRunStatus.error) {
    return `${props.run.status}(${props.run.exit_code})`
  } else if (props.run.status == CRunStatus.internal_error) {
    return `${props.run.status}(SSH)`
  } else {
    return props.run.status
  }
})

</script>

<template>
  <span :class="colorClass" class="badge">
    {{ runStatus }}
  </span>
</template>