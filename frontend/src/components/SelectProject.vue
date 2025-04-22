<script setup lang="ts">
import { computed, onMounted } from "vue";
import router from "@/router";

import { useProjectStore } from "@/stores/ProjectStore";
import IdentifierUrl from "@/components/IdentifierUrl.vue";

const useProject = useProjectStore();
const projects = computed(() => useProject.getProjects);

const gotoProject = (projectId: string) => {
  router.push({ name: "project", params: { projectId: projectId } });
};

onMounted(async () => {
  await useProject.fetchProjects();
});
</script>

<template>
  <div class="mx-auto p-1 md:p-2 rounded overflow-x-auto">
    <table class="table table-sm w-full mx-auto text-sm md:text-base">
      <thead class="">
        <tr>
          <th class="whitespace-nowrap px-2 md:px-4">id</th>
          <th class="whitespace-nowrap px-2 md:px-4">name</th>
          <th class="whitespace-nowrap px-2 md:px-4">config</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="project in projects" :key="project.id" class="hover:bg-base-200">
          <td class="p-1 md:p-2">
            <IdentifierUrl :id="project.id" @click="gotoProject(project.id)" />
          </td>
          <td class="p-1 md:p-2">{{ project.name }}</td>
          <td class="p-1 md:p-2 text-xs">{{ project.config }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
