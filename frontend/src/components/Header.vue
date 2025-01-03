<script setup lang="ts">
import { onMounted, computed } from "vue";
import router from "@/router";
import { useAuthStore } from "@/stores/AuthStore";
import UserIcon from "./UserIcon.vue";
import config from "@/config";

const auth = useAuthStore();

onMounted(() => {
  auth.fetchUser();
});

const pbAdminUrl = computed(() => `${config.baseUrl}_`);
const handleLogout = async () => {
  auth.logout();
  router.push({ name: "home" });
};
</script>

<template>
  <div class="navbar bg-neutral-content">
    <div class="flex-1">
      <a class="btn btn-ghost text-xl bg-slate-200" href="./">ScriptFlow</a>
    </div>
    <div v-if="auth.isAuthenticated" class="flex-none gap-2">
      <div class="dropdown dropdown-end">
        <UserIcon :username="auth.user?.username" />
        <ul
          tabindex="0"
          class="menu menu-sm dropdown-content bg-base-100 rounded-box z-[1] mt-3 w-52 p-2 shadow"
        >
          <li><a :href="pbAdminUrl">PocketBase admin</a></li>
          <li><a @click="handleLogout">Logout</a></li>
        </ul>
      </div>
    </div>
  </div>
</template>
