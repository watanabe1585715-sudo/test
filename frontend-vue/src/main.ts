import { VueQueryPlugin, QueryClient } from "@tanstack/vue-query";
import { createPinia } from "pinia";
import { createApp } from "vue";
import App from "./App.vue";
import "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, refetchOnWindowFocus: false },
  },
});

const app = createApp(App);
app.use(createPinia());
app.use(VueQueryPlugin, { queryClient });
app.mount("#app");
