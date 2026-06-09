<script setup lang="ts">
import LogViewer from "@/components/LogViewer.vue";

const props = defineProps<{
  task: ITask;
}>();
</script>

<template>
  <div class="h-full w-full">
    <LogViewer mode="live" :task="props.task" />
  </div>
</template>
