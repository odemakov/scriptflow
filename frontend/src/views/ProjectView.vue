<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { RecordSubscription } from "pocketbase";

import { getPocketBaseInstance } from "@/stores/AuthStore";
import { useToastStore } from '@/stores/ToastStore';
import { useTaskStore } from '@/stores/TaskStore';
import { useRunStore } from '@/stores/RunStore';

import Identifier from '@/components/Identifier.vue';
import IdentifierUrl from '@/components/IdentifierUrl.vue';
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

const projectSlug = Array.isArray(route.params.projectSlug) ? route.params.projectSlug[0] : route.params.projectSlug
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
    await useProjects.fetchProject(projectSlug)
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
}

const fetchTasksAndSubsribe = async () => {
  try {
    await useTasks.fetchTasks(projectSlug)
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

const gotoTask = (taskSlug: string) => {
  router.push({ name: 'task', params: { projectSlug: projectSlug, taskSlug: taskSlug } })
}

const gotoRun = (run: IRun) => {
  if (run.status === CRunStatus.started) {
    router.push({ name: 'task-log', params: { projectSlug: projectSlug, taskSlug: run.expand.task.slug } })
  } else {
    router.push({ name: 'run', params: { id: run.id } })
  }
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
    <table class="table table-xs">
      <!-- Table head -->
      <thead>
        <tr class="">
          <th class=""></th>
          <th class="">id</th>
          <th class="">slug</th>
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
            <Identifier :id="task.id" />
          </td>

          <td>
            <IdentifierUrl @click="gotoTask(task.slug)" :id="task.slug" />
          </td>

          <td>
            <span class="whitespace-nowrap">
              {{ task.schedule }}
            </span>
          </td>

          <td>
            <Command :command="task.command" />
          </td>

          <td>
            <template v-if="taskLastRun(task.id)">
              <IdentifierUrl @click="gotoRun(taskLastRun(task.id))" :id="taskLastRun(task.id)?.id" />
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