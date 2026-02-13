import { ref, computed } from "vue";
import { defineStore } from "pinia";

import { getPocketBaseInstance } from "./AuthStore";

export const useProjectStore = defineStore("projects", () => {
  const pb = getPocketBaseInstance();
  const projects = ref([] as IProject[]);
  const project = ref({} as IProject);

  // getters
  const getProjects = computed(() => projects.value);
  const getProject = computed(() => project.value);

  // methods
  async function fetchProjects() {
    projects.value = await pb
      .collection(CCollectionName.projects)
      .getFullList<IProject>({
        sort: "-created",
      });
  }

  async function fetchProject(projectId: string) {
    const record = await pb
      .collection(CCollectionName.projects)
      .getFirstListItem<IProject>(`id="${projectId}"`);
    project.value = record;
  }

  return {
    getProjects,
    getProject,
    fetchProjects,
    fetchProject,
  };
});
