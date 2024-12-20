<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

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
import { ICrumb } from '@/types';
import Breadcrumbs from '@/components/Breadcrumbs.vue';
import RunTimeDiff from '@/components/RunTimeDiff.vue';

const router = useRouter()
const route = useRoute()
const useToasts = useToastStore()
const useProjects = useProjectStore()
const useTasks = useTaskStore()
const useRuns = useRunStore()

const projectId = Array.isArray(route.params.projectId) ? route.params.projectId[0] : route.params.projectId
const tasks = computed(() => useTasks.getTasks)
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
    useTasks.subscribe()
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
}

const fetchLastRunsAndSubscribe = async () => {
  try {
    for (const task of tasks.value) {
      await useRuns.fetchLastRuns(task.id, 1, false)
    }
    useRuns.subscribe()
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
}

onMounted(async () => {
  // unsubscribe from runs collection just in case
  useTasks.unsubscribe()
  useRuns.unsubscribe()

  // fetch project
  await fetchProject()

  // fetch tasks
  await fetchTasksAndSubsribe()

  // for each task fetch last run
  await fetchLastRunsAndSubscribe()
})

onUnmounted(() => {
  useTasks.unsubscribe()
  useRuns.unsubscribe()
})

const gotoTask = (taskSlug: string) => {
  router.push({ name: 'task', params: { projectId: projectId, taskSlug: taskSlug } })
}

const gotoRun = (run: IRun) => {
  if (run.status === CRunStatus.started) {
    router.push({ name: 'task-log', params: { projectId: projectId, taskSlug: run.expand.task.slug } })
  } else {
    router.push({ name: 'run', params: { projectId: projectId, taskSlug: run.expand.task.slug, id: run.id } })
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

const crumbs = [
  { label: projectId } as ICrumb,
]

</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle title="Project tasks" />

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
          <th class="">running time</th>
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
              <RunTimeDiff :run="taskLastRun(task.id)" />
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