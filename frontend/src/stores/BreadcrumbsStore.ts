import { ref, computed } from "vue";
import { defineStore } from "pinia";

export const useBreadcrumbsStore = defineStore("breadcrumbs", () => {
  const breadcrumbs = ref([] as IProject[]);

  // getters
  const getBreabcrumbs = computed(() => breadcrumbs.value);

  // methods
  return {
    getBreabcrumbs,
  };
});
