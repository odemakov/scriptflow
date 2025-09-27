<script setup lang="ts">
import "@xterm/xterm/css/xterm.css";
import { watch, ref, onMounted, onBeforeUnmount } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";

import { useToastStore } from "@/stores/ToastStore";
import { useAuthStore } from "@/stores/AuthStore";
import config from "@/config";

const props = defineProps<{
  run: IRun;
}>();

const auth = useAuthStore();
const useToasts = useToastStore();
const terminalRef = ref(null);
const term = new Terminal(CTerminalDefaults);

// Initialize FitAddon
const fitAddon = new FitAddon();
const handleResize = () => fitAddon.fit();

watch(
  () => props.run,
  async () => {
    try {
      await fetchLogs();
    } catch (error: unknown) {
      useToasts.addToast((error as Error).message, "error");
    }
  },
);

// function to retrieve logs from the server
const fetchLogs = async () => {
  const logUrl = `${config.baseUrl}api/scriptflow/run/${props.run.id}/log`;

  try {
    const response = await fetch(logUrl, {
      method: "GET",
      headers: {
        Authorization: `${auth.token}`,
      },
    });

    const data = await response.text();
    const parsed = JSON.parse(data);

    if ("status" in parsed) {
      throw new Error(parsed.message);
    } else {
      for (const log of parsed.logs) {
        term.write(log + "\n");
      }
    }
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
};

onMounted(() => {
  // Open Terminal
  if (terminalRef.value) {
    term.open(terminalRef.value);
    // set some styling
    if (term.element) {
      term.element.style.padding = "0.5rem";
    }
  }
  term.loadAddon(fitAddon);

  // Fit the Terminal to its container
  fitAddon.fit();

  window.addEventListener("resize", handleResize);

  // Handle resizing dynamically
  window.addEventListener("resize", () => {
    fitAddon.fit();
  });
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", handleResize);
  term.dispose();
});
</script>

<template>
  <div class="h-full w-full">
    <div ref="terminalRef" class="m-2"></div>
  </div>
</template>
