import { defineStore } from "pinia";
import { RecordSubscription } from "pocketbase";
import { computed, ref } from "vue";

import { getPocketBaseInstance } from "./AuthStore";

export const useRunStore = defineStore("runs", () => {
  const pb = getPocketBaseInstance();
  // lastRuns is a dictionary of 100 last runs for each task with taskId as key
  const lastRuns = ref({} as Record<string, IRun[]>);
  const run = ref({} as IRun);

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
  function subscribe() {
    pb.collection(CCollectionName.runs).subscribe("*", (data: RecordSubscription) => {
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
    });
  }
  function unsubscribe() {
    pb.collection(CCollectionName.runs).unsubscribe();
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
    if (taskId in lastRuns.value) {
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
    fetchRun,
    getConsecutiveFailureCount,
    getLastRuns,
    getRun,
    subscribe,
    unsubscribe,
  };
});
