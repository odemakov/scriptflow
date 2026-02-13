import { defineStore } from "pinia";
import { RecordSubscription } from "pocketbase";
import { computed, ref } from "vue";

import config from "@/config";
import { useAuthStore } from "./AuthStore";
import { getPocketBaseInstance } from "./AuthStore";

export const useRunStore = defineStore("runs", () => {
  const pb = getPocketBaseInstance();
  const perPage = 50;
  // lastRuns is a dictionary of last runs for each task with taskId as key
  const lastRuns = ref({} as Record<string, IRun[]>);
  const totalRuns = ref({} as Record<string, number>);
  const run = ref({} as IRun);
  // Track active subscriptions by taskId
  const activeSubscriptions = new Map<string, () => void>();

  // getters
  const getLastRuns = computed(() => lastRuns.value);
  const getTotalRuns = computed(() => totalRuns.value);
  const getRun = computed(() => run.value);

  function hasMoreRuns(taskId: string): boolean {
    return (lastRuns.value[taskId]?.length ?? 0) < (totalRuns.value[taskId] ?? 0);
  }

  // methods
  async function fetchRuns(
    taskId: string,
    options?: { limit?: number; expand_task?: boolean; more?: boolean },
  ) {
    const { limit = perPage, expand_task = true, more = false } = options || {};
    const current = more ? (lastRuns.value[taskId]?.length ?? 0) : 0;
    const page = more ? Math.floor(current / perPage) + 1 : 1;
    const records = await pb.collection(CCollectionName.runs).getList<IRun>(page, limit, {
      requestKey: more ? `${taskId}-more` : taskId,
      filter: pb.filter("task.id={:taskId}", { taskId: taskId }),
      sort: "-created",
      expand: expand_task ? "task" : "",
    });
    if (more) {
      // Deduplicate: subscription may have prepended runs since the last fetch,
      // shifting server-side pagination and causing overlap.
      const existingIds = new Set(lastRuns.value[taskId]?.map((r) => r.id));
      const newItems = records.items.filter((r) => !existingIds.has(r.id));
      lastRuns.value[taskId] = [...(lastRuns.value[taskId] || []), ...newItems];
    } else {
      lastRuns.value[taskId] = records.items;
    }
    totalRuns.value[taskId] = records.totalItems;
  }
  async function fetchLatestRuns(taskIds: string[]) {
    if (taskIds.length === 0) return;

    const auth = useAuthStore();
    const url = `${config.baseUrl}api/scriptflow/runs/latest?taskIds=${taskIds.join(",")}`;
    const response = await fetch(url, {
      headers: { Authorization: `${auth.token}` },
    });
    if (!response.ok) {
      throw new Error(`Failed to fetch latest runs: ${response.statusText}`);
    }
    const data: Record<string, IRun> = await response.json();
    for (const [taskId, run] of Object.entries(data)) {
      lastRuns.value[taskId] = [run];
    }
  }

  async function fetchRun(runId: string) {
    const record = await pb.collection(CCollectionName.runs).getOne<ITask>(runId, {
      expand: "task",
    });
    run.value = record;
  }

  async function subscribe(options?: { taskId?: string; projectId?: string; nodeId?: string }) {
    const { taskId, projectId, nodeId } = options || {};

    // Build subscription key
    const key = taskId ? taskId : projectId ? `project:${projectId}` : nodeId ? `node:${nodeId}` : "all";

    // Prevent duplicate subscriptions
    if (activeSubscriptions.has(key)) {
      return;
    }

    // Reserve this key immediately to prevent race conditions
    activeSubscriptions.set(key, () => {});

    // Build filter
    let filter = "";
    if (taskId) {
      filter = pb.filter("task.id={:taskId}", { taskId });
    } else if (projectId) {
      filter = pb.filter("task.project.id={:projectId}", { projectId });
    } else if (nodeId) {
      filter = pb.filter("task.node.id={:nodeId}", { nodeId });
    }

    const unsubscribeFn = await pb
      .collection(CCollectionName.runs)
      .subscribe("*", (data: RecordSubscription) => {
        if (
          data.record?.collectionName == CCollectionName.runs &&
          (data.action == "create" || data.action == "update")
        ) {
          const run = {
            id: data.record.id,
            created: data.record.created,
            updated: data.record.updated,
            status: data.record.status,
            command: data.record.command,
            connection_error: data.record.connection_error,
            exit_code: data.record.exit_code,
          } as IRun;
          if (data.action == "update") {
            _updateStoreRun(data.record.task, run);
          } else if (data.action == "create") {
            _addStoreRun(data.record.task, run);
          }
        }
      }, filter ? { filter } : undefined);

    // Replace placeholder with actual unsubscribe function
    activeSubscriptions.set(key, unsubscribeFn);
  }

  function unsubscribe(options?: { taskId?: string; projectId?: string; nodeId?: string }) {
    const { taskId, projectId, nodeId } = options || {};
    const key = taskId ? taskId : projectId ? `project:${projectId}` : nodeId ? `node:${nodeId}` : "all";
    const unsubscribeFn = activeSubscriptions.get(key);
    if (unsubscribeFn) {
      unsubscribeFn();
      activeSubscriptions.delete(key);
    }
  }

  // private methods
  function _updateStoreRun(taskId: string, updatedRun: IRun) {
    if (taskId in lastRuns.value) {
      lastRuns.value[taskId] = lastRuns.value[taskId].map((run) => {
        if (run.id === updatedRun.id) {
          // update existing one
          return {
            ...run,
            ...updatedRun,
          };
        }
        return run;
      });
    }
  }
  function _addStoreRun(taskId: string, newRun: IRun) {
    // Initialize array for new taskIds
    if (!(taskId in lastRuns.value)) {
      lastRuns.value[taskId] = [];
    }
    // Check if run already exists to prevent duplicates
    const exists = lastRuns.value[taskId].some((run) => run.id === newRun.id);
    if (!exists) {
      lastRuns.value[taskId].unshift(newRun);
      lastRuns.value[taskId] = lastRuns.value[taskId].slice(0, Math.max(lastRuns.value[taskId].length, perPage));
    }
  }
  function getConsecutiveFailureCount(taskId: string) {
    const errorTypes = [CRunStatus.error, CRunStatus.internal_error];
    const runs = lastRuns.value[taskId] || [];

    if (runs.length === 0 || !errorTypes.includes(runs[0].status)) {
      return 0;
    }

    let count = 0;
    for (const run of runs) {
      if (errorTypes.includes(run.status)) {
        count++;
      } else {
        break;
      }
    }
    return count;
  }

  return {
    fetchLatestRuns,
    fetchRun,
    fetchRuns,
    getConsecutiveFailureCount,
    getLastRuns,
    getTotalRuns,
    getRun,
    hasMoreRuns,
    subscribe,
    unsubscribe,
  };
});
