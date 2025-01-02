<script setup lang="ts">
import Header from "@/components/Header.vue";
import Toasts from "@/components/Toasts.vue";
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
