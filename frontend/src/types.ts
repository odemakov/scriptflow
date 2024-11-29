// global const with collection names
export const CCollectionName = {
  tasks: "tasks",
  runs: "runs",
  nodes: "nodes",
  projects: "projects",
} as const;

export const CRunStatus = {
  started: "started",
  completed: "completed",
  interrupted: "interrupted",
  error: "error",
  internal_error: "internal_error",
} as const;

export const CNodeStatus = {
  online: "online",
  offline: "offline",
} as const;

export const CTerminalDefaults = {
  cursorBlink: true,
  // fontFamily: 'monospace',
  // fontSize: 13,
  lineHeight: 1.2,
  tabStopWidth: 4,
  convertEol: true,
  theme: {
    background: "#303446",
    foreground: "#c6d0f5",
    cursor: "#c6d0f5",
  },
} as const;

export type Toast = {
  fired: string;
  message: string;
  type: "success" | "error" | "info" | "warning";
  duration: number;
  timeout?: ReturnType<typeof setTimeout>;
};

export type Breadcrumbs = {
  name: string;
  route: string;
};

export interface IBack {
  to: () => void;
  label: string;
}

export let emptyBack = {
  to: () => {},
  label: "",
} as IBack;

export interface INode {
  id: string;
  collectionName: string;
  host: string;
  user: string;
  name: string;
  created: string;
  updated: string;
}

export interface IProject {
  id: string;
  collectionName: string;
  name: string;
  created: string;
  updated: string;
}

export interface ITask {
  id: string;
  collectionName: string;
  name: string;
  active: boolean;
  command: string;
  project: string;
  node: string;
  expand: {
    project?: IProject;
    node?: INode;
  };
  created: string;
  updated: string;
}

export interface IRun {
  id: string;
  collectionName: string;
  task: string;
  status: string; // started, completed, interrupted, error, internal_error
  host: string;
  command: string;
  connection_error: string;
  exit_code: number;
  expand: {
    task?: ITask;
  };
  created: string;
  updated: string;
}
