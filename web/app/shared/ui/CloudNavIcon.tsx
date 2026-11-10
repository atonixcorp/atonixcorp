import type { ReactNode } from "react";
import type { CloudNavigationIcon } from "./cloud-navigation";

const icons: Record<CloudNavigationIcon, ReactNode> = {
  overview: <><rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" /><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" /></>,
  compute: <><rect x="5" y="5" width="14" height="14" rx="2" /><path d="M9 1v4m6-4v4M9 19v4m6-4v4M1 9h4m-4 6h4m14-6h4m-4 6h4" /><rect x="9" y="9" width="6" height="6" rx="1" /></>,
  containers: <><path d="m12 3 9 5v8l-9 5-9-5V8zM3 8l9 5 9-5M12 13v8M7.5 5.5l9 5" /></>,
  storage: <><ellipse cx="12" cy="5" rx="8" ry="3" /><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3" /></>,
  networking: <><rect x="8" y="2" width="8" height="6" rx="1" /><rect x="2" y="16" width="7" height="6" rx="1" /><rect x="15" y="16" width="7" height="6" rx="1" /><path d="M12 8v4M5.5 16v-4h13v4" /></>,
  observability: <><path d="M3 3v18h18M6 14l4-5 4 7 6-10" /><circle cx="20" cy="6" r="1" /></>,
  security: <><path d="m12 2 8 3v6c0 5-4 9-8 11-4-2-8-6-8-11V5zM8 12l3 3 5-6" /></>,
  databases: <><rect x="3" y="3" width="18" height="7" rx="2" /><rect x="3" y="14" width="18" height="7" rx="2" /><path d="M7 6.5h.01M7 17.5h.01M12 6.5h5M12 17.5h5" /></>,
  account: <><rect x="3" y="5" width="18" height="15" rx="2" /><path d="M3 10h18M7 15h4M16 15h1" /></>,
};

export function CloudNavIcon({ name }: { name: CloudNavigationIcon }) {
  return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">{icons[name]}</svg>;
}
