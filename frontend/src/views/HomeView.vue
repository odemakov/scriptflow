<script setup lang="ts">
import { onMounted } from "vue";
import { useAuthStore } from "@/stores/AuthStore";
import LoginForm from "@/components/LoginForm.vue";
import SelectProject from "@/components/SelectProject.vue";
import SelectFailedTask from "@/components/SelectFailedTask.vue";
import PageTitle from "@/components/PageTitle.vue";
import SelectNode from "@/components/SelectNode.vue";
import { CEntityType } from "@/types";
import { UpdateTitle } from "@/lib/helpers";

const auth = useAuthStore();

onMounted(async () => {
  UpdateTitle();
});
</script>

<template>
  <template v-if="auth.isAuthenticated">
    <div class="flex flex-col xl:flex-row gap-4 p-1 md:p-2">
      <div class="flex-1 sm:p-1 md:p-2 border rounded-lg">
        <PageTitle title="Tasks with failed runs" :icon="CEntityType.task" />
        <SelectFailedTask />
      </div>
      <div class="flex-1 sm:p-1 md:p-2 border rounded-lg">
        <PageTitle title="Projects" :icon="CEntityType.project" />
        <SelectProject />
      </div>
      <div class="flex-1 sm:p-1 md:p-2 border rounded-lg">
        <PageTitle title="Nodes" :icon="CEntityType.node" />
        <SelectNode />
      </div>
    </div>
  </template>
  <template v-else>
    <LoginForm />
  </template>
</template>
