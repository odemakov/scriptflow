<script setup lang="ts">
import { computed, onMounted } from "vue";
import router from "@/router";

import { useTaskStore } from "@/stores/TaskStore";
import IdentifierUrl from "@/components/IdentifierUrl.vue";

const useTask = useTaskStore();
const tasks = computed(() => useTask.getTasks);

const gotoTask = (taskId: string) => {
  router.push({ name: "task", params: { taskId: taskId } });
};

onMounted(async () => {
  await useTask.fetchTasks();
});
</script>

<template>
  <div class="mx-auto p-1 md:p-2 rounded overflow-x-auto">
    <table class="table table-sm w-full mx-auto text-sm md:text-base">
      <thead class="">
        <tr>
          <th class="whitespace-nowrap px-2 md:px-4">id</th>
          <th class="whitespace-nowrap px-2 md:px-4">name</th>
          <th class="whitespace-nowrap px-2 md:px-4">failed count</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="task in tasks" :key="task.id">
          <td class="p-1 md:p-2">
            <IdentifierUrl :id="task.id" @click="gotoTask(task.id)" />
          </td>
          <td class="p-1 md:p-2">{{ task.name }}</td>
          <td class="p-1 md:p-2 text-xs">{{ task.consecutiveFailedCount }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
