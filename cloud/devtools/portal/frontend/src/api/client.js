const API = (import.meta.env.VITE_ATCLOUD_API || "/api").replace(/\/$/, "");

export async function apiGet(path) {
  const token = sessionStorage.getItem("atcloud.portalToken");
  const res = await fetch(API + path, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });

  if (!res.ok) throw new Error(await res.text());
  if (res.status === 204) return null;
  return res.json();
}
