import config from "@/config";
import HomeView from "@/views/HomeView.vue";
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
          path: "project/:projectId",
          name: "project",
          component: ProjectView,
          meta: {
            title: "Project",
            requireAuth: true,
          },
        },
        {
          name: "task",
          path: "task/:taskId/history",
          component: TaskView,
          meta: {
            title: "Task",
            requireAuth: true,
          },
        },
        {
          name: "task-log",
          path: "task/:taskId/log",
          component: TaskLogView,
          meta: {
            title: "Task",
            requireAuth: true,
          },
        },
        {
          path: "task/:taskId/:id",
          name: "run",
          component: RunView,
          meta: {
            title: "Run",
            requireAuth: true,
          },
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
