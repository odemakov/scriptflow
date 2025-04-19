<script setup lang="ts">
import { onMounted } from "vue";
import { useRoute } from "vue-router";
import { useToastStore } from "@/stores/ToastStore";
import { useProjectStore } from "@/stores/ProjectStore";
import TaskListView from "@/components/TaskListView.vue";

const route = useRoute();
const useToasts = useToastStore();
const useProjects = useProjectStore();

const projectId = Array.isArray(route.params.projectId)
  ? route.params.projectId[0]
  : route.params.projectId;

const fetchProject = async () => {
  try {
    await useProjects.fetchProject(projectId);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
};

onMounted(async () => {
  // fetch project
  await fetchProject();
});
</script>

<template>
  <TaskListView
    :entity-id="projectId"
    entity-type="project"
    page-title="Project tasks"
  />
</template>
