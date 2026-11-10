"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { getProjects, getTenants, type ComputeTenant } from "@/app/compute/lib/api";
import { resolveScopeID, scopedCloudHref } from "../lib/cloud-scope";
import { useCloudResource, type CloudResource } from "./useCloudResource";

export function useCloudScope(revision: number, availableWorkspaces?: ComputeTenant[]) {
  const query = useSearchParams();
  const [workspace, setWorkspace] = useState<number | null>(null);
  const [project, setProject] = useState<number | null>(null);
  const loadedTenants = useCloudResource(getTenants, availableWorkspaces ? 0 : 1, 1, revision);
  const tenants: CloudResource<ComputeTenant[]> = availableWorkspaces
    ? { state: "ready", data: availableWorkspaces } : loadedTenants;
  let tenantID = 0;
  let error = "";
  if (tenants.state === "ready") {
    try {
      tenantID = workspace === null
        ? resolveScopeID(tenants.data, "tenant_id", query)
        : tenants.data.find((item) => item.id === workspace)?.id ?? 0;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "Workspace scope could not be resolved.";
    }
  }
  const projects = useCloudResource(getProjects, tenantID, tenantID, revision);
  let projectID = 0;
  if (projects.state === "ready") {
    try {
      projectID = project === null
        ? resolveScopeID(projects.data, "project_id", query, tenantID)
        : projects.data.find((item) => item.id === project)?.id ?? 0;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "Project scope could not be resolved.";
    }
  }
  const updateLocation = (tenant: number, selectedProject: number) => {
    const url = scopedCloudHref(`${window.location.pathname}${window.location.search}${window.location.hash}`, tenant, selectedProject);
    window.history.replaceState(null, "", url);
  };
  return {
    tenants, projects, tenantID, projectID, error,
    selectWorkspace: (id: number) => { setWorkspace(id); setProject(null); updateLocation(id, 0); },
    selectProject: (id: number) => { setProject(id); updateLocation(tenantID, id); },
    href: (path: string) => scopedCloudHref(path, tenantID, projectID),
  };
}
