import { ref, computed } from "vue";
import { defineStore } from "pinia";

import { getPocketBaseInstance } from "./AuthStore";

export const useNodeStore = defineStore("nodes", () => {
  const pb = getPocketBaseInstance();
  const nodes = ref([] as INode[]);
  const node = ref({} as INode);

  // getters
  const getNodes = computed(() => nodes.value);
  const getNode = computed(() => node.value);

  // methods
  async function fetchNodes() {
    nodes.value = await pb
      .collection(CCollectionName.nodes)
      .getFullList<INode>({
        sort: "-created",
      });
  }

  async function fetchNode(nodeId: string) {
    const record = await pb
      .collection(CCollectionName.nodes)
      .getFirstListItem<INode>(`id="${nodeId}"`);
    node.value = record;
  }

  return {
    getNodes,
    getNode,
    fetchNodes,
    fetchNode,
  };
});
