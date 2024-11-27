<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { RecordSubscription } from "pocketbase";

import { getPocketBaseInstance } from "@/stores/pocketbase";
import { useToastStore } from '@/stores/ToastStore';
import { useTaskStore } from '@/stores/TaskStore';
import { useRunStore } from '@/stores/RunStore';

import Identifier from '@/components/Identifier.vue';
import Command from '@/components/Command.vue';
import RunStatus from '@/components/RunStatus.vue';
import RunTimeAgo from '@/components/RunTimeAgo.vue';
import PageTitle from '@/components/PageTitle.vue';
import { useProjectStore } from '@/stores/ProjectStore';
import { IBack } from '@/types';

const pb = getPocketBaseInstance()
const router = useRouter()
const route = useRoute()
const useToasts = useToastStore()
const useProjects = useProjectStore()
const useTasks = useTaskStore()
const useRuns = useRunStore()

const projectId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id
const back = {
  to: () => router.push({ name: 'home' }),
  label: 'back to projects'
} as IBack

const tasks = computed(() => useTasks.getTasks)
const project = computed(() => useProjects.getProject)
const lastRuns = computed(() => useRuns.getLastRuns)
const taskLastRun = (taskId: string) => {
  if (taskId in lastRuns.value && lastRuns.value[taskId].length > 0) {
    return lastRuns.value[taskId][0]
  } else {
    return null
  }
}

const fetchProject = async () => {
  try {
    await useProjects.fetchProject(projectId)
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
}

const fetchTasksAndSubsribe = async () => {
  try {
    await useTasks.fetchTasks(projectId)
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
  pb.collection(CCollectionName.tasks).subscribe(
    "*",
    (data: RecordSubscription) => {
      if (data.record?.collectionName == CCollectionName.tasks && (data.action == "create" || data.action == "update")) {
        useTasks.updateStoredTask(data.record.id, {
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

const fetchTaskLastRunsAndSubscribe = async () => {
  try {
    for (const task of tasks.value) {
      await useRuns.fetchTaskLastRuns(task.id, 1)
    }
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
  pb.collection(CCollectionName.runs).subscribe(
    "*",
    (data: RecordSubscription) => {
      if (data.record?.collectionName == CCollectionName.runs && (data.action == "create" || data.action == "update")) {
        const run = {
          id: data.record.id,
          updated: data.record.updated,
          status: data.record.status,
          command: data.record.command,
          connection_error: data.record.connection_error,
          exit_code: data.record.exit_code,
        } as IRun;
        if (data.action == "update") {
          useRuns.updateStoreRun(data.record.task, run)
        } else if (data.action == "create") {
          run.created = new Date().toISOString()
          run.updated = run.created
          useRuns.addStoreRun(data.record.task, run)
        }
      }
    }
  );
}

onMounted(async () => {
  // unsubscribe from runs collection
  pb.collection(CCollectionName.runs).unsubscribe()
  pb.collection(CCollectionName.tasks).unsubscribe()

  // fetch project
  await fetchProject()

  // fetch tasks
  await fetchTasksAndSubsribe()

  // for each task fetch last run
  await fetchTaskLastRunsAndSubscribe()
})

onUnmounted(() => {
  pb.collection(CCollectionName.runs).unsubscribe()
  pb.collection(CCollectionName.tasks).unsubscribe()
})

const gotoTask = (taskId: string) => {
  router.push({ name: 'task', params: { id: taskId } })
}

const gotoRun = (runId: string) => {
  router.push({ name: 'run', params: { id: runId } })
}


const toggleTaskActive = async (taskId: string) => {
  const task = tasks.value.find((t: ITask) => t.id === taskId)
  if (task) {
    try {
      task.active = !task.active
      useTasks.updateTask(task.id, { active: task.active })
    } catch (error: unknown) {
      task.active = !task.active
      useToasts.addToast(
        (error as Error).message,
        'error',
      )
    }
  }
}

</script>

<template>
  <PageTitle :title="`&lt;${project.name}&gt; project`" :back="back" />

  <div class="overflow-x-auto">
    <table class="table">
      <!-- Table head -->
      <thead>
        <tr class="">
          <th class=""></th>
          <th class="">id</th>
          <th class="">name</th>
          <th class="">schedule</th>
          <th class="">command</th>
          <th class="">run id</th>
          <th class="">run status</th>
          <th class="">run updated</th>
        </tr>
      </thead>

      <!-- Table body -->
      <tbody>
        <tr v-for="task in tasks" :key="task.id" class="">

          <td class="">
            <input type="checkbox" class="toggle toggle-sm" :checked="task.active"
              @change="toggleTaskActive(task.id)" />
          </td>

          <td>
            <Identifier @click="gotoTask(task.id)" :id="task.id" />
          </td>

          <td>
            {{ task.name }}
          </td>

          <td>
            {{ task.schedule }}
          </td>

          <td>
            <Command :command="task.command" />
          </td>

          <td>
            <template v-if="taskLastRun(task.id)">
              <Identifier @click="gotoRun(task.id)" :id="taskLastRun(task.id)?.id" />
            </template>
          </td>

          <td>
            <template v-if="taskLastRun(task.id)">
              <RunStatus :run="taskLastRun(task.id)" />
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