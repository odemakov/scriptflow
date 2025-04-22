import { defineStore } from "pinia";
import { ref, watch } from "vue";

export const useThemeStore = defineStore("theme", () => {
  const theme = ref("light");

  // Function to sync with system preference
  function syncWithSystemTheme() {
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    theme.value = prefersDark ? "dark" : "light";
  }

  // Apply theme when it changes
  watch(theme, (newTheme) => {
    document.documentElement.setAttribute("data-theme", newTheme);
  });

  // Initial sync with system theme
  syncWithSystemTheme();

  return { theme, syncWithSystemTheme };
});
