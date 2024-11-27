import { ref, computed } from "vue";
import { defineStore } from "pinia";

import { getPocketBaseInstance } from "./pocketbase";

export const useRunStore = defineStore("runs", () => {
  const pb = getPocketBaseInstance();
  // lastRuns is a dictionary of 100 last runs for each task with taskId as key
  const lastRuns = ref({} as Record<string, IRun[]>);
  const run = ref({} as IRun);

  // getters
  const getLastRuns = computed(() => lastRuns.value);
  const getRun = computed(() => run.value);

  // methods
  async function fetchTaskLastRuns(taskId: string, limit: number = 100) {
    const records = await pb
      .collection(CCollectionName.runs)
      .getList<IRun>(1, limit, {
        requestKey: taskId,
        filter: pb.filter("task.id={:taskId}", { taskId: taskId }),
        sort: "-created",
      });
    lastRuns.value[taskId] = records.items;
  }
  async function fetchRun(runId: string) {
    const record = await pb
      .collection(CCollectionName.runs)
      .getOne<ITask>(runId, {
        expand: "task",
      });
    run.value = record;
  }
  function updateStoreRun(taskId: string, updatedRun: IRun) {
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
  function addStoreRun(taskId: string, newRun: IRun) {
    lastRuns.value[taskId].unshift(newRun);
    lastRuns.value[taskId] = lastRuns.value[taskId].slice(0, 100);
  }

  return {
    getLastRuns,
    getRun,
    fetchTaskLastRuns,
    fetchRun,
    updateStoreRun,
    addStoreRun,
  };
});
