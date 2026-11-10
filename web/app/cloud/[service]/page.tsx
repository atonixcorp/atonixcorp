import { notFound } from "next/navigation";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import { CloudServiceDirectory, CloudServiceWorkspace } from "@/app/shared/ui/CloudServiceWorkspace";
import { cloudServices } from "@/app/shared/ui/cloud-navigation";

export default async function CloudServicePage({ params, searchParams }: {
  params: Promise<{ service: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { service: id } = await params;
  const query = await searchParams;
  for (const key of ["tenant_id", "project_id"]) {
    const value = query[key];
    if (value !== undefined && (typeof value !== "string" || !/^[1-9]\d*$/.test(value) || !Number.isSafeInteger(Number(value)))) notFound();
  }
  if (query.project_id && !query.tenant_id) notFound();
  const scope = { tenantID: Number(query.tenant_id ?? 0), projectID: Number(query.project_id ?? 0) };
  if (id === "services") return <CloudShell><CloudServiceDirectory scope={scope} /></CloudShell>;
  const service = cloudServices.find((item) => item.id === id && item.href === `/cloud/${id}`);
  if (!service) notFound();
  return <CloudShell><CloudServiceWorkspace service={service} scope={scope} /></CloudShell>;
}
