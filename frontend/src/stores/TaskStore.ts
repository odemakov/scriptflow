import { ref, computed } from "vue";
import { defineStore } from "pinia";
import { RecordSubscription } from "pocketbase";

import { getPocketBaseInstance } from "./AuthStore";

export const useTaskStore = defineStore("tasks", () => {
  const pb = getPocketBaseInstance();
  const tasks = ref([] as ITask[]);
  const task = ref({} as ITask);

  // getters
  const getTasks = computed(() => tasks.value);
  const getTask = computed(() => task.value);

  // methods
  async function fetchTasks(projectSlug: string) {
    const records = await pb
      .collection(CCollectionName.tasks)
      .getList<ITask>(1, 100, {
        expand: "node,project",
        sort: "-active,-created",
        filter: pb.filter("project.slug = {:slug}", { slug: projectSlug }),
      });
    tasks.value = records.items;
  }
  async function fetchTask(taskSlug: string) {
    const record = await pb
      .collection(CCollectionName.tasks)
      .getFirstListItem<IProject>(`slug="${taskSlug}"`, {
        expand: "node,project",
      });
    task.value = record;
  }
  async function updateTask(taskId: string, updatedData: Object) {
    await pb.collection(CCollectionName.tasks).update(taskId, updatedData);
  }
  function subscribe() {
    pb.collection(CCollectionName.tasks).subscribe(
      "*",
      (data: RecordSubscription) => {
        if (
          data.record?.collectionName == CCollectionName.tasks &&
          (data.action == "create" || data.action == "update")
        ) {
          _updateStoredTask(data.record.id, {
            id: data.record.id,
            updated: data.record.updated,
            name: data.record.name,
            command: data.record.command,
            schedule: data.record.schedule,
            node: data.record.node,
            project: data.record.project,
            active: data.record.active,
            prepend_datetime: data.record.prepend_datetime,
          });
        }
      }
    );
  }
  function unsubscribe() {
    pb.collection(CCollectionName.tasks).unsubscribe();
  }
  // private functions
  function _updateStoredTask(taskId: string, updatedData: Object) {
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
    subscribe,
    unsubscribe,
    fetchTask,
    updateTask,
    fetchTasks,
  };
});
