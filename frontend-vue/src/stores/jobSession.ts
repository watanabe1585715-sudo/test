import { defineStore } from "pinia";

/** 案件管理画面の JWT と顧客 ID（Pinia でアプリ全体から参照しやすくする）。 */
export const useJobSessionStore = defineStore("jobSession", {
  state: () => ({
    token: null as string | null,
    customerId: null as number | null,
  }),
  actions: {
    /** ページ再読込時に localStorage から復元する。 */
    hydrate() {
      const t = localStorage.getItem("job_token");
      const c = localStorage.getItem("job_cid");
      this.token = t;
      this.customerId = c ? Number(c) : null;
    },
    setSession(token: string, customerId: number) {
      this.token = token;
      this.customerId = customerId;
      localStorage.setItem("job_token", token);
      localStorage.setItem("job_cid", String(customerId));
    },
    clear() {
      this.token = null;
      this.customerId = null;
      localStorage.removeItem("job_token");
      localStorage.removeItem("job_cid");
    },
  },
});
