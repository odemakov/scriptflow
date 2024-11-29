<script setup lang="ts">
import "@xterm/xterm/css/xterm.css";
import { watch, ref, onMounted, onUnmounted, computed } from 'vue';
import { useRoute } from 'vue-router'
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';

import { useToastStore } from '@/stores/ToastStore';
import { useAuthStore } from '@/stores/AuthStore';
import { useProjectStore } from '@/stores/ProjectStore';

const props = defineProps<{
  run: IRun,
}>()

const auth = useAuthStore()
const route = useRoute()
const useToasts = useToastStore()
const useProjects = useProjectStore()
const terminalRef = ref(null);
const term = new Terminal(CTerminalDefaults)
const projectSlug = Array.isArray(route.params.projectSlug) ? route.params.projectSlug[0] : route.params.projectSlug
const project = computed(() => useProjects.getProject)

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
  const logUrl = `/api/scriptflow/${project.value.id}/run/${props.run.id}/log`;

  // set Autorization header with token
  fetch(logUrl, {
    method: 'GET',
    headers: {
      'Authorization': `${auth.token}`,
    }
  })
    .then(response => response.text())
    .then(data => {
      try {
        const parsed = JSON.parse(data);
        if ('code' in parsed) {
          return new Error(parsed.message);
        } else {
          for (const log of parsed.logs) {
            term.write(log + '\n');
          }
        }
      } catch (error: unknown) {
        useToasts.addToast(
          (error as Error).message,
          'error',
        )
      }
    })
}

onMounted(() => {
  useProjects.fetchProject(projectSlug)

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
