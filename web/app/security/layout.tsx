import type { Metadata } from "next";
import type { ReactNode } from "react";

export const metadata: Metadata = {
  title: "Security | AtonixCorp",
  description: "Scoped security policies, network groups, and audit records.",
};

export default function SecurityLayout({ children }: { children: ReactNode }) {
  return children;
}