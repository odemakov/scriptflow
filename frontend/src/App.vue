<script setup lang="ts">
import { watch } from "vue";
import { useRouter } from "vue-router";
import Header from "@/components/Header.vue";
import Toasts from "@/components/Toasts.vue";
import { useAuthStore } from "@/stores/AuthStore";
import { useToastStore } from "@/stores/ToastStore";

const auth = useAuthStore();
const toasts = useToastStore();
const router = useRouter();

watch(
  () => auth.isAuthenticated,
  (val, oldVal) => {
    if (oldVal && !val) {
      toasts.addToast("Session expired, please log in", "warning");
      router.push({ name: "home" });
    }
  },
);
</script>

<template>
  <Header />
  <div class="p-1 md:p-2 w-full">
    <div
      class="border-base-300 bg-base-100 rounded-lg items-center justify-center gap-2 overflow-x-hidden bg-cover bg-top p-1 md:p-2 [border-width:var(--tab-border)]"
    >
      <RouterView />
    </div>
  </div>
  <Toasts />
</template>
