export const apiOrigin = process.env.NEXT_PUBLIC_API_ORIGIN || "http://localhost:8080";

export function authHeaders(token: string | null): HeadersInit {
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}
