"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { apiOrigin, authHeaders } from "@/lib/api";

type Row = {
  id: number;
  title: string;
  body: string;
  channel: string;
};

type Props =
  | { variant: "public"; channel: "public" | "job_admin" | "customer_admin" }
  | { variant: "customer_feed" };

/** 各サイトトップ付近にお知らせを並べる（顧客管理ログイン後は JWT 付き feed）。 */
export function AnnouncementBanner(props: Props) {
  const [token, setToken] = useState<string | null>(() =>
    typeof window === "undefined" ? null : localStorage.getItem("cust_token")
  );
  useEffect(() => {
    setToken(localStorage.getItem("cust_token"));
  }, []);

  const isCustomerFeed = props.variant === "customer_feed";
  const { data: items = [] } = useQuery<Row[]>({
    queryKey: isCustomerFeed ? ["ann-feed", token] : ["ann-pub", props.channel],
    queryFn: async () => {
      if (isCustomerFeed) {
        const t = localStorage.getItem("cust_token");
        if (!t) return [];
        const res = await fetch(`${apiOrigin}/admin/customers/announcements/feed`, { headers: authHeaders(t) });
        if (!res.ok) return [];
        const data = await res.json();
        return (data.announcements || []) as Row[];
      }
      const u = new URL(`${apiOrigin}/public/announcements`);
      u.searchParams.set("channel", props.channel);
      const res = await fetch(u.toString());
      if (!res.ok) return [];
      const data = await res.json();
      return (data.announcements || []) as Row[];
    },
    enabled: isCustomerFeed ? !!token : true,
    staleTime: 60_000,
  });

  if (!items.length) return null;

  return (
    <div style={{ marginBottom: "0.75rem" }}>
      {items.map((a) => (
        <div
          key={a.id}
          className="card"
          style={{
            marginBottom: "0.5rem",
            borderLeft: "4px solid #2563eb",
            padding: "0.65rem 1rem",
            background: "#eff6ff",
          }}
        >
          <strong style={{ display: "block", marginBottom: "0.25rem" }}>{a.title}</strong>
          <div style={{ fontSize: "0.9rem", whiteSpace: "pre-wrap", color: "#334155" }}>{a.body}</div>
        </div>
      ))}
    </div>
  );
}
