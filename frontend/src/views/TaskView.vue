<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useTaskStore } from '@/stores/TaskStore';
import { useToastStore } from '@/stores/ToastStore';
import { ICrumb } from '@/types';
import PageTitle from '@/components/PageTitle.vue'
import TaskCard from '@/components/TaskCard.vue';
import TaskRuns from '@/components/TaskRuns.vue';
import Breadcrumbs from '@/components/Breadcrumbs.vue';

const useToasts = useToastStore()
const useTasks = useTaskStore()
const router = useRouter()
const route = useRoute()
const taskSlug = Array.isArray(route.params.taskSlug) ? route.params.taskSlug[0] : route.params.taskSlug
const projectSlug = Array.isArray(route.params.projectSlug) ? route.params.projectSlug[0] : route.params.projectSlug

const task = computed(() => useTasks.getTask)

onMounted(async () => {
  try {
    await useTasks.fetchTask(taskSlug)
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
})

const gotoTaskLog = () => {
  router.push({ name: 'task-log', params: { projectSlug: projectSlug, taskSlug: taskSlug } })
}

const crumbs = [
  { to: () => router.push({ name: 'project', params: { projectSlug: projectSlug } }), label: projectSlug } as ICrumb,
  { label: taskSlug } as ICrumb,
]

</script>

<template>
  <Breadcrumbs :crumbs="crumbs" />
  <PageTitle title="Task history" />
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
