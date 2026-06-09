<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import router from "@/router";

import { useProjectStore } from "@/stores/ProjectStore";
import IdentifierUrl from "@/components/IdentifierUrl.vue";
import SettingsIcon from "@/components/icons/SettingsIcon.vue";

const useProject = useProjectStore();
const projects = computed(() => useProject.getProjects);

const selectedConfig = ref<Record<string, unknown> | null>(null);
const selectedProjectName = ref("");
const dialogRef = ref<HTMLDialogElement | null>(null);

const gotoProject = (projectId: string) => {
  router.push({ name: "project", params: { projectId: projectId } });
};

const openConfig = (name: string, config: Record<string, unknown> | undefined) => {
  selectedProjectName.value = name;
  selectedConfig.value = config ?? null;
  dialogRef.value?.showModal();
};

const closeConfig = () => {
  dialogRef.value?.close();
};

const formattedConfig = computed(() => JSON.stringify(selectedConfig.value ?? {}, null, 2));

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
        </tr>
      </thead>
      <tbody>
        <tr v-for="project in projects" :key="project.id" class="hover:bg-base-200">
          <td class="p-1 md:p-2">
            <IdentifierUrl :id="project.id" @click="gotoProject(project.id)" />
          </td>
          <td class="p-1 md:p-2">
            <span>{{ project.name }}</span>
            <button
              v-if="project.config"
              class="btn btn-ghost btn-xs ml-1"
              title="View settings"
              @click="openConfig(project.name, project.config)"
            >
              <SettingsIcon />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>

  <dialog ref="dialogRef" class="modal">
    <div class="modal-box max-w-2xl">
      <h3 class="font-bold text-lg mb-4">{{ selectedProjectName }} settings</h3>
      <pre class="bg-base-200 rounded p-4 text-xs overflow-auto max-h-96">{{ formattedConfig }}</pre>
      <div class="modal-action">
        <button class="btn" @click="closeConfig">Close</button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
</template>
