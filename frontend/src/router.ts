import config from "@/config";
import HomeView from "@/views/HomeView.vue";
import NodeView from "@/views/NodeView.vue";
import NotFoundView from "@/views/NotFoundView.vue";
import ProjectView from "@/views/ProjectView.vue";
import RunView from "@/views/RunView.vue";
import TaskLogView from "@/views/TaskLogView.vue";
import TaskView from "@/views/TaskView.vue";
import { createRouter, createWebHashHistory } from "vue-router";

// Get the base URL from the current location

const router = createRouter({
  history: createWebHashHistory(config.baseUrl),
  routes: [
    {
      path: "/",
      children: [
        {
          path: "",
          name: "home",
          component: HomeView,
          meta: {
            title: "Home",
            requireAuth: false,
          },
        },
        {
          path: "project",
          children: [
            {
              path: ":projectId",
              name: "project",
              component: ProjectView,
              meta: {
                title: "Project",
                requireAuth: true,
              },
            },
            {
              name: "project-task",
              path: ":projectId/task/:taskId/history",
              component: TaskView,
              meta: {
                title: "Task",
                requireAuth: true,
              },
            },
            {
              name: "project-task-log",
              path: ":projectId/task/:taskId/log",
              component: TaskLogView,
              meta: {
                title: "Task",
                requireAuth: true,
              },
            },
            {
              path: ":projectId/task/:taskId/:id",
              name: "project-task-run",
              component: RunView,
              meta: {
                title: "Run",
                requireAuth: true,
              },
            },
          ],
        },
        {
          path: "node",
          children: [
            {
              path: ":nodeId",
              name: "node",
              component: NodeView,
              meta: {
                title: "Node",
                requireAuth: true,
              },
            },
            {
              name: "node-task",
              path: ":nodeId/task/:taskId/history",
              component: TaskView,
              meta: {
                title: "Task",
                requireAuth: true,
              },
            },
            {
              name: "node-task-log",
              path: ":nodeId/task/:taskId/log",
              component: TaskLogView,
              meta: {
                title: "Task",
                requireAuth: true,
              },
            },
            {
              path: ":nodeId/task/:taskId/:id",
              name: "node-task-run",
              component: RunView,
              meta: {
                title: "Run",
                requireAuth: true,
              },
            },
          ],
        },
        {
          path: "task",
          children: [
            {
              name: "task",
              path: ":taskId/history",
              component: TaskView,
              meta: {
                title: "Task",
                requireAuth: true,
              },
            },
            {
              name: "task-log",
              path: ":taskId/log",
              component: TaskLogView,
              meta: {
                title: "Task",
                requireAuth: true,
              },
            },
            {
              path: ":taskId/:id",
              name: "task-run",
              component: RunView,
              meta: {
                title: "Run",
                requireAuth: true,
              },
            },
          ],
        },
      ],
    },
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: NotFoundView,
      meta: {
        title: "Page not found",
        requireAuth: false,
      },
    },
  ],
});

export default router;
