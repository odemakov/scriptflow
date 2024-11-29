import { ref } from "vue";
import { defineStore } from "pinia";

import PocketBase from "pocketbase";

// Initialize PocketBase client
const pb = new PocketBase();

export const useAuthStore = defineStore("auth", () => {
  // State variables
  const user = ref(pb.authStore.model); // PocketBase stores the current user in `pb.authStore.model`
  const isAuthenticated = ref(pb.authStore.isValid);
  const token = ref(pb.authStore.token);

  // Watch for auth state changes
  pb.authStore.onChange(() => {
    user.value = pb.authStore.model;
    isAuthenticated.value = pb.authStore.isValid;
    token.value = pb.authStore.token;
  });

  // Actions
  const login = async (email: string, password: string) => {
    const authData = await pb
      .collection("users")
      .authWithPassword(email, password);
    user.value = authData.record;
    token.value = authData.token;
    isAuthenticated.value = true;
  };

  const logout = () => {
    pb.authStore.clear(); // Clear auth data
    user.value = null;
    token.value = "";
    isAuthenticated.value = false;
  };

  const fetchUser = async () => {
    if (isAuthenticated.value && pb.authStore.model) {
      try {
        user.value = await pb.collection("users").getOne(pb.authStore.model.id);
      } catch (error) {
        console.error("Failed to fetch user details:", error);
      }
    }
  };

  return {
    user,
    isAuthenticated,
    token,
    login,
    logout,
    fetchUser,
  };
});

// Utility to access PocketBase instance directly
export const getPocketBaseInstance = () => pb;
