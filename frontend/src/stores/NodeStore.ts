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
    const records = await pb
      .collection(CCollectionName.nodes)
      .getList<INode>(1, 50, {
        sort: "-created",
      });
    nodes.value = records.items;
  }

  async function fetchNode(nodeSlug: string) {
    const record = await pb
      .collection(CCollectionName.nodes)
      .getFirstListItem<INode>(`slug="${nodeSlug}"`);
    node.value = record;
  }

  return {
    getNodes,
    getNode,
    fetchNodes,
    fetchNode,
  };
});
