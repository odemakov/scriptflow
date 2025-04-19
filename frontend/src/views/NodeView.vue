<script setup lang="ts">
import { onMounted } from "vue";
import { useRoute } from "vue-router";
import { useToastStore } from "@/stores/ToastStore";
import { useNodeStore } from "@/stores/NodeStore";
import TaskListView from "@/components/TaskListView.vue";

const route = useRoute();
const useToasts = useToastStore();
const useNodes = useNodeStore();

const nodeId = Array.isArray(route.params.nodeId)
  ? route.params.nodeId[0]
  : route.params.nodeId;

const fetchNode = async () => {
  try {
    await useNodes.fetchNode(nodeId);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
};

onMounted(async () => {
  // fetch node
  await fetchNode();
});
</script>

<template>
  <TaskListView :entity-id="nodeId" entity-type="node" page-title="Node tasks" />
</template>
