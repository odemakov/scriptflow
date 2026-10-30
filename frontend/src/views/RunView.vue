<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useRunStore } from "@/stores/RunStore";
import { useToastStore } from "@/stores/ToastStore";
import { ICrumb, IRun } from "@/types";
import PageTitle from "@/components/PageTitle.vue";
import RunCard from "@/components/RunCard.vue";
import RunLogTerminal from "@/components/RunLogTerminal.vue";
import Breadcrumbs from "@/components/Breadcrumbs.vue";
import { UpdateTitle, isEditableTarget } from "@/lib/helpers";

const useToasts = useToastStore();
const useRuns = useRunStore();
const route = useRoute();
const router = useRouter();
const projectId = Array.isArray(route.params.projectId)
  ? route.params.projectId[0]
  : route.params.projectId;
const nodeId = Array.isArray(route.params.nodeId)
  ? route.params.nodeId[0]
  : route.params.nodeId;
const taskId = Array.isArray(route.params.taskId)
  ? route.params.taskId[0]
  : route.params.taskId;

const runId = computed(() => (Array.isArray(route.params.id) ? route.params.id[0] : route.params.id));
const run = computed(() => useRuns.getRun);
const neighbors = computed(() => useRuns.getRunNeighbors[runId.value] ?? null);
const hasPrev = computed(() => !!neighbors.value?.prev);
const hasNext = computed(() => !!neighbors.value?.next);

async function loadRun(id: string) {
  UpdateTitle(`Run log: ${id}`);
  try {
    await useRuns.fetchRun(id);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
    return;
  }
  const current = useRuns.getRun;
  if (!current?.task || !current?.created) return;
  try {
    await useRuns.fetchRunNeighbors(id, current.task, current.created);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
}

function navigateTo(neighbor: IRun | null | undefined) {
  if (!neighbor) return;
  router.push({ name: route.name!, params: { ...route.params, id: neighbor.id } });
}

function onKeydown(event: KeyboardEvent) {
  if (isEditableTarget(event) || event.ctrlKey || event.metaKey || event.altKey) return;
  if (event.key === "ArrowLeft") {
    navigateTo(neighbors.value?.prev);
  } else if (event.key === "ArrowRight") {
    navigateTo(neighbors.value?.next);
  }
}

onMounted(async () => {
  window.addEventListener("keydown", onKeydown);
  await loadRun(runId.value);
});

watch(runId, (id) => loadRun(id));

onUnmounted(() => {
  window.removeEventListener("keydown", onKeydown);
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
  breadcrumbs.push({ label: runId.value } as ICrumb);

  return breadcrumbs;
});
</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle title="Run log">
    <template #left>
      <button
        class="btn btn-sm btn-ghost"
        :disabled="!hasPrev"
        title="Older run (Left arrow)"
        @click="navigateTo(neighbors?.prev)"
      >
        &larr; Prev
      </button>
    </template>
    <template #right>
      <button
        class="btn btn-sm btn-ghost"
        :disabled="!hasNext"
        title="Newer run (Right arrow)"
        @click="navigateTo(neighbors?.next)"
      >
        Next &rarr;
      </button>
    </template>
  </PageTitle>
  <div class="flex flex-row gap-4">
    <div class="basis-1/4">
      <RunCard :run="run" />
    </div>
    <div class="basis-3/4">
      <RunLogTerminal :runId="runId" />
    </div>
  </div>
</template>
