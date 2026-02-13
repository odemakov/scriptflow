<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from "vue";
import { useRouter } from "vue-router";

import { useToastStore } from "@/stores/ToastStore";
import { useTaskStore } from "@/stores/TaskStore";
import { useRunStore } from "@/stores/RunStore";
import { isAutoCancelError } from "@/lib/helpers";

import IdentifierUrl from "@/components/IdentifierUrl.vue";
import Command from "@/components/Command.vue";
import RunStatus from "@/components/RunStatus.vue";
import RunTimeAgo from "@/components/RunTimeAgo.vue";
import PageTitle from "@/components/PageTitle.vue";
import { ICrumb, CRunStatus, IRun, ITask, CEntityType, EntityType, type TaskSortField } from "@/types";
import Breadcrumbs from "@/components/Breadcrumbs.vue";
import RunTimeDiff from "@/components/RunTimeDiff.vue";
import { UpdateTitle, Capitalize } from "@/lib/helpers";

const props = defineProps<{
  entityId: string;
  entityType: EntityType;
  pageTitle: string;
}>();

const router = useRouter();
const useToasts = useToastStore();
const useTasks = useTaskStore();
const useRuns = useRunStore();

const loading = ref(true);
const sortField = computed(() => useTasks.sortField);
const sortDirection = computed(() => useTasks.sortDirection);

const rawTasks = computed(() => {
  return props.entityType === CEntityType.node
    ? useTasks.getTasksByNode
    : useTasks.getTasksByProject;
});
const lastRuns = computed(() => useRuns.getLastRuns);

const tasks = computed(() => {
  const items = [...rawTasks.value];
  const dir = sortDirection.value === "asc" ? 1 : -1;

  return items.sort((a: ITask, b: ITask) => {
    const runA = taskLastRun(a.id);
    const runB = taskLastRun(b.id);

    switch (sortField.value) {
      case "id":
        return dir * a.id.localeCompare(b.id);
      case "run_status": {
        const sA = runA?.status ?? "";
        const sB = runB?.status ?? "";
        return dir * sA.localeCompare(sB);
      }
      case "running_time": {
        const tA = runA ? new Date(runA.updated).getTime() - new Date(runA.created).getTime() : 0;
        const tB = runB ? new Date(runB.updated).getTime() - new Date(runB.created).getTime() : 0;
        return dir * (tA - tB);
      }
      case "run_updated": {
        const uA = runA?.updated ?? "";
        const uB = runB?.updated ?? "";
        return dir * uA.localeCompare(uB);
      }
      default:
        return 0;
    }
  });
});

const toggleSort = (field: TaskSortField) => {
  if (sortField.value === field) {
    useTasks.setSortDirection(sortDirection.value === "asc" ? "desc" : "asc");
  } else {
    useTasks.setSortField(field);
    useTasks.setSortDirection(field === "id" ? "asc" : "desc");
  }
};
const taskLastRun = (taskId: string) => {
  if (taskId in lastRuns.value && lastRuns.value[taskId].length > 0) {
    return lastRuns.value[taskId][0];
  } else {
    return null;
  }
};

const fetchTasks = async () => {
  try {
    if (props.entityType === CEntityType.node) {
      await useTasks.fetchTasksByNode(props.entityId);
      await useTasks.subscribe({ nodeId: props.entityId });
    } else {
      await useTasks.fetchTasksByProject(props.entityId);
      await useTasks.subscribe({ projectId: props.entityId });
    }
  } catch (error: unknown) {
    if (!isAutoCancelError(error)) {
      useToasts.addToast((error as Error).message, "error");
    }
  }
};

const fetchLastRunsAndSubscribe = async () => {
  try {
    const taskIds = tasks.value.map((t: ITask) => t.id);
    await useRuns.fetchLatestRuns(taskIds);

    // Single subscription for all runs in this entity
    if (props.entityType === CEntityType.node) {
      await useRuns.subscribe({ nodeId: props.entityId });
    } else {
      await useRuns.subscribe({ projectId: props.entityId });
    }
  } catch (error: unknown) {
    if (!isAutoCancelError(error)) {
      useToasts.addToast((error as Error).message, "error");
    }
  }
};

onMounted(async () => {
  UpdateTitle(`${Capitalize(props.entityType)}: ${props.entityId}`);

  loading.value = true;

  // unsubscribe just in case
  if (props.entityType === CEntityType.node) {
    useTasks.unsubscribe({ nodeId: props.entityId });
    useRuns.unsubscribe({ nodeId: props.entityId });
  } else {
    useTasks.unsubscribe({ projectId: props.entityId });
    useRuns.unsubscribe({ projectId: props.entityId });
  }

  // fetch tasks
  await fetchTasks();

  // for each task fetch last run
  await fetchLastRunsAndSubscribe();

  loading.value = false;
});

onBeforeUnmount(() => {
  if (props.entityType === CEntityType.node) {
    useTasks.unsubscribe({ nodeId: props.entityId });
    useRuns.unsubscribe({ nodeId: props.entityId });
  } else {
    useTasks.unsubscribe({ projectId: props.entityId });
    useRuns.unsubscribe({ projectId: props.entityId });
  }
});

const gotoTask = (taskId: string) => {
  const routeParams =
    props.entityType === CEntityType.project
      ? { projectId: props.entityId, taskId }
      : { nodeId: props.entityId, taskId };

  router.push({
    name: props.entityType === CEntityType.project ? "project-task" : "node-task",
    params: routeParams,
  });
};

const gotoRun = (task: ITask, run: IRun) => {
  const baseParams =
    props.entityType === CEntityType.project
      ? { projectId: props.entityId, taskId: task.id }
      : { nodeId: props.entityId, taskId: task.id };

  if (run.status === CRunStatus.started) {
    router.push({
      name:
        props.entityType === CEntityType.project
          ? "project-task-tail"
          : "node-task-tail",
      params: baseParams,
    });
  } else {
    router.push({
      name:
        props.entityType === CEntityType.project ? "project-task-run" : "node-task-run",
      params: { ...baseParams, id: run.id },
    });
  }
};

const toggleTaskActive = async (taskId: string) => {
  try {
    await useTasks.toggleTaskActive(taskId);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
};

const crumbs = [{ label: props.entityId } as ICrumb];
</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle :title="pageTitle" />

  <div v-if="loading" class="flex justify-center items-center py-8">
    <span class="loading loading-spinner loading-lg"></span>
  </div>

  <div v-else class="overflow-x-auto">
    <table class="table table-xs">
      <!-- Table head -->
      <thead>
        <tr class="">
          <th class=""></th>
          <th class="sticky left-0 cursor-pointer select-none bg-base-200/50" @click="toggleSort('id')">
            id <span :class="sortField !== 'id' && 'invisible'">{{ sortDirection === "asc" ? "\u25B2" : "\u25BC" }}</span>
          </th>
          <th class="">schedule</th>
          <th class="">command</th>
          <th class="">run id</th>
          <th class="cursor-pointer select-none bg-base-200/50" @click="toggleSort('run_status')">
            run status <span :class="sortField !== 'run_status' && 'invisible'">{{ sortDirection === "asc" ? "\u25B2" : "\u25BC" }}</span>
          </th>
          <th class="cursor-pointer select-none bg-base-200/50" @click="toggleSort('running_time')">
            running time <span :class="sortField !== 'running_time' && 'invisible'">{{ sortDirection === "asc" ? "\u25B2" : "\u25BC" }}</span>
          </th>
          <th class="cursor-pointer select-none bg-base-200/50" @click="toggleSort('run_updated')">
            run updated <span :class="sortField !== 'run_updated' && 'invisible'">{{ sortDirection === "asc" ? "\u25B2" : "\u25BC" }}</span>
          </th>
        </tr>
      </thead>

      <!-- Table body -->
      <tbody>
        <tr v-for="task in tasks" :key="task.id" class="hover:bg-base-200">
          <td class="">
            <input
              type="checkbox"
              class="toggle toggle-sm"
              :checked="task.active"
              @change="toggleTaskActive(task.id)"
            />
          </td>

          <td class="sticky left-0">
            <IdentifierUrl @click="gotoTask(task.id)" :id="task.id" />
          </td>

          <td>
            <span class="whitespace-nowrap">
              {{ task.schedule }}
            </span>
          </td>

          <td>
            <Command :command="task.command" />
          </td>

          <td>
            <template v-if="taskLastRun(task.id)">
              <IdentifierUrl
                @click="gotoRun(task, taskLastRun(task.id))"
                :id="taskLastRun(task.id)?.id"
              />
            </template>
          </td>

          <td class="whitespace-nowrap min-w-[10ch]">
            <template v-if="taskLastRun(task.id)">
              <RunStatus :run="taskLastRun(task.id)" />
            </template>
          </td>

          <td>
            <template v-if="taskLastRun(task.id)">
              <RunTimeDiff :run="taskLastRun(task.id)" />
            </template>
          </td>

          <td>
            <template v-if="taskLastRun(task.id)">
              <RunTimeAgo :datetime="taskLastRun(task.id)?.updated" />
            </template>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
