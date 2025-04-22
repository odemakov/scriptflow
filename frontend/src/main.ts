import { createPinia } from "pinia";
import { createApp } from "vue";

import App from "@/App.vue";
import router from "@/router";
import "./style.css";

import { useAuthStore } from "@/stores/AuthStore";
import { useThemeStore } from "@/stores/ThemeStore";

const app = createApp(App);

app.use(createPinia());

const useAuth = useAuthStore();
const themeStore = useThemeStore();

const themeMediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
themeMediaQuery.addEventListener("change", () => {
  themeStore.syncWithSystemTheme();
});

router.beforeEach((to) => {
  if (to.meta.requireAuth && !useAuth.isAuthenticated) {
    router.push({ name: "home" });
    return false;
  }
  return true;
});
app.use(router);

app.mount("#app");
