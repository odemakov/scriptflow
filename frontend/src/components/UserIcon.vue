<script setup lang="ts">
import { computed } from "vue";
const props = defineProps<{ username: string }>();
const un = computed(() => {
  return props.username.slice(0, 2).toUpperCase();
});
</script>
<template>
  <div
    tabindex="0"
    role="button"
    class="btn btn-ghost btn-circle avatar bg-slate-200 text-lg"
  >
    {{ un }}
  </div>
</template>
