<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useTaskStore } from "@/stores/TaskStore";
import { useToastStore } from "@/stores/ToastStore";
import { ICrumb } from "@/types";
import PageTitle from "@/components/PageTitle.vue";
import TaskCard from "@/components/TaskCard.vue";
import TaskRuns from "@/components/TaskRuns.vue";
import Breadcrumbs from "@/components/Breadcrumbs.vue";
import SubscriptionCard from "@/components/SubscriptionCard.vue";

const useToasts = useToastStore();
const useTasks = useTaskStore();
const router = useRouter();
const route = useRoute();
const projectId = Array.isArray(route.params.projectId)
  ? route.params.projectId[0]
  : route.params.projectId;
const nodeId = Array.isArray(route.params.nodeId)
  ? route.params.nodeId[0]
  : route.params.nodeId;
const taskId = Array.isArray(route.params.taskId)
  ? route.params.taskId[0]
  : route.params.taskId;

const task = computed(() => useTasks.getTask);

onMounted(async () => {
  try {
    await useTasks.fetchTask(taskId);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
});

const gotoTaskLog = () => {
  const params: { taskId: string; projectId?: string; nodeId?: string } = { taskId };

  if (projectId) {
    params.projectId = projectId;
    router.push({ name: "project-task-log", params });
  } else if (nodeId) {
    params.nodeId = nodeId;
    router.push({ name: "node-task-log", params });
  } else {
    router.push({ name: "task-log", params });
  }
};

const crumbs = computed(() => {
  const baseCrumb = { label: taskId } as ICrumb;

  if (projectId) {
    return [
      {
        label: projectId,
        to: () => router.push({ name: "project", params: { projectId } }),
      } as ICrumb,
      baseCrumb,
    ];
  }

  if (nodeId) {
    return [
      {
        label: nodeId,
        to: () => router.push({ name: "node", params: { nodeId } }),
      } as ICrumb,
      baseCrumb,
    ];
  }

  return [baseCrumb];
});
</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle title="Task history" />
  <div class="flex flex-col lg:flex-row gap-4">
    <div v-if="task" class="w-full lg:basis-1/4 flex flex-col gap-4">
      <TaskCard :task="task" />
      <SubscriptionCard :task="task" />
    </div>
    <div class="w-full lg:basis-3/4">
      <div role="tablist" class="tabs tabs-lifted">
        <a role="tab" class="tab tab-active">History</a>
        <div
          role="tabpanel"
          class="tab-content bg-base-100 border-base-300 rounded-box p-6"
        >
          <TaskRuns :task="task" :projectId="projectId" :nodeId="nodeId" />
        </div>
        <a role="tab" class="tab" @click="gotoTaskLog()">Logs</a>
        <div
          role="tabpanel"
          class="tab-content bg-base-100 border-base-300 rounded-box p-6"
        ></div>
      </div>
    </div>
  </div>
</template>
