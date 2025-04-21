<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useRunStore } from "@/stores/RunStore";
import { useToastStore } from "@/stores/ToastStore";
import { ICrumb } from "@/types";
import PageTitle from "@/components/PageTitle.vue";
import RunCard from "@/components/RunCard.vue";
import RunLogTerminal from "@/components/RunLogTerminal.vue";
import Breadcrumbs from "@/components/Breadcrumbs.vue";

const useToasts = useToastStore();
const useRuns = useRunStore();
const route = useRoute();
const router = useRouter();
const runId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id;
const projectId = Array.isArray(route.params.projectId)
  ? route.params.projectId[0]
  : route.params.projectId;
const nodeId = Array.isArray(route.params.nodeId)
  ? route.params.nodeId[0]
  : route.params.nodeId;
const taskId = Array.isArray(route.params.taskId)
  ? route.params.taskId[0]
  : route.params.taskId;

const run = computed(() => useRuns.getRun);

onMounted(async () => {
  try {
    await useRuns.fetchRun(runId);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
});

const crumbs = computed(() => {
  const breadcrumbs = [];

  // Add project or node crumb if available
  if (projectId) {
    breadcrumbs.push({
      label: projectId,
      to: () => router.push({ name: "project", params: { projectId } }),
    } as ICrumb);
  } else if (nodeId) {
    breadcrumbs.push({
      label: nodeId,
      to: () => router.push({ name: "node", params: { nodeId } }),
    } as ICrumb);
  }

  // Add task crumb
  breadcrumbs.push({
    label: taskId,
    to: () => {
      const routeName = projectId ? "project-task" : nodeId ? "node-task" : "task";
      const params = projectId
        ? { projectId, taskId }
        : nodeId
          ? { nodeId, taskId }
          : { taskId };

      router.push({ name: routeName, params });
    },
  } as ICrumb);

  // Add run crumb
  breadcrumbs.push({ label: runId } as ICrumb);

  return breadcrumbs;
});
</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle title="Run log" />
  <div class="flex flex-row gap-4">
    <div class="basis-1/4">
      <RunCard :run="run" />
    </div>
    <div class="basis-3/4">
      <RunLogTerminal :run="run" />
    </div>
  </div>
</template>
