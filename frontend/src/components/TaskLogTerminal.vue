<script setup lang="ts">
import "@xterm/xterm/css/xterm.css";
import { watch, ref, onMounted, onUnmounted } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { AttachAddon } from "@xterm/addon-attach";
import config from "@/config";

import { useToastStore } from "@/stores/ToastStore";

const props = defineProps<{
  task: ITask;
}>();

const useToasts = useToastStore();

const terminalRef = ref(null);
const term = new Terminal(CTerminalDefaults);

// Initialize FitAddon
const fitAddon = new FitAddon();
const handleResize = () => fitAddon.fit();

let webSocket: WebSocket;

watch(
  () => props.task,
  async () => {
    try {
      const token = localStorage.getItem("pocketbase_auth");
      if (!token) {
        useToasts.addToast("No authentication token found.", "error");
        return;
      }

      const ws = `${config.baseUrl}api/scriptflow/task/${props.task.id}/log-ws`;
      webSocket = new WebSocket(ws);

      // Send authentication as the first message
      webSocket.onopen = function () {
        webSocket.send(token);
      };

      // Handle WebSocket events
      webSocket.onerror = () => {
        useToasts.addToast("WebSocket connection failed.", "error");
      };

      const attachAddon = new AttachAddon(webSocket, { bidirectional: false });
      term.loadAddon(attachAddon);
    } catch (error) {
      useToasts.addToast((error as any).message, "error");
    }
  },
);

onMounted(() => {
  if (terminalRef.value) {
    // Open Terminal
    term.open(terminalRef.value);
    term.loadAddon(fitAddon);

    // Fit the Terminal to its container
    fitAddon.fit();

    window.addEventListener("resize", handleResize);

    // Handle resizing dynamically
    window.addEventListener("resize", () => {
      fitAddon.fit();
    });
  }
});

onUnmounted(() => {
  window.removeEventListener("resize", handleResize);
  term.dispose();
  // Close the WebSocket connection if it exists
  if (webSocket) {
    webSocket.close();
  }
});
</script>

<template>
  <div class="h-full w-full overflow-auto">
    <div ref="terminalRef" class="inline-block min-w-full"></div>
  </div>
</template>
