<script setup lang="ts">
import AnsiToHtml from "ansi-to-html";
import {
  ref,
  computed,
  watch,
  onMounted,
  onBeforeUnmount,
  nextTick,
} from "vue";
import { useToastStore } from "@/stores/ToastStore";
import { useAuthStore } from "@/stores/AuthStore";
import config from "@/config";

type LogLevel = "ERROR" | "WARN" | "INFO" | "DEBUG" | "NONE";
type LogStream = "stdout" | "stderr";

interface LogLine {
  id: number;
  ts: string;
  stream: LogStream;
  content: string;
  level: LogLevel;
  html: string;
  isRunSep: boolean;
}

const props = defineProps<{
  mode: "static" | "live";
  runId?: string;
  task?: ITask;
  logHeight?: string;
}>();

const auth = useAuthStore();
const useToasts = useToastStore();
// Catppuccin Frappé palette for ANSI 16 colors
const CF = {
  bg: "#303446",
  fg: "#C6D0F5",
  red: "#E78284",
  green: "#A6D189",
  yellow: "#E5C890",
  blue: "#8CAAEE",
  magenta: "#F4B8E4",
  cyan: "#81C8BE",
  white: "#B5BFE2",
  brightBlack: "#626880",
  brightWhite: "#A5ADCE",
};

const ansi = new AnsiToHtml({
  fg: CF.fg,
  bg: CF.bg,
  escapeXML: true,
  colors: [
    "#51576D", CF.red, CF.green, CF.yellow, CF.blue, CF.magenta, CF.cyan, CF.white,
    CF.brightBlack, CF.red, CF.green, CF.yellow, CF.blue, CF.magenta, CF.cyan, CF.brightWhite,
  ],
});

const lines = ref<LogLine[]>([]);
const activeFilters = ref<Set<LogLevel>>(new Set(["ERROR", "WARN", "INFO", "DEBUG", "NONE"]));
const activeStreams = ref<Set<LogStream>>(new Set(["stdout", "stderr"]));
const searchQuery = ref("");
const tailMode = ref(true);
const logContainerRef = ref<HTMLElement | null>(null);

// live mode pagination state
const liveLineOffset = ref(0);
const hasMore = ref(false);
const loadingMore = ref(false);

const PAGE_SIZE = 100;

// New format: [RFC3339] [stdout|stderr] content
const newFmtRe = /^\[([^\]]+)\] \[(stdout|stderr)\] (.*)$/;
// Legacy format: [RFC3339] content
const legacyFmtRe = /^\[([^\]]+)\] (.*)$/;
// Run separator: [RFC3339] [scriptflow] run <id>
const runSepRe = /^\[([^\]]+)\] \[scriptflow\] run (\S+)$/;
const levelRe = /\b(ERROR|CRITICAL|WARNING|WARN|INFO|DEBUG)\b/i;
let lineSeq = 0;

function detectLevel(content: string): LogLevel {
  const m = content.match(levelRe);
  if (!m) return "NONE";
  const v = m[1].toUpperCase();
  if (v === "CRITICAL") return "ERROR";
  if (v === "WARNING") return "WARN";
  return v as LogLevel;
}

function parseLine(raw: string): LogLine {
  const sep = raw.match(runSepRe);
  if (sep) {
    return {
      id: lineSeq++,
      ts: sep[1],
      stream: "stdout",
      content: raw,
      level: "NONE",
      html: ansi.toHtml(`[scriptflow] run ${sep[2]}`),
      isRunSep: true,
    };
  }
  const m1 = raw.match(newFmtRe);
  if (m1) {
    const content = m1[3];
    return {
      id: lineSeq++,
      ts: m1[1],
      stream: m1[2] as LogStream,
      content,
      level: detectLevel(content),
      html: ansi.toHtml(content),
      isRunSep: false,
    };
  }
  const m2 = raw.match(legacyFmtRe);
  if (m2) {
    const content = m2[2];
    return {
      id: lineSeq++,
      ts: m2[1],
      stream: "stdout",
      content,
      level: detectLevel(content),
      html: ansi.toHtml(content),
      isRunSep: false,
    };
  }
  return {
    id: lineSeq++,
    ts: "",
    stream: "stdout",
    content: raw,
    level: detectLevel(raw),
    html: ansi.toHtml(raw),
    isRunSep: false,
  };
}

function pushLine(raw: string) {
  lines.value.push(parseLine(raw));
}

function prependLines(raws: string[]) {
  const parsed = raws.map(parseLine);
  lines.value.unshift(...parsed);
}

function toggleFilter(lvl: LogLevel) {
  const f = new Set(activeFilters.value);
  if (f.has(lvl)) f.delete(lvl); else f.add(lvl);
  activeFilters.value = f;
}

function toggleStream(s: LogStream) {
  const f = new Set(activeStreams.value);
  if (f.has(s)) f.delete(s); else f.add(s);
  activeStreams.value = f;
}

const visibleLines = computed(() => {
  const q = searchQuery.value.toLowerCase();
  return lines.value.filter((line) => {
    if (line.isRunSep) return true;
    if (!activeStreams.value.has(line.stream)) return false;
    if (!activeFilters.value.has(line.level)) return false;
    if (q && !line.content.toLowerCase().includes(q)) return false;
    return true;
  });
});

function highlightSearch(html: string): string {
  if (!searchQuery.value) return html;
  const escaped = searchQuery.value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const re = new RegExp(`(${escaped})`, "gi");
  // Only replace in text nodes — skip content inside < ... >
  return html.replace(/(<[^>]*>)|([^<]+)/g, (_, tag, text) => {
    if (tag) return tag;
    return text.replace(re, `<mark style="background:${CF.yellow};color:${CF.bg}">$1</mark>`);
  });
}

const LEVEL_COLORS: Record<string, string> = {
  ERROR: CF.red,
  WARN: CF.yellow,
  INFO: CF.fg,
  DEBUG: CF.brightBlack,
  NONE: CF.fg,
};

function levelBtnStyle(lvl: LogLevel): string {
  if (!activeFilters.value.has(lvl)) return "";
  return `background: ${LEVEL_COLORS[lvl]}; color: ${CF.bg}; border-color: ${LEVEL_COLORS[lvl]};`;
}

function lineStyle(line: LogLine): string {
  if (line.isRunSep) {
    return `color: ${CF.bg}; background: ${CF.fg}; display: block; width: 100%; padding: 1px 8px; font-weight: 600;`;
  }
  const color = LEVEL_COLORS[line.level] ?? CF.fg;
  const border = line.stream === "stderr" ? `border-left: 2px solid ${CF.yellow}; padding-left: 4px;` : "";
  return `color: ${color}; ${border}`;
}

async function scrollToBottom() {
  await nextTick();
  if (logContainerRef.value) {
    logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight;
  }
}

watch(visibleLines, () => {
  if (tailMode.value) scrollToBottom();
});

// Static mode: fetch logs via API
async function fetchLogs() {
  if (!props.runId) return;
  const currentRunId = props.runId;
  lines.value = [];
  try {
    const res = await fetch(`${config.baseUrl}api/scriptflow/run/${currentRunId}/log`, {
      headers: { Authorization: auth.token },
    });
    if (currentRunId !== props.runId) return;
    const data = await res.json();
    if (data.status) throw new Error(data.message);
    for (const raw of data.logs) {
      pushLine(raw);
    }
  } catch (e: unknown) {
    if (currentRunId !== props.runId) return;
    useToasts.addToast((e as Error).message, "error");
  }
}

// Live mode: WebSocket streaming
let ws: WebSocket | null = null;
let wsBuffer = "";

function connectWs() {
  if (!props.task?.id) return;
  wsBuffer = "";
  lines.value = [];
  liveLineOffset.value = PAGE_SIZE;
  hasMore.value = true;

  const token = localStorage.getItem("pocketbase_auth");
  if (!token) {
    useToasts.addToast("No authentication token found.", "error");
    return;
  }

  const url = `${config.baseUrl}api/scriptflow/task/${props.task.id}/log-ws`;
  const localWs = new WebSocket(url);
  ws = localWs;

  localWs.onopen = () => localWs.send(token);

  let fillScheduled = false;
  let burstSettled = false;
  let burstParts = 0;
  let burstEmpty = 0;
  localWs.onmessage = (event: MessageEvent) => {
    if (ws !== localWs) return; // stale connection, discard
    wsBuffer += event.data;
    const parts = wsBuffer.split("\n");
    wsBuffer = parts.pop() ?? "";
    const before = lines.value.length;
    for (const part of parts) {
      if (!burstSettled) {
        burstParts++;
        if (!part) burstEmpty++;
      }
      pushLine(part);
    }
    if (burstSettled) {
      liveLineOffset.value += lines.value.length - before;
    } else if (!fillScheduled) {
      fillScheduled = true;
      setTimeout(() => {
        if (ws !== localWs) return;
        fillScheduled = false;
        burstSettled = true;
        // Sync offset to actual viewer line count. liveLineOffset was initialized to PAGE_SIZE
        // assuming WS delivers exactly PAGE_SIZE lines, but live lines that arrived during the
        // burst window are also in the viewer and must be accounted for to avoid overlapping
        // them with the first scroll page.
        liveLineOffset.value = lines.value.length;
        console.debug(`[LogViewer] WS burst settled: parts=${burstParts} empty=${burstEmpty} displayed=${lines.value.length}`);
        fillIfNeeded();
      }, 300);
    }
  };

  localWs.onerror = () => {
    if (ws === localWs) useToasts.addToast("WebSocket connection failed.", "error");
  };
}

function disconnectWs() {
  if (ws) {
    ws.close();
    ws = null;
  }
}

async function loadMoreLines() {
  if (!props.task || loadingMore.value || !hasMore.value) return;
  loadingMore.value = true;

  const container = logContainerRef.value;
  const prevScrollHeight = container?.scrollHeight ?? 0;

  try {
    const res = await fetch(
      `${config.baseUrl}api/scriptflow/task/${props.task.id}/log?offset=${liveLineOffset.value}&limit=${PAGE_SIZE}`,
      { headers: { Authorization: auth.token } },
    );
    const data = await res.json();
    if (data.lines?.length) {
      prependLines(data.lines);
      liveLineOffset.value += data.lines.length;
      await nextTick();
      if (container) {
        container.scrollTop = container.scrollHeight - prevScrollHeight;
      }
    }
    hasMore.value = data.has_more ?? false;
  } catch (e: unknown) {
    useToasts.addToast((e as Error).message, "error");
  } finally {
    loadingMore.value = false;
  }
}

function onScroll() {
  const container = logContainerRef.value;
  if (!container) return;

  const atBottom = container.scrollTop + container.clientHeight >= container.scrollHeight - 10;
  tailMode.value = atBottom;

  if (props.mode === "live" && container.scrollTop === 0 && !loadingMore.value && hasMore.value) {
    loadMoreLines();
  }
}

// Load more until container is scrollable or no more lines remain.
// Needed when filters hide most lines leaving content shorter than container.
async function fillIfNeeded() {
  if (props.mode !== "live") return;
  const container = logContainerRef.value;
  if (!container) return;
  await nextTick();
  while (hasMore.value && !loadingMore.value && container.scrollHeight <= container.clientHeight) {
    await loadMoreLines();
    await nextTick();
  }
}

// Watchers for prop changes
watch(() => props.runId, fetchLogs);
watch([activeFilters, activeStreams, searchQuery], fillIfNeeded);
watch(
  () => props.task?.id,
  () => {
    disconnectWs();
    connectWs();
  },
);

onMounted(() => {
  if (props.mode === "static") fetchLogs();
  else connectWs();
});

onBeforeUnmount(() => disconnectWs());

function toggleTail() {
  tailMode.value = !tailMode.value;
  if (tailMode.value) scrollToBottom();
}

const levels: LogLevel[] = ["ERROR", "WARN", "INFO", "DEBUG", "NONE"];
</script>

<template>
  <div class="w-full flex flex-col bg-base-300 rounded-lg overflow-hidden">
    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-2 px-3 py-2 bg-base-200 border-b border-base-300 shrink-0">
      <!-- Level filters -->
      <div class="join">
        <button
          v-for="lvl in levels"
          :key="lvl"
          class="join-item btn btn-xs"
          :class="activeFilters.has(lvl) ? '' : 'btn-ghost'"
          :style="levelBtnStyle(lvl)"
          @click="toggleFilter(lvl)"
        >{{ lvl }}</button>
      </div>

      <!-- Stream filters -->
      <div class="join">
        <button
          class="join-item btn btn-xs"
          :class="activeStreams.has('stdout') ? 'btn-primary' : 'btn-ghost'"
          @click="toggleStream('stdout')"
        >STDOUT</button>
        <button
          class="join-item btn btn-xs"
          :class="activeStreams.has('stderr') ? 'btn-warning' : 'btn-ghost'"
          @click="toggleStream('stderr')"
        >STDERR</button>
      </div>

      <!-- Search -->
      <input
        v-model="searchQuery"
        class="input input-xs w-40"
        placeholder="Search..."
      />

      <!-- Tail toggle -->
      <button
        class="btn btn-xs ml-auto"
        :class="tailMode ? 'btn-success' : 'btn-ghost'"
        @click="toggleTail"
        title="Toggle auto-scroll"
      >Tail</button>

      <!-- Line count -->
      <span class="text-xs opacity-50">{{ visibleLines.length }} rows</span>
    </div>

    <!-- Log output -->
    <div
      ref="logContainerRef"
      class="overflow-y-auto p-1 font-mono text-sm"
      :style="`height: ${props.logHeight ?? 'calc(100vh - 20rem)'}; background: ${CF.bg}; color: ${CF.fg}; scrollbar-width: thin; scrollbar-color: ${CF.brightBlack} ${CF.bg};`"
      @scroll="onScroll"
    >
      <!-- Load more indicator -->
      <div
        v-if="mode === 'live'"
        class="px-2 py-1 text-xs text-center"
        :style="`color: ${CF.brightBlack}`"
      >
        <span v-if="loadingMore">loading…</span>
        <span v-else-if="hasMore">scroll up for older lines</span>
        <span v-else>beginning of log</span>
      </div>

      <div
        v-for="line in visibleLines"
        :key="line.id"
        class="leading-5 whitespace-pre-wrap break-all px-2 py-0"
        :style="lineStyle(line)"
        v-html="highlightSearch(line.html)"
      />
      <div v-if="visibleLines.length === 0" class="text-sm opacity-40 p-4 font-mono">
        No log lines match the current filter.
      </div>
    </div>
  </div>
</template>
