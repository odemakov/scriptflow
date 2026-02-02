import { defineStore } from "pinia";
import { RecordSubscription } from "pocketbase";
import { computed, ref } from "vue";

import { getPocketBaseInstance } from "./AuthStore";

export const useRunStore = defineStore("runs", () => {
  const pb = getPocketBaseInstance();
  // lastRuns is a dictionary of 100 last runs for each task with taskId as key
  const lastRuns = ref({} as Record<string, IRun[]>);
  const run = ref({} as IRun);
  // Track active subscriptions by taskId
  const activeSubscriptions = new Map<string, () => void>();

  // getters
  const getLastRuns = computed(() => lastRuns.value);
  const getRun = computed(() => run.value);

  // methods
  async function fetchLastRuns(
    taskId: string,
    limit: number = 100,
    expand_task: boolean = true,
  ) {
    const records = await pb.collection(CCollectionName.runs).getList<IRun>(1, limit, {
      requestKey: taskId,
      filter: pb.filter("task.id={:taskId}", { taskId: taskId }),
      sort: "-created",
      expand: expand_task ? "task" : "",
    });
    lastRuns.value[taskId] = records.items;
  }
  async function fetchRun(runId: string) {
    const record = await pb.collection(CCollectionName.runs).getOne<ITask>(runId, {
      expand: "task",
    });
    run.value = record;
  }

  // Fetch last N runs for multiple tasks in a single request
  async function fetchLastRunsForTasks(taskIds: string[], limitPerTask: number = 10) {
    if (taskIds.length === 0) return;

    // Build filter: task.id='id1' || task.id='id2' || ...
    const filter = taskIds.map((id) => pb.filter("task.id={:id}", { id })).join(" || ");

    // Fetch enough records to cover all tasks (worst case: limitPerTask * taskIds.length)
    const records = await pb.collection(CCollectionName.runs).getList<IRun>(1, limitPerTask * taskIds.length, {
      filter,
      sort: "-created",
      expand: "task",
    });

    // Group by taskId
    const grouped: Record<string, IRun[]> = {};
    for (const taskId of taskIds) {
      grouped[taskId] = [];
    }
    for (const run of records.items) {
      const taskId = run.expand?.task?.id || run.task;
      if (taskId && grouped[taskId] && grouped[taskId].length < limitPerTask) {
        grouped[taskId].push(run);
      }
    }

    // Update store
    for (const [taskId, runs] of Object.entries(grouped)) {
      lastRuns.value[taskId] = runs;
    }
  }
  async function subscribe(options?: { taskId?: string }) {
    const { taskId } = options || {};

    // Build subscription key
    const key = taskId || "all";

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

  function unsubscribe(options?: { taskId?: string }) {
    const { taskId } = options || {};
    const key = taskId || "all";
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
      lastRuns.value[taskId] = lastRuns.value[taskId].slice(0, 100);
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
    fetchLastRuns,
    fetchLastRunsForTasks,
    fetchRun,
    getConsecutiveFailureCount,
    getLastRuns,
    getRun,
    subscribe,
    unsubscribe,
  };
});
