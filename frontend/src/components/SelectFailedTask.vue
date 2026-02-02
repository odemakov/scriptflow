<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount } from "vue";
import router from "@/router";

import { useTaskStore } from "@/stores/TaskStore";
import IdentifierUrl from "@/components/IdentifierUrl.vue";
import RunTimeAgo from "@/components/RunTimeAgo.vue";
import { ITask } from "@/types";

const useTask = useTaskStore();

// Filtered tasks with failures, sorted by updated
const failedTasks = computed(() => {
  return useTask.getTasks
    .filter((task: ITask) => (task.consecutive_failure_count ?? 0) > 0)
    .sort((a: ITask, b: ITask) => new Date(b.updated).getTime() - new Date(a.updated).getTime());
});

// Navigate to task details
const gotoTask = (taskId: string) => {
  router.push({ name: "task", params: { taskId } });
};

onMounted(async () => {
  await useTask.fetchTasks();
  await useTask.subscribe();
});

onBeforeUnmount(() => {
  useTask.unsubscribe();
});
</script>

<template>
  <div class="mx-auto p-1 md:p-2 rounded overflow-x-auto">
    <!-- No tasks message -->
    <div v-if="failedTasks.length === 0" class="text-center py-4 text-gray-500">
      No failed tasks found
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
          v-for="task in failedTasks"
          :key="task.id"
          class="hover:bg-base-200"
        >
          <td class="p-1 md:p-2">
            <IdentifierUrl :id="task.id" @click="gotoTask(task.id)" />
          </td>
          <td class="p-1 md:p-2">{{ task.name }}</td>
          <td class="p-1 md:p-2 text-xs">
            {{ task.consecutive_failure_count }}
          </td>
          <td class="p-1 md:p-2 text-xs">
            <RunTimeAgo :datetime="task.updated" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
