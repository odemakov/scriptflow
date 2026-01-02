<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from "vue";
import router from "@/router";

import { useTaskStore } from "@/stores/TaskStore";
import { useRunStore } from "@/stores/RunStore";
import IdentifierUrl from "@/components/IdentifierUrl.vue";
import RunTimeAgo from "@/components/RunTimeAgo.vue";
import ConsecutiveFailureCount from "@/components/ConsecutiveFailureCount.vue";
import { ITask } from "@/types";

// Simple interface for tasks with failure count
interface TaskWithFailureInfo extends ITask {
  consecutiveFailedCount: number;
  runUpdated: string;
  countLoaded: boolean;
}

const useTask = useTaskStore();
const useRuns = useRunStore();
const tasks = ref<TaskWithFailureInfo[]>([]);
const isLoading = ref(true);
const showAllTasks = ref(false);
// We'll use this to force re-evaluation of computed properties
const forceRefresh = ref(0);

// Track when all runs data has been loaded
const allRunsLoaded = computed(() => {
  return tasks.value.every((task) => task.countLoaded);
});

// Filtered tasks based on the toggle
const displayedTasks = computed(() => {
  // Include forceRefresh in dependency tracking
  forceRefresh.value;

  // Show all tasks initially until all counts are loaded
  if (!allRunsLoaded.value) {
    return tasks.value;
  }

  if (showAllTasks.value) {
    return tasks.value;
  } else {
    return tasks.value.filter((task) => task.consecutiveFailedCount > 0);
  }
});

// Sort tasks by runUpdated value
const sortedTasks = computed(() => {
  return displayedTasks.value.sort((a, b) => {
    const dateA = new Date(a.runUpdated);
    const dateB = new Date(b.runUpdated);
    return dateB.getTime() - dateA.getTime();
  });
});

// Navigate to task details
const gotoTask = (taskId: string) => {
  router.push({ name: "task", params: { taskId } });
};

// Update a task's failure count
const updateFailureCount = (taskId: string, count: number, runUpdated: string) => {
  const index = tasks.value.findIndex((t) => t.id === taskId);
  if (index !== -1) {
    const oldCount = tasks.value[index].consecutiveFailedCount;
    tasks.value[index].consecutiveFailedCount = count;
    tasks.value[index].runUpdated = runUpdated;
    tasks.value[index].countLoaded = true;

    // If going from 0 to non-zero or vice versa, force refresh
    if ((oldCount === 0 && count > 0) || (oldCount > 0 && count === 0)) {
      forceRefresh.value++;
    }
  }
};

// Watch for changes in the runs store
watch(
  () => useRuns.getLastRuns,
  () => {
    // Force refresh when run data changes
    forceRefresh.value++;
  },
  { deep: true },
);

// Watch for all runs being loaded
watch(allRunsLoaded, (newValue) => {
  if (newValue) {
    // When all counts are loaded, force refresh to apply filtering
    forceRefresh.value++;
  }
});

onMounted(async () => {
  isLoading.value = true;
  try {
    // Fetch tasks
    await useTask.fetchTasks();

    // Initialize with all tasks, failure count = 0
    tasks.value = useTask.getTasks.map((task: ITask) => ({
      ...task,
      consecutiveFailedCount: 0,
      runUpdated: "",
      countLoaded: false,
    }));

    // Subscribe to run updates for each task
    for (const task of tasks.value) {
      await useRuns.subscribe({ taskId: task.id });
    }
  } finally {
    isLoading.value = false;
  }
});

onBeforeUnmount(() => {
  // Clean up subscriptions for all tasks
  for (const task of tasks.value) {
    useRuns.unsubscribe({ taskId: task.id });
  }
});
</script>

<template>
  <div class="mx-auto p-1 md:p-2 rounded overflow-x-auto">
    <!-- Loading indicator -->
    <div v-if="isLoading" class="text-center py-4">
      <span class="loading loading-spinner loading-md"></span>
    </div>

    <!-- No tasks message -->
    <div v-else-if="displayedTasks.length === 0" class="text-center py-4 text-gray-500">
      <span v-if="!showAllTasks">No failed tasks found</span>
      <span v-else>No tasks found</span>
    </div>

    <!-- Task table -->
    <table v-else class="table table-sm w-full mx-auto text-sm md:text-base">
      <thead>
        <tr>
          <th class="whitespace-nowrap px-2 md:px-4">id</th>
          <th class="whitespace-nowrap px-2 md:px-4">name</th>
          <th class="whitespace-nowrap px-2 md:px-4">failures</th>
          <th class="whitespace-nowrap px-2 md:px-4">updated</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="task in sortedTasks"
          :key="task.id"
          class="hover:bg-base-200"
          :class="{ hidden: task.consecutiveFailedCount == 0 }"
        >
          <td class="p-1 md:p-2">
            <IdentifierUrl :id="task.id" @click="gotoTask(task.id)" />
          </td>
          <td class="p-1 md:p-2">{{ task.name }}</td>
          <td class="p-1 md:p-2 text-xs">
            <ConsecutiveFailureCount
              :taskId="task.id"
              @update:count-and-updated="
                (count: number, updated: string) =>
                  updateFailureCount(task.id, count, updated)
              "
            />
          </td>
          <td class="p-1 md:p-2 text-xs">
            <RunTimeAgo :datetime="task.runUpdated" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
