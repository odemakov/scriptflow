<script setup lang="ts">
import { computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";

import { useToastStore } from "@/stores/ToastStore";
import { useTaskStore } from "@/stores/TaskStore";
import { useRunStore } from "@/stores/RunStore";

import IdentifierUrl from "@/components/IdentifierUrl.vue";
import Command from "@/components/Command.vue";
import RunStatus from "@/components/RunStatus.vue";
import RunTimeAgo from "@/components/RunTimeAgo.vue";
import PageTitle from "@/components/PageTitle.vue";
import { ICrumb, CRunStatus, IRun, ITask, CEntityType, EntityType } from "@/types";
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

const tasks = computed(() => {
  return props.entityType === CEntityType.node
    ? useTasks.getTasksByNode
    : useTasks.getTasksByProject;
});
const lastRuns = computed(() => useRuns.getLastRuns);
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
    } else {
      await useTasks.fetchTasksByProject(props.entityId);
    }
    useTasks.subscribe();
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
};

const fetchLastRunsAndSubscribe = async () => {
  try {
    for (const task of tasks.value) {
      await useRuns.fetchLastRuns(task.id, 1, false);
    }
    useRuns.subscribe();
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
};

onMounted(async () => {
  UpdateTitle(`${Capitalize(props.entityType)}: ${props.entityId}`);

  // unsubscribe from runs collection just in case
  useTasks.unsubscribe();
  useRuns.unsubscribe();

  // fetch tasks
  await fetchTasks();

  // for each task fetch last run
  await fetchLastRunsAndSubscribe();
});

onUnmounted(() => {
  useTasks.unsubscribe();
  useRuns.unsubscribe();
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
  const task = tasks.value.find((t: ITask) => t.id === taskId);
  if (task) {
    try {
      task.active = !task.active;
      useTasks.updateTask(task.id, { active: task.active });
    } catch (error: unknown) {
      task.active = !task.active;
      useToasts.addToast((error as Error).message, "error");
    }
  }
};

const crumbs = [{ label: props.entityId } as ICrumb];
</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle :title="pageTitle" />

  <div class="overflow-x-auto">
    <table class="table table-xs">
      <!-- Table head -->
      <thead>
        <tr class="">
          <th class=""></th>
          <th class="sticky left-0 bg-base-100">id</th>
          <th class="">schedule</th>
          <th class="">command</th>
          <th class="">run id</th>
          <th class="">run status</th>
          <th class="">running time</th>
          <th class="">run updated</th>
        </tr>
      </thead>

      <!-- Table body -->
      <tbody>
        <tr v-for="task in tasks" :key="task.id" class="">
          <td class="">
            <input
              type="checkbox"
              class="toggle toggle-sm"
              :checked="task.active"
              @change="toggleTaskActive(task.id)"
            />
          </td>

          <td class="sticky left-0 bg-base-100">
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

          <td>
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
