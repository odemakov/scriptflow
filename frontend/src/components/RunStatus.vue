<script setup lang="ts">
import { computed } from 'vue';
import { RunStatusClass } from "@/lib/helpers";

const props = defineProps<{
  run: IRun,
  title?: string
}>()

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
  <span :class="RunStatusClass(props.run.status)">
    {{ runStatus }}
  </span>
</template>