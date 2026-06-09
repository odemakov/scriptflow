<script setup lang="ts">
import LogViewer from "@/components/LogViewer.vue";

const props = defineProps<{
  run: IRun;
}>();
</script>

<template>
  <div class="h-full w-full">
    <LogViewer mode="static" :runId="props.run.id" logHeight="calc(100vh - 15rem)" />
  </div>
</template>
