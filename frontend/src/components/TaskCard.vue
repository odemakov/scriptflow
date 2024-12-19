<script setup lang="ts">
import { computed, watch, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router';

import Command from './Command.vue';
import Identifier from './Identifier.vue';
import IdentifierUrl from './IdentifierUrl.vue';
import { useRunStore } from '@/stores/RunStore';
import { useTaskStore } from '@/stores/TaskStore';
import { useToastStore } from '@/stores/ToastStore';

const props = defineProps<{
  task: ITask,
}>()

const router = useRouter()
const useToasts = useToastStore()
const useRuns = useRunStore()
const useTask = useTaskStore()

const lastRuns = computed(() => useRuns.getLastRuns[props.task.id])
const lastRunStarted = computed(() => {
  if (lastRuns.value && lastRuns.value.length > 0) {
    return lastRuns.value[0].status === CRunStatus.started
  } else {
    return false
  }
})
// this variable is used to disable the run button when a run is in progress
// we can't fully rely on the last run status because it's updated with small delay
const runTaskButtonDisabled = ref(false)

watch(() => props.task, async () => {
  try {
    await useRuns.fetchLastRuns(props.task.id)
    useRuns.subscribe()
  } catch (error: unknown) {
    // useToasts.addToast(
    //   (error as Error).message,
    //   'error',
    // )
  }
})

onUnmounted(() => {
  useRuns.unsubscribe()
})

const gotoProject = () => {
  router.push({ name: 'project', params: { projectSlug: props.task?.expand?.project?.slug } })
}

const runTask = async () => {
  runTaskButtonDisabled.value = true

  const oldSchedule = props.task.schedule
  try {
    await useTask.updateTask(props.task.id, {
      ...props.task,
      schedule: '@every 1s',
    })
    runTaskButtonDisabled.value = false
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }

  // wait 2 seconds and set oldSchedule back
  setTimeout(async () => {
    try {
      await useTask.updateTask(props.task.id, {
        ...props.task,
        schedule: oldSchedule,
      })
    } catch (error: unknown) {
      useToasts.addToast(
        (error as Error).message,
        'error',
      )
    }
  }, 2000)
}

</script>

<template>
  <div class="card card-compact bg-base-100 shadow-xl">
    <div class="card-body">
      <h2 class="card-title">{{ props.task.name }}</h2>
      <button class="btn btn-xs" :disabled="runTaskButtonDisabled || lastRunStarted" @click="runTask()">Run
        once</button>
      <Command v-if="props.task.command" :command="props.task.command" />
      <table class="table table-xs">
        <tbody>
          <tr>
            <td>Project</td>
            <td>
              <IdentifierUrl :id="props.task.expand?.project?.name" @click="gotoProject()" />
            </td>
          </tr>
          <tr>
            <td>Id</td>
            <td>
              <Identifier :id="props.task.id" />
            </td>
          </tr>
          <tr>
            <td>Node</td>
            <td>{{ props.task.expand?.node?.host }}</td>
          </tr>
          <tr>
            <td>Schedule</td>
            <td>{{ props.task.schedule }}</td>
          </tr>
          <tr>
            <td>Active</td>
            <td>{{ props.task.active }}</td>
          </tr>
          <tr>
            <td>Prepend datetime</td>
            <td>{{ props.task.prepend_datetime }}</td>
          </tr>
          <tr>
            <td>Created</td>
            <td>{{ props.task.created }}</td>
          </tr>
          <tr>
            <td>Updated</td>
            <td>{{ props.task.updated }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
