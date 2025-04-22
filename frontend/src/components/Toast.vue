<script setup lang="ts">
import { ref } from "vue";
import InfoIcon from "./icons/InfoIcon.vue";
import SuccessIcon from "./icons/SuccessIcon.vue";
import WarningIcon from "./icons/WarningIcon.vue";
import ErrorIcon from "./icons/ErrorIcon.vue";

const props = defineProps({
  toast: Object as () => Toast,
});
const visible = ref(true);
const closeToast = () => {
  visible.value = false;
};
</script>

<template>
  <div
    class="fixed bottom-4 left-1/2 transform -translate-x-1/2 flex items-center w-full max-w-xs p-4 text-base-content bg-base-100 rounded-lg shadow-md border border-base-300"
    role="alert"
    v-show="visible"
  >
    <SuccessIcon v-if="props.toast.type === 'success'" />
    <InfoIcon v-else-if="props.toast.type === 'icon'" />
    <WarningIcon v-else-if="props.toast.type === 'warning'" />
    <ErrorIcon v-else-if="props.toast.type === 'error'" />
    <div class="ms-3 text-sm font-normal">{{ props.toast.message }}</div>
    <button
      type="button"
      class="ms-auto -mx-1.5 -my-1.5 bg-base-100 text-base-content opacity-70 hover:opacity-100 rounded-lg focus:ring-2 focus:ring-base-300 p-1.5 hover:bg-base-200 inline-flex items-center justify-center h-8 w-8"
      @click="closeToast"
      aria-label="Close"
    >
      <span class="sr-only">Close</span>
      <svg
        class="w-3 h-3"
        aria-hidden="true"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 14 14"
      >
        <path
          stroke="currentColor"
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="m1 1 6 6m0 0 6 6M7 7l6-6M7 7l-6 6"
        />
      </svg>
    </button>
  </div>
</template>
