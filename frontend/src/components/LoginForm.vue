<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter } from "vue-router";

import { useAuthStore } from "../stores/AuthStore";
import Button from "./Button.vue";

const auth = useAuthStore();
const router = useRouter();
const email = ref("");
const password = ref("");
const errorMessage = ref("");
const loginButtonDisabled = computed(() => !email.value || !password.value);

const handleLogin = async () => {
  errorMessage.value = "";
  try {
    await auth.login(email.value, password.value);
    router.push({ name: "home" });
  } catch (error) {
    errorMessage.value = (error as any).message;
  }
};
</script>

<template>
  <div
    class="mx-auto p-4 sm:p-8 w-full max-w-[400px] rounded shadow-lg bg-neutral-content"
  >
    <h2 class="text-xl font-bold">Login</h2>
    <div class="mt-4">
      <label for="email" class="block">Email:</label>
      <input
        v-model="email"
        type="email"
        id="email"
        required
        placeholder="Enter your email"
        class="w-full px-4 py-2 border rounded mt-2"
      />
    </div>
    <div class="mt-4">
      <label for="password" class="block">Password:</label>
      <input
        v-model="password"
        type="password"
        id="password"
        required
        placeholder="Enter your password"
        class="w-full px-4 py-2 border rounded mt-2"
      />
    </div>
    <div class="mt-4 text-center">
      <button
        @click="handleLogin"
        :disabled="loginButtonDisabled"
        :class="[
          'text-white px-4 py-2 rounded-md',
          loginButtonDisabled ? 'bg-gray-400' : 'bg-gray-700',
          { 'hover:bg-gray-600': !loginButtonDisabled },
        ]"
      >
        Login
      </button>
    </div>
    <p v-if="errorMessage" class="text-error mt-4">{{ errorMessage }}</p>
  </div>
</template>
