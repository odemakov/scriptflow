<script setup lang="ts">
import { onBeforeUnmount, watch, computed, ref } from "vue";
import { useRouter } from "vue-router";
import { useRunStore } from "@/stores/RunStore";
import { useToastStore } from "@/stores/ToastStore";
import { isAutoCancelError } from "@/lib/helpers";
import IdentifierUrl from "./IdentifierUrl.vue";
import RunStatus from "./RunStatus.vue";
import RunTimeAgo from "./RunTimeAgo.vue";
import RunTimeDiff from "./RunTimeDiff.vue";

const props = defineProps<{
  task: ITask;
  projectId?: string;
  nodeId?: string;
}>();

const router = useRouter();
const useToasts = useToastStore();
const useRuns = useRunStore();
const lastRuns = computed(() => useRuns.getLastRuns[props.task.id]);
const loading = ref(true);

const gotoRun = (run: IRun) => {
  // Determine base name and params based on run status
  const isRunning = run.status === CRunStatus.started;
  const baseName = isRunning ? "task-tail" : "task-run";
  let routeName = baseName;
  let params: Record<string, string> = { taskId: props.task.id };

  // Add context-specific prefix and params
  if (props.projectId) {
    routeName = `project-${baseName}`;
    params.projectId = props.projectId;
  } else if (props.nodeId) {
    routeName = `node-${baseName}`;
    params.nodeId = props.nodeId;
  }

  // Add run ID for completed runs
  if (!isRunning) {
    params.id = run.id;
  }

  // Navigate to the route
  router.push({ name: routeName, params });
};

let previousTaskId: string | null = null;

watch(
  () => props.task,
  async () => {
    if (!props.task?.id) return;

    loading.value = true;

    // Unsubscribe from previous task if changed
    if (previousTaskId && previousTaskId !== props.task.id) {
      useRuns.unsubscribe({ taskId: previousTaskId });
    }

    try {
      await useRuns.fetchLastRuns(props.task.id);
      await useRuns.subscribe({ taskId: props.task.id });
      previousTaskId = props.task.id;
    } catch (error: unknown) {
      if (!isAutoCancelError(error)) {
        useToasts.addToast((error as Error).message, "error");
      }
    } finally {
      loading.value = false;
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  if (previousTaskId) {
    useRuns.unsubscribe({ taskId: previousTaskId });
  }
});
</script>

<template>
  <div v-if="loading" class="flex justify-center items-center py-8">
    <span class="loading loading-spinner loading-lg"></span>
  </div>

  <div v-else class="overflow-x-auto">
    <table class="table table-xs">
      <!-- Table head -->
      <thead>
        <tr class="">
          <th class="">id</th>
          <th class="">status</th>
          <th class="">exit code</th>
          <th class="">error</th>
          <th class="">running time</th>
          <th class="">created</th>
          <th class="">updated</th>
        </tr>
      </thead>

      <!-- Table body -->
      <tbody>
        <tr v-for="run in lastRuns" :key="run.id" class="hover:bg-base-200">
          <td>
            <IdentifierUrl @click="gotoRun(run)" :id="run.id" />
          </td>
          <td>
            <RunStatus :run="run" />
          </td>
          <td>
            {{ run.exit_code }}
          </td>
          <td>
            <div v-if="run.connection_error" class="bg-error/20 p-1 rounded-md">
              {{ run.connection_error }}
            </div>
          </td>
          <td>
            <RunTimeDiff :run="run" />
          </td>
          <td>
            <RunTimeAgo :datetime="run.created" />
          </td>
          <td>
            <RunTimeAgo :datetime="run.updated" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
