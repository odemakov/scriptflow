<script setup lang="ts">
import { computed, onMounted } from 'vue'
import router from '@/router';

import { useProjectStore } from '@/stores/ProjectStore';
import IdentifierUrl from './IdentifierUrl.vue';

const useProject = useProjectStore()
const projects = computed(() => useProject.getProjects)

const gotoTasks = (projectId: string) => {
  router.push({ name: 'project', params: { projectId: projectId } })
}

onMounted(async () => {
  await useProject.fetchProjects()
})
</script>

<template>
  <div class="mx-auto p-8 rounded">
    <table class="table mx-auto">
      <thead>
        <tr>
          <th>id</th>
          <th>name</th>
          <th>config</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="project in projects" :key="project.id">
          <td>
            <IdentifierUrl :id="project.id" @click="gotoTasks(project.id)" />
          </td>
          <td>{{ project.name }}</td>
          <td>{{ project.config }}</td>
        </tr>
      </tbody>
    </table>

  </div>
</template>