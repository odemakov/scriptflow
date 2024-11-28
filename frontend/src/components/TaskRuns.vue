<script setup lang="ts">
import { watch, computed } from 'vue';
import { useRouter } from 'vue-router'
import Command from './Command.vue';
import { useRunStore } from '@/stores/RunStore';
import { useToastStore } from '@/stores/ToastStore';
import IdentifierUrl from './IdentifierUrl.vue';
import RunStatus from './RunStatus.vue';
import RunTimeAgo from './RunTimeAgo.vue';
import RunTimeDiff from './RunTimeDiff.vue';

const props = defineProps<{
  task: ITask
}>()

const router = useRouter()
const useToasts = useToastStore()
const useRuns = useRunStore()
const lastRuns = computed(() => useRuns.getLastRuns[props.task.id])

const gotoRun = (run: IRun) => {
  if (run.status === CRunStatus.started) {
    router.push({ name: 'task-log', params: { id: props.task.id } })
  } else {
    router.push({ name: 'run', params: { id: run.id } })
  }
}

watch(() => props.task, async () => {
  try {
    await useRuns.fetchTaskLastRuns(props.task.id)
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
})
</script>

<template>
  <div class="overflow-x-auto">
    <table class="table table-xs">
      <!-- Table head -->
      <thead>
        <tr class="">
          <th class="">id</th>
          <th class="">command</th>
          <th class="">status</th>
          <th class="">exit code</th>
          <th class="">error</th>
          <th class="">running time</th>
          <th class="">created</th>
          <th class="">updated</th>
        </tr>
      </thead>

      <!-- Table body -->
      <tbody>
        <tr v-for="run in lastRuns" :key="run.id" class="">
          <td>
            <IdentifierUrl @click="gotoRun(run)" :id="run.id" />
          </td>
          <td>
            <Command :command="run.command" />
          </td>
          <td>
            <RunStatus :run="run" />
          </td>
          <td>
            {{ run.exit_code }}
          </td>
          <td>
            <div v-if="run.connection_error" class="bg-error/20 p-1 rounded-md">
              {{ run.connection_error }}
            </div>
          </td>
          <td>
            <RunTimeDiff :run="run" />
          </td>
          <td>
            <RunTimeAgo :datetime="run.created" />
          </td>
          <td>
            <RunTimeAgo :datetime="run.updated" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>