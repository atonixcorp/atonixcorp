import type { ComputeCatalog, ComputeSSHKey, DeploymentDraft } from "./types";

export const draftStorageKey = "atonixcorp-compute-deployment-draft";

export function restoreDraft(): DeploymentDraft | null {
  try {
    const saved = window.localStorage.getItem(draftStorageKey);
    if (!saved) return null;
    const parsed: unknown = JSON.parse(saved);
    if (!parsed || typeof parsed !== "object" || !("name" in parsed)) return null;
    return parsed as DeploymentDraft;
  } catch {
    return null;
  }
}

export function saveDraft(draft: DeploymentDraft): boolean {
  try {
    window.localStorage.setItem(draftStorageKey, JSON.stringify(draft));
    return true;
  } catch {
    return false;
  }
}

export function validateDraft(
  draft: DeploymentDraft,
  catalog: ComputeCatalog,
  sshKeys: ComputeSSHKey[],
  requireSSHKey = true,
): { section: string; message: string } | null {
  if (!catalog.compute_types.some((item) => item.id === draft.computeType)) {
    return { section: "compute-type", message: "Choose an available compute type." };
  }
  if (!catalog.regions.some((item) => item.id === draft.region && item.available)) {
    return { section: "region", message: "Choose an available region." };
  }
  if (!catalog.images.some((item) => item.id === draft.image && item.available)) {
    return { section: "image", message: "Choose an available operating system image." };
  }
  if (!catalog.plans.some((item) =>
    item.id === draft.plan &&
    item.region === draft.region &&
    item.compute_type === draft.computeType &&
    item.available)) {
    return { section: "plan", message: "Choose a plan available for this region and compute type." };
  }
  if (draft.network && !catalog.networks.some((item) =>
    item.id === draft.network && item.region === draft.region && item.available)) {
    return { section: "network", message: "Choose a network available in the selected region." };
  }
  if (requireSSHKey && (draft.sshKeyIds.length === 0 || draft.sshKeyIds.some((id) => !sshKeys.some((key) => key.id === id)))) {
    return { section: "security", message: "Select at least one SSH key that belongs to your account." };
  }
  if (!/^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$/.test(draft.name)) {
    return { section: "identity", message: "Use 1–63 letters, numbers, dots, hyphens, or underscores for the instance name." };
  }
  const hostname = draft.hostname || draft.name.replaceAll("_", "-");
  if (!/^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/.test(hostname)) {
    return { section: "identity", message: "Enter a valid DNS hostname." };
  }
  if (!draft.label.trim() || draft.label.length > 128) {
    return { section: "identity", message: "Enter a label between 1 and 128 characters." };
  }
  if (!Number.isInteger(draft.quantity) || draft.quantity < 1 || draft.quantity > 50) {
    return { section: "identity", message: "Choose a quantity between 1 and 50." };
  }
  if (draft.quantity > 1 && (draft.name.length > 60 || hostname.length > 60)) {
    return { section: "identity", message: "Shorten the name so sequential suffixes fit." };
  }
  if (draft.tags.length > 20 || draft.tags.some((tag) => !/^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,62}$/.test(tag))) {
    return { section: "identity", message: "Tags must be 1–63 valid characters, with at most 20 tags." };
  }
  if (draft.userData.length > 65536) {
    return { section: "initialization", message: "User data must be no larger than 64 KiB." };
  }
  if (draft.volumes.length > 16 || draft.volumes.some((volume) =>
    !volume.name.trim() || volume.size_gb < 1 || volume.size_gb > 10000)) {
    return { section: "storage", message: "Each volume needs a name and a size from 1 to 10000 GB." };
  }
  return null;
}
