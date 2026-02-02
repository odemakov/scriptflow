<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useRunStore } from "@/stores/RunStore";
import { CRunStatus } from "@/types";

const props = defineProps<{
  taskId: string;
}>();

const useRuns = useRunStore();
const failureCount = ref(0);
const isLoading = ref(true);
const errorTypes = [CRunStatus.error, CRunStatus.internal_error];

// Emits the count to parent components
const emit = defineEmits<{
  (e: "update:count-and-updated", count: number, updated: string): void;
}>();

// Calculate consecutive failure count
const calculateFailureCount = () => {
  const runs = useRuns.getLastRuns[props.taskId] || [];

  // Filter out 'started' runs
  const completedRuns = runs.filter((run) => run.status !== CRunStatus.started);

  if (completedRuns.length === 0) {
    return { count: 0, updated: "" };
  }

  // If the most recent run is not an error, return 0
  if (!errorTypes.includes(completedRuns[0].status)) {
    return { count: 0, updated: completedRuns[0].updated };
  }

  // Count consecutive failures
  let count = 0;
  for (const run of completedRuns) {
    if (errorTypes.includes(run.status)) {
      count++;
    } else {
      break;
    }
  }

  return { count, updated: completedRuns[0].updated };
};

// Update the failure count and emit the new value
const updateFailureCount = () => {
  const result = calculateFailureCount();
  if (result.count !== failureCount.value) {
    failureCount.value = result.count;
    emit("update:count-and-updated", result.count, result.updated);
  }
};

// Watch for changes in the runs data for this task
watch(
  () => useRuns.getLastRuns[props.taskId],
  () => {
    updateFailureCount();
  },
  { deep: true, immediate: true },
);

// Load data
onMounted(async () => {
  // Skip fetch if data already exists (batch fetched by parent)
  if (useRuns.getLastRuns[props.taskId]) {
    updateFailureCount();
    isLoading.value = false;
    return;
  }

  isLoading.value = true;
  try {
    // Fetch run data for this task
    await useRuns.fetchLastRuns(props.taskId, 10, false);
    updateFailureCount();
  } catch (error) {
    console.error("Error fetching run data:", error);
  } finally {
    isLoading.value = false;
  }
});
</script>

<template>
  <div>
    <span v-if="isLoading" class="loading loading-spinner loading-xs"></span>
    <span v-else>{{ failureCount }}</span>
  </div>
</template>
