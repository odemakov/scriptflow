import { createRouter, createWebHistory } from "vue-router";
import HomeView from "@/views/HomeView.vue";
import ProjectView from "@/views/ProjectView.vue";
import TaskView from "@/views/TaskView.vue";
import TaskLogView from "@/views/TaskLogView.vue";
import RunView from "@/views/RunView.vue";
import NotFoundView from "@/views/NotFoundView.vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      name: "home",
      component: HomeView,
      meta: {
        title: "Home",
        requireAuth: false,
      },
    },
    {
      path: "/project/:id",
      name: "project",
      component: ProjectView,
      meta: {
        title: "Project",
        requireAuth: true,
      },
    },
    {
      path: "/task/:id",
      children: [
        {
          name: "task",
          path: "history",
          component: TaskView,
          meta: {
            title: "Task",
            requireAuth: true,
          },
        },
        {
          name: "task-log",
          path: "log",
          component: TaskLogView,
          meta: {
            title: "Task",
            requireAuth: true,
          },
        },
      ],
    },
    {
      path: "/run/:id",
      name: "run",
      component: RunView,
      meta: {
        title: "Run",
        requireAuth: true,
      },
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
