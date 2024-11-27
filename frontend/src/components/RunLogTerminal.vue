<script setup lang="ts">
import "@xterm/xterm/css/xterm.css";
import { watch, ref, onMounted, onUnmounted } from 'vue';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';

import { useToastStore } from '@/stores/ToastStore';
import { useAuthStore } from '@/stores/pocketbase'

const props = defineProps<{
  run: IRun,
}>()

const auth = useAuthStore()
const useToasts = useToastStore()
const terminalRef = ref(null);
const term = new Terminal(CTerminalDefaults)

// Initialize FitAddon
const fitAddon = new FitAddon();
const handleResize = () => fitAddon.fit();

watch(() => props.run, async () => {
  try {
    await fetchLogs();
  } catch (error: unknown) {
    useToasts.addToast(
      (error as Error).message,
      'error',
    )
  }
})

// function to retrieve logs from the server
const fetchLogs = async () => {
  const logUrl = `${import.meta.env.VITE_PB_BACKEND_URL}/api/scriptflow/run/${props.run.id}/log`;

  // set Autorization header with token
  fetch(logUrl, {
    method: 'GET',
    headers: {
      'Authorization': `${auth.token}`,
    }
  })
    .then(response => response.text())
    .then(data => {
      const parsed = JSON.parse(data);
      if ('code' in parsed) {
        useToasts.addToast(parsed.message, 'error');
      } else {
        for (const log of parsed.logs) {
          term.write(log + '\n');
        }
      }
    })
}

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

  window.addEventListener('resize', handleResize);

  // Handle resizing dynamically
  window.addEventListener('resize', () => {
    fitAddon.fit();
  });
});

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  term.dispose();
});
</script>

<template>
  <div class="h-full w-full">
    <div ref="terminalRef" class="m-2"></div>
  </div>
</template>
