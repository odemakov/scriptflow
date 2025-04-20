<script setup lang="ts">
import { useAuthStore } from "@/stores/AuthStore";
import LoginForm from "@/components/LoginForm.vue";
import SelectProject from "@/components/SelectProject.vue";
import SelectFailedTask from "@/components/SelectFailedTask.vue";
import PageTitle from "@/components/PageTitle.vue";
import SelectNode from "@/components/SelectNode.vue";

const auth = useAuthStore();
</script>

<template>
  <template v-if="auth.isAuthenticated">
    <div class="flex flex-col xl:flex-row gap-4 p-1 md:p-2">
      <div class="flex-1 sm:p-1 md:p-2 border rounded-lg">
        <PageTitle title="Tasks with failed runs" />
        <SelectFailedTask />
      </div>
      <div class="flex-1 sm:p-1 md:p-2 border rounded-lg">
        <PageTitle title="Projects" />
        <SelectProject />
      </div>
      <div class="flex-1 sm:p-1 md:p-2 border rounded-lg">
        <PageTitle title="Nodes" />
        <SelectNode />
      </div>
    </div>
  </template>
  <template v-else>
    <LoginForm />
  </template>
</template>
