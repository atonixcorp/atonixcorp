"use client";

import type { ReactNode } from "react";
import { CloudShell } from "@/app/shared/ui/CloudShell";

export function ComputeShell({ children }: { children: ReactNode }) {
  return (
    <CloudShell>
      <main className="compute-console">
          <section className="compute-main">{children}</section>
      </main>
    </CloudShell>
  );
}
