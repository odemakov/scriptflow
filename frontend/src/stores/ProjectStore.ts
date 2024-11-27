import { ref, computed } from "vue";
import { defineStore } from "pinia";

import { getPocketBaseInstance } from "./pocketbase";

export const useProjectStore = defineStore("projects", () => {
  const pb = getPocketBaseInstance();
  const projects = ref([] as IProject[]);
  const project = ref({} as IProject);

  // getters
  const getProjects = computed(() => projects.value);
  const getProject = computed(() => project.value);

  // methods
  async function fetchProjects() {
    const records = await pb
      .collection(CCollectionName.projects)
      .getList<IProject>(1, 50, {
        sort: "-created",
      });
    projects.value = records.items;
  }

  async function fetchProject(projectId: string) {
    const record = await pb
      .collection(CCollectionName.projects)
      .getOne<IProject>(projectId);
    project.value = record;
  }

  return {
    getProjects,
    getProject,
    fetchProjects,
    fetchProject,
  };
});
