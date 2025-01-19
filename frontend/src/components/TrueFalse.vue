<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  status: boolean;
}>();

const className = computed(() => {
  if (props.status == true) {
    return "badge badge-success bg-opacity-20";
  } else {
    return "badge badge-error bg-opacity-20";
  }
});
</script>

<template>
  <span :class="className">
    {{ status }}
  </span>
</template>
