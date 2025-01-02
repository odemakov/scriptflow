<script setup lang="ts">
import { computed, onMounted } from "vue";

import { useNodeStore } from "@/stores/NodeStore";
import Identifier from "./Identifier.vue";
import NodeStatus from "./NodeStatus.vue";

const useNode = useNodeStore();
const nodes = computed(() => useNode.getNodes);
const nodeHost = (node: INode) => `${node.host} (${node.username})`;

onMounted(async () => {
  await useNode.fetchNodes();
});
</script>

<template>
  <div class="mx-auto p-1 md:p-2 rounded overflow-x-auto">
    <table class="table table-sm w-full mx-auto text-sm md:text-base">
      <thead>
        <tr>
          <th class="whitespace-nowrap px-2 md:px-4">id</th>
          <th class="whitespace-nowrap px-2 md:px-4">host</th>
          <th class="whitespace-nowrap px-2 md:px-4">status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="node in nodes" :key="node.id">
          <td class="p-1 md:p-2">
            <Identifier :id="node.id" />
          </td>
          <td class="p-1 md:p-2">
            <Identifier :id="nodeHost(node)" />
          </td>
          <td class="p-1 md:p-2">
            <NodeStatus :node="node" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
