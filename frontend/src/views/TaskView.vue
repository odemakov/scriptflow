<script setup lang="ts">
import { watch, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useTaskStore } from '@/stores/TaskStore';
import { useToastStore } from '@/stores/ToastStore';
import { emptyBack, IBack } from '@/types';
import PageTitle from '@/components/PageTitle.vue'
import TaskCard from '@/components/TaskCard.vue';
import TaskRuns from '@/components/TaskRuns.vue';

const useToasts = useToastStore()
const useTasks = useTaskStore()
const router = useRouter()
const route = useRoute()
const taskId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id

const task = computed(() => useTasks.getTask)
let back = emptyBack

onMounted(async () => {
  try {
    await useTasks.fetchTask(taskId)
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
})

const gotoTaskLog = () => {
  router.push({ name: 'task-log', params: { id: taskId } })
}

watch(task, (newTask) => {
  if (newTask.id) {
    back = {
      to: () => router.push({ name: 'project', params: { id: newTask.expand?.project.id } }),
      label: 'back to project'
    } as IBack
  }
}, { immediate: true })

</script>

<template>
  <PageTitle :title="`&lt;${task.name}&gt; task history`" :back="back" />
  <div class="flex flex-row gap-4">
    <div class="basis-1/4">
      <TaskCard :task="task" />
    </div>
    <div class="basis-3/4">
      <div role="tablist" class="tabs tabs-lifted">
        <a role="tab" class="tab tab-active">History</a>
        <div role="tabpanel" class="tab-content bg-base-100 border-base-300 rounded-box p-6">
          <TaskRuns :task="task" />
        </div>
        <a role="tab" class="tab" @click="gotoTaskLog()">Logs</a>
        <div role="tabpanel" class="tab-content bg-base-100 border-base-300 rounded-box p-6">
        </div>
      </div>
    </div>
  </div>
</template>
