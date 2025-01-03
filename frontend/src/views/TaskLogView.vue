<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useTaskStore } from "@/stores/TaskStore";
import { useToastStore } from "@/stores/ToastStore";
import { ICrumb } from "@/types";
import PageTitle from "@/components/PageTitle.vue";
import TaskCard from "@/components/TaskCard.vue";
import TaskLogTerminal from "@/components/TaskLogTerminal.vue";
import Breadcrumbs from "@/components/Breadcrumbs.vue";
import SubscriptionCard from "@/components/SubscriptionCard.vue";

const useToasts = useToastStore();
const useTasks = useTaskStore();
const router = useRouter();
const route = useRoute();
const taskId = Array.isArray(route.params.taskId)
  ? route.params.taskId[0]
  : route.params.taskId;
const projectId = Array.isArray(route.params.projectId)
  ? route.params.projectId[0]
  : route.params.projectId;

const task = computed(() => useTasks.getTask);

onMounted(async () => {
  try {
    await useTasks.fetchTask(taskId);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
});

const gotoTaskHistory = () => {
  router.push({
    name: "task",
    params: { projectId: projectId, taskId: taskId },
  });
};

const crumbs = [
  {
    to: () =>
      router.push({ name: "project", params: { projectId: projectId } }),
    label: projectId,
  } as ICrumb,
  { label: taskId } as ICrumb,
];
</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle title="Task today's log" />
  <div class="flex flex-col lg:flex-row gap-4">
    <div v-if="task" class="w-full lg:basis-1/4 flex flex-col gap-4">
      <TaskCard :task="task" />
      <SubscriptionCard :task="task" />
    </div>
    <div class="w-full lg:basis-3/4">
      <div role="tablist" class="tabs tabs-lifted">
        <a role="tab" class="tab" @click="gotoTaskHistory()">History</a>
        <div
          role="tabpanel"
          class="tab-content bg-base-100 border-base-300 rounded-box p-6"
        ></div>
        <a role="tab" class="tab tab-active">Logs</a>
        <div
          role="tabpanel"
          class="tab-content bg-base-100 border-base-300 rounded-box p-6"
        >
          <TaskLogTerminal :task="task" />
        </div>
      </div>
    </div>
  </div>
</template>
