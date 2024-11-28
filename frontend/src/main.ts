import { createApp } from "vue";
import { createPinia } from "pinia";

import "./style.css";
import router from "./router";
import App from "./App.vue";

import { useAuthStore } from "@/stores/AuthStore";

const app = createApp(App);

app.use(createPinia());

const useAuth = useAuthStore();
router.beforeEach((to) => {
  if (to.meta.requireAuth && !useAuth.isAuthenticated) {
    router.push({ name: "home" });
    return false;
  }
  return true;
});
app.use(router);

app.mount("#app");
