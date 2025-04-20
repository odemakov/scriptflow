import { defineStore } from "pinia";
import { RecordSubscription } from "pocketbase";
import { computed, ref } from "vue";

import { getPocketBaseInstance } from "./AuthStore";

export const useTaskStore = defineStore("tasks", () => {
  const pb = getPocketBaseInstance();
  const task = ref({} as ITask);
  const tasks = ref([] as ITask[]);
  const tasksByNode = ref([] as ITask[]);
  const tasksByProject = ref([] as ITask[]);

  // getters
  const getTask = computed(() => task.value);
  const getTasks = computed(() => tasks.value);
  const getTasksByNode = computed(() => tasksByNode.value);
  const getTasksByProject = computed(() => tasksByProject.value);

  // methods
  async function fetchTasksByProject(projectId: string) {
    const records = await pb.collection(CCollectionName.tasks).getList<ITask>(1, 200, {
      expand: "node,project",
      sort: "-active,-created",
      filter: pb.filter("project.id={:id}", { id: projectId }),
    });
    tasksByProject.value = records.items;
  }
  async function fetchTasksByNode(nodeId: string) {
    const records = await pb.collection(CCollectionName.tasks).getList<ITask>(1, 200, {
      expand: "node,project",
      sort: "-active,-created",
      filter: pb.filter("node.id={:id}", { id: nodeId }),
    });
    tasksByNode.value = records.items;
  }
  async function fetchTasks() {
    const records = await pb.collection(CCollectionName.tasks).getList<ITask>(1, 200, {
      sort: "id",
    });
    tasks.value = records.items;
  }
  async function fetchTask(taskId: string) {
    const record = await pb
      .collection(CCollectionName.tasks)
      .getFirstListItem<IProject>(`id="${taskId}"`, {
        expand: "node,project",
      });
    task.value = record;
  }
  async function updateTask(taskId: string, updatedData: Object) {
    await pb.collection(CCollectionName.tasks).update(taskId, updatedData);
  }
  function subscribe() {
    pb.collection(CCollectionName.tasks).subscribe("*", (data: RecordSubscription) => {
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
    });
  }
  function unsubscribe() {
    pb.collection(CCollectionName.tasks).unsubscribe();
  }
  // private functions
  function _updateStoredTask(taskId: string, updatedData: Object) {
    // update state task
    const taskIndex: number = tasks.value.findIndex(
      (task: ITask) => task.id === taskId,
    );
    if (taskIndex !== -1) {
      tasks.value[taskIndex] = {
        ...tasks.value[taskIndex],
        ...updatedData,
      };
    }
  }
  return {
    fetchTask,
    fetchTasks,
    fetchTasksByNode,
    fetchTasksByProject,
    getTask,
    getTasks,
    getTasksByNode,
    getTasksByProject,
    subscribe,
    unsubscribe,
    updateTask,
  };
});
