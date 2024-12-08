import * as Types from "./src/types";

declare global {
  type Toast = Types.Toast;
  type ITask = Types.ITask;
  type IRun = Types.IRun;
  type INode = Types.INode;
  type IProject = Types.IProject;
  type IChannel = Types.IChannel;
  type ISubscription = Types.ISubscription;
  type INotification = Types.INotification;

  // Declare constants in a global namespace
  const CRunStatus: typeof Types.CRunStatus;
  const CNodeStatus: typeof Types.CNodeStatus;
  const CCollectionName: typeof Types.CCollectionName;
  const CTerminalDefaults: typeof Types.CTerminalDefaults;
}

export {};
