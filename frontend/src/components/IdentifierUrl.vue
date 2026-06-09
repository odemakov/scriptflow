<script setup lang="ts">
import LinkIcon from "./icons/LinkIcon.vue";

const props = defineProps<{
  id: string | undefined;
}>();
</script>
<template>
  <span
    class="bg-base-300 text-base-900 font-mono whitespace-nowrap rounded-md py-1 px-2 my-1 text-xs hover:cursor-pointer inline-flex items-center gap-1"
  >
    {{ props.id }}
    <LinkIcon />
  </span>
</template>
