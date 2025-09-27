<script setup lang="ts">
import "@xterm/xterm/css/xterm.css";
import { watch, ref, onMounted, onBeforeUnmount } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { AttachAddon } from "@xterm/addon-attach";
import { SearchAddon } from "@xterm/addon-search";

import config from "@/config";

import { useToastStore } from "@/stores/ToastStore";

const props = defineProps<{
  task: ITask;
}>();

const useToasts = useToastStore();

const terminalRef = ref(null);
const searchInputRef = ref(null);
const dropdownRef = ref(null);
const searchQuery = ref("");

// Configure terminal with buffer management
const terminalConfig = {
  ...CTerminalDefaults,
  scrollback: 10000, // Limit buffer to prevent infinite growth
};
const term = new Terminal(terminalConfig);

// Initialize addons
const fitAddon = new FitAddon();
const searchAddon = new SearchAddon();
const handleResize = () => fitAddon.fit();

// Search functionality
const openSearchMenu = () => {
  // Focus the dropdown button to open it
  if (dropdownRef.value) {
    (dropdownRef.value as HTMLElement).focus();
    // Give dropdown time to open, then focus search
    setTimeout(() => {
      if (searchInputRef.value) {
        (searchInputRef.value as HTMLInputElement).focus();
      }
    }, 150);
  }
};

const performSearch = (direction: "next" | "previous" = "next") => {
  if (!searchQuery.value) return;

  if (direction === "next") {
    searchAddon.findNext(searchQuery.value);
  } else {
    searchAddon.findPrevious(searchQuery.value);
  }
};

const onSearchKeydown = (event: KeyboardEvent) => {
  if (event.key === "Enter") {
    event.preventDefault();
    performSearch(event.shiftKey ? "previous" : "next");
  } else if (event.key === "Escape") {
    searchQuery.value = "";
    // Blur the search input to close dropdown
    if (searchInputRef.value) {
      (searchInputRef.value as HTMLInputElement).blur();
    }
  }
};

const onSearchInput = () => {
  if (searchQuery.value) {
    performSearch("next");
  }
};

// Scroll to bottom functionality
const scrollToBottom = () => {
  term.scrollToBottom();
  closeDropdown();
};

// Global keyboard shortcuts
const onGlobalKeydown = (event: KeyboardEvent) => {
  if (event.key === "/") {
    event.preventDefault();
    event.stopPropagation();
    openSearchMenu();
  }
};

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

      // Clear terminal and reset search when switching tasks
      term.clear();
      searchQuery.value = "";
    } catch (error) {
      useToasts.addToast((error as any).message, "error");
    }
  },
);

const closeDropdown = () => {
  // Close dropdown by removing focus from any focused element within dropdown
  const activeElement = document.activeElement;
  if (activeElement instanceof HTMLElement) {
    activeElement.blur();
  }
};

onMounted(() => {
  if (terminalRef.value) {
    // Open Terminal
    term.open(terminalRef.value);
    term.loadAddon(fitAddon);
    term.loadAddon(searchAddon);

    // Fit the Terminal to its container
    fitAddon.fit();

    // Add event listeners
    window.addEventListener("resize", handleResize);
    document.addEventListener("keydown", onGlobalKeydown);
  }
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", handleResize);
  document.removeEventListener("keydown", onGlobalKeydown);
  term.dispose();
  // Close the WebSocket connection if it exists
  if (webSocket) {
    webSocket.close();
  }
});
</script>

<template>
  <div class="h-full w-full flex flex-col relative">
    <!-- Burger Menu -->
    <div class="absolute top-2 right-2 z-10">
      <div class="dropdown dropdown-end">
        <div
          ref="dropdownRef"
          tabindex="0"
          role="button"
          class="btn btn-sm btn-circle btn-ghost bg-base-200/80 hover:bg-base-300"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M4 12h16M4 18h16"
            ></path>
          </svg>
        </div>
        <ul
          tabindex="0"
          class="dropdown-content menu bg-base-100 rounded-box z-[1] w-80 p-2 shadow-lg border border-base-300"
        >
          <li>
            <div class="flex items-center gap-2 px-4 py-2">
              <svg
                class="w-4 h-4 text-base-content/70"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                ></path>
              </svg>
              <input
                ref="searchInputRef"
                v-model="searchQuery"
                @keydown="onSearchKeydown"
                @input="onSearchInput"
                class="input input-xs flex-1 ml-2"
                placeholder="type here... (/ to focus)"
              />
            </div>
          </li>

          <li>
            <a @click="scrollToBottom" class="flex items-center gap-2">
              <svg
                class="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 14l-7 7m0 0l-7-7m7 7V3"
                ></path>
              </svg>
              Scroll to Bottom
            </a>
          </li>
        </ul>
      </div>
    </div>

    <!-- Terminal Container -->
    <div class="flex-1 overflow-hidden">
      <div ref="terminalRef" class="terminal-container h-full w-full"></div>
    </div>
  </div>
</template>

<style scoped>
.terminal-container {
  /* Custom scrollbar styling for terminal */
}

.terminal-container :deep(.xterm-viewport) {
  /* Style the terminal scrollbar */
  scrollbar-width: thin;
  scrollbar-color: #4a5568 #2d3748;
}

.terminal-container :deep(.xterm-viewport)::-webkit-scrollbar {
  width: 12px;
}

.terminal-container :deep(.xterm-viewport)::-webkit-scrollbar-track {
  background: #2d3748;
}

.terminal-container :deep(.xterm-viewport)::-webkit-scrollbar-thumb {
  background: #4a5568;
  border-radius: 6px;
}

.terminal-container :deep(.xterm-viewport)::-webkit-scrollbar-thumb:hover {
  background: #718096;
}

.terminal-container :deep(.xterm-screen) {
  padding: 0.5rem;
}
</style>
