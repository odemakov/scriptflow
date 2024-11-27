<script setup lang="ts">
import { watch, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useRunStore } from '@/stores/RunStore';
import { useToastStore } from '@/stores/ToastStore';
import { emptyBack, IBack } from '@/types';
import PageTitle from '@/components/PageTitle.vue'
import RunCard from '@/components/RunCard.vue';
import RunLogTerminal from '@/components/RunLogTerminal.vue';

const useToasts = useToastStore()
const useRuns = useRunStore()
const route = useRoute()
const router = useRouter()
const runId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id

const run = computed(() => useRuns.getRun)
let back = emptyBack

onMounted(async () => {
  try {
    await useRuns.fetchRun(runId)
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
})

watch(run, (newRun) => {
  if (newRun.id) {
    back = {
      to: () => router.push({ name: 'task', params: { id: newRun.expand?.task.id } }),
      label: 'back to task'
    } as IBack
  }
}, { immediate: true })
</script>

<template>
  <PageTitle :title="`&lt;${run.id}&gt; run`" :back="back" />
  <div class="flex flex-row gap-4">
    <div class="basis-1/4">
      <RunCard :run="run" />
    </div>
    <div class="basis-3/4">
      <RunLogTerminal :run="run" />
    </div>
  </div>
</template>
