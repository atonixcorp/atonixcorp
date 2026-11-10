"use client";

import { useEffect, useState } from "react";

export type CloudResource<T> =
  | { state: "idle"; data?: never; error?: never }
  | { state: "loading"; data?: never; error?: never }
  | { state: "ready"; data: T; error?: never }
  | { state: "error"; data?: never; error: string; status?: number };

export function useCloudResource<T>(
  load: (tenantID: number, projectID: number) => Promise<T>,
  tenantID: number,
  projectID: number,
  revision = 0,
): CloudResource<T> {
  const key = `${tenantID}:${projectID}:${revision}`;
  const [result, setResult] = useState<{ key: string; resource: CloudResource<T> }>();

  useEffect(() => {
    if (!tenantID || !projectID) return;
    let active = true;
    void load(tenantID, projectID).then(
      (data) => {
        if (active) setResult({ key, resource: { state: "ready", data } });
      },
      (error: unknown) => {
        if (active) setResult({
          key,
          resource: {
            state: "error",
            error: error instanceof Error ? error.message : "The resource service could not be reached.",
            status: error instanceof Error && "status" in error && typeof error.status === "number" ? error.status : undefined,
          },
        });
      },
    );
    return () => { active = false; };
  }, [key, load, tenantID, projectID]);

  if (!tenantID || !projectID) return { state: "idle" };
  // Never show another project's data, including the render before effect cleanup.
  return result?.key === key ? result.resource : { state: "loading" };
}
