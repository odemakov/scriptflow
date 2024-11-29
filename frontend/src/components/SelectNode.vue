<script setup lang="ts">
import { computed, onMounted } from 'vue'

import { useNodeStore } from '@/stores/NodeStore';
import Identifier from './Identifier.vue';
import NodeStatus from './NodeStatus.vue';

const useNode = useNodeStore()
const nodes = computed(() => useNode.getNodes)

onMounted(async () => {
  await useNode.fetchNodes()
})
</script>

<template>
  <div class="mx-auto p-8 rounded">
    <table class="table mx-auto">
      <thead>
        <tr>
          <th>id</th>
          <th>host</th>
          <th>username</th>
          <th>status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="node in nodes" :key="node.id">
          <td>
            <Identifier :id="node.id" />
          </td>
          <td>
            <Identifier :id="node.host" />
          </td>
          <td>{{ node.username }}</td>
          <td>
            <NodeStatus :node="node" />
          </td>
        </tr>
      </tbody>
    </table>

  </div>
</template>