import { ref, computed } from "vue";
import { defineStore } from "pinia";

export const useToastStore = defineStore("toasts", () => {
  const toasts = ref([] as Toast[]);
  // getters
  const getToasts = computed(() => toasts.value);
  // methods
  function addToast(
    message: string,
    type: "success" | "error" | "info" | "warning",
    duration: number = 4000
  ) {
    const toast: Toast = {
      fired: new Date().toISOString(),
      message: message,
      type: type,
      duration: duration,
      timeout: setTimeout(() => {
        removeToast(toast.fired);
      }, duration),
    };
    toasts.value.push(toast);
  }
  function removeToast(fired: string) {
    // remove toast by fired value
    const toast: Toast | undefined = toasts.value.find(
      (t: Toast) => t.fired === fired
    );
    if (toast) {
      clearTimeout(toast.timeout);
      toasts.value = toasts.value.filter((t: Toast) => t !== toast);
    }
  }
  return { getToasts, addToast, removeToast };
});
