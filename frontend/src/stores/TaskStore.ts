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
  // Track active subscriptions by key (e.g., "project:abc123" or "node:xyz789")
  const activeSubscriptions = new Map<string, () => void>();

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

  async function toggleTaskActive(taskId: string) {
    // Find task in all possible arrays and toggle optimistically
    const taskArrays = [
      tasks.value,
      tasksByNode.value,
      tasksByProject.value,
      [task.value],
    ];
    let foundTask: ITask | null = null;

    for (const taskArray of taskArrays) {
      const taskInArray = taskArray.find((t: ITask) => t.id === taskId);
      if (taskInArray) {
        foundTask = taskInArray;
        taskInArray.active = !taskInArray.active;
      }
    }

    if (foundTask) {
      try {
        await updateTask(taskId, { active: foundTask.active });
      } catch (error: unknown) {
        // Rollback on error - toggle back in all arrays
        for (const taskArray of taskArrays) {
          const taskInArray = taskArray.find((t: ITask) => t.id === taskId);
          if (taskInArray) {
            taskInArray.active = !taskInArray.active;
          }
        }
        throw error; // Re-throw so components can handle toast
      }
    }
  }
  async function subscribe(options?: { projectId?: string; nodeId?: string }) {
    const { projectId, nodeId } = options || {};

    // Build subscription key
    const key = projectId ? `project:${projectId}` : nodeId ? `node:${nodeId}` : "all";

    // Prevent duplicate subscriptions
    if (activeSubscriptions.has(key)) {
      return;
    }

    // Reserve this key immediately to prevent race conditions
    activeSubscriptions.set(key, () => {});

    // Build filter
    let filter = "";
    if (projectId) {
      filter = pb.filter("project.id={:projectId}", { projectId });
    } else if (nodeId) {
      filter = pb.filter("node.id={:nodeId}", { nodeId });
    }

    const unsubscribeFn = await pb
      .collection(CCollectionName.tasks)
      .subscribe("*", (data: RecordSubscription) => {
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
            consecutive_failure_count: data.record.consecutive_failure_count,
          });
        }
      }, filter ? { filter } : undefined);

    // Replace placeholder with actual unsubscribe function
    activeSubscriptions.set(key, unsubscribeFn);
  }

  function unsubscribe(options?: { projectId?: string; nodeId?: string }) {
    const { projectId, nodeId } = options || {};
    const key = projectId ? `project:${projectId}` : nodeId ? `node:${nodeId}` : "all";
    const unsubscribeFn = activeSubscriptions.get(key);
    if (unsubscribeFn) {
      unsubscribeFn();
      activeSubscriptions.delete(key);
    }
  }
  // private functions
  function _updateStoredTask(taskId: string, updatedData: Object) {
    // Update all task arrays
    const taskArrays = [
      tasks.value,
      tasksByNode.value,
      tasksByProject.value,
    ];

    for (const taskArray of taskArrays) {
      const taskIndex = taskArray.findIndex((t: ITask) => t.id === taskId);
      if (taskIndex !== -1) {
        taskArray[taskIndex] = {
          ...taskArray[taskIndex],
          ...updatedData,
        };
      }
    }

    // Update single task if it matches
    if (task.value.id === taskId) {
      task.value = {
        ...task.value,
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
    toggleTaskActive,
  };
});
