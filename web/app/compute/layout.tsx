import "./compute.css";
import type { ReactNode } from "react";
import { ComputeShell } from "./components/ComputeShell";

export default function ComputeLayout({ children }: { children: ReactNode }) {
  return <ComputeShell>{children}</ComputeShell>;
}
