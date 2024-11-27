import { ref, computed } from "vue";
import { defineStore } from "pinia";

import { getPocketBaseInstance } from "./pocketbase";

export const useTaskStore = defineStore("tasks", () => {
  const pb = getPocketBaseInstance();
  const tasks = ref([] as ITask[]);
  const task = ref({} as ITask);

  // getters
  const getTasks = computed(() => tasks.value);
  const getTask = computed(() => task.value);

  // methods
  async function fetchTasks(projectId: string) {
    const records = await pb
      .collection(CCollectionName.tasks)
      .getList<ITask>(1, 100, {
        expand: "node,project",
        sort: "-created",
        filter: pb.filter("project = {:projectId}", { projectId: projectId }),
      });
    tasks.value = records.items;
  }
  async function fetchTask(taskId: string) {
    const record = await pb
      .collection(CCollectionName.tasks)
      .getOne<ITask>(taskId, {
        expand: "node,project",
      });
    task.value = record;
  }
  async function updateTask(taskId: string, updatedData: Object) {
    await pb.collection(CCollectionName.tasks).update(taskId, updatedData);
    updateStoredTask(taskId, updatedData);
  }
  function updateStoredTask(taskId: string, updatedData: Object) {
    // update state task
    const taskIndex: number = tasks.value.findIndex(
      (task: ITask) => task.id === taskId
    );
    if (taskIndex !== -1) {
      tasks.value[taskIndex] = {
        ...tasks.value[taskIndex],
        ...updatedData,
      };
    }
  }
  return {
    getTasks,
    getTask,
    fetchTasks,
    fetchTask,
    updateTask,
    updateStoredTask,
  };
});
