<script setup lang="ts">
import { computed } from 'vue';
import { TimeAgo } from '@/lib/helpers';

const props = defineProps<{
  node: INode,
}>()

// color class based on node status
const colorClass = computed(() => {
  switch (props.node.status) {
    case CNodeStatus.online:
      return 'badge-success bg-opacity-60'
    case CNodeStatus.offline:
      return 'badge-error bg-opacity-60'
    default:
      return ''
  }
})

</script>

<template>
  <div class="flex gap-2">
    <span :class="colorClass" class="badge">
      {{ node.status }}
    </span>
    <span class="text-xs">{{ TimeAgo(props.node.updated) }} ago</span>
  </div>
</template>