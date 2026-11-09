"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";

type StoredObject = {
  id: string;
  original_filename: string;
  category: string;
  size_bytes: number;
  created_at: string;
};
type UploadForm = { object_id: string; upload: { url: string; fields: Record<string, string> } };
const categories = ["uploads", "images", "documents", "backups", "logs", "datasets", "models", "users"] as const;
const contentTypes = ["text/plain", "application/pdf", "image/jpeg", "image/png", "image/gif", "image/webp", "application/zip", "application/x-gzip", "application/octet-stream"];
type Category = typeof categories[number];

async function api<T>(path: string, method = "GET", body?: unknown): Promise<T> {
  const response = await fetch(`/api/files/${path}`, {
    method,
    cache: "no-store",
    ...(body === undefined ? {} : { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
  });
  if (!response.ok) {
    const data: { error?: string } = await response.json().catch(() => ({}));
    throw new Error(data.error || `File request failed (${response.status}).`);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export default function ObjectStorage({ tenantID }: { tenantID: number }) {
  const [objects, setObjects] = useState<StoredObject[]>([]);
  const [offset, setOffset] = useState(0);
  const [category, setCategory] = useState<Category>("uploads");
  const [file, setFile] = useState<File | null>(null);
  const [contentType, setContentType] = useState("");
  const [subcategory, setSubcategory] = useState("general");
  const [service, setService] = useState("manual");
  const [project, setProject] = useState("general");
  const [dataset, setDataset] = useState("general");
  const [model, setModel] = useState("general");
  const [version, setVersion] = useState("v1");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const refresh = useCallback(async () => {
    const items = await api<StoredObject[]>(`objects?${new URLSearchParams({ tenant_id: String(tenantID), limit: "50", offset: String(offset) })}`);
    setObjects(items);
  }, [tenantID, offset]);
  useEffect(() => {
    let active = true;
    api<StoredObject[]>(`objects?${new URLSearchParams({ tenant_id: String(tenantID), limit: "50", offset: String(offset) })}`)
      .then((items) => { if (active) setObjects(items); })
      .catch((err: unknown) => { if (active) setError(err instanceof Error ? err.message : "Objects could not be loaded."); });
    return () => { active = false; };
  }, [tenantID, offset]);

  async function upload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!file) return;
    setBusy(true); setError(""); setMessage("");
    try {
      if (file.size < 1 || file.size > 100 * 1024 * 1024) throw new Error("Choose a non-empty file of at most 100 MiB.");
      const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
      const checksum = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
      const signed = await api<UploadForm>("direct-uploads", "POST", {
        tenant_id: tenantID, category, filename: file.name, content_type: contentType, size_bytes: file.size, checksum,
        subcategory, service, project, dataset, model, version,
      });
      const form = new FormData();
      for (const [key, value] of Object.entries(signed.upload.fields)) form.append(key, value);
      form.append("file", file);
      const response = await fetch(signed.upload.url, { method: "POST", body: form, credentials: "omit" });
      if (!response.ok) throw new Error(`Object upload failed (${response.status}).`);
      await api<StoredObject>(`direct-uploads/${encodeURIComponent(signed.object_id)}/complete`, "POST");
      setMessage("File uploaded and verified.");
      if (offset === 0) await refresh(); else setOffset(0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "File upload failed.");
    } finally { setBusy(false); }
  }

  async function download(id: string) {
    setError("");
    try {
      const result = await api<{ url: string }>(`${encodeURIComponent(id)}/download-url`);
      window.location.assign(result.url);
    } catch (err) { setError(err instanceof Error ? err.message : "Download failed."); }
  }

  async function remove(id: string) {
    if (!window.confirm("Delete this object? This cannot be undone.")) return;
    setBusy(true); setError("");
    try {
      await api<void>(encodeURIComponent(id), "DELETE");
      await refresh();
    } catch (err) { setError(err instanceof Error ? err.message : "Deletion failed."); }
    finally { setBusy(false); }
  }

  return <section className="storage-table-panel">
    <div className="storage-panel-heading"><div><p className="storage-eyebrow">PRIVATE OBJECT STORAGE</p><h2>My objects</h2><p>Objects belong to you within the selected organization, not the selected block-storage project.</p></div></div>
    {error && <div className="storage-alert" role="alert">{error}</div>}
    {message && <p role="status">{message}</p>}
    <form onSubmit={upload} className="storage-object-form">
      <label className="storage-field"><span>Category</span><select value={category} onChange={(event) => setCategory(event.target.value as Category)}>{categories.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
      <label className="storage-field"><span>File (maximum 100 MiB)</span><input type="file" required onChange={(event) => { const selected = event.target.files?.[0] ?? null; setFile(selected); setContentType(selected && contentTypes.includes(selected.type) ? selected.type : ""); }} /></label>
      <label className="storage-field"><span>Content type</span><select value={contentType} onChange={(event) => setContentType(event.target.value)} required><option value="">Select content type</option>{contentTypes.map((type) => <option key={type} value={type}>{type}</option>)}</select></label>
      {["images", "documents", "users"].includes(category) && <label className="storage-field"><span>Subcategory</span><input value={subcategory} onChange={(event) => setSubcategory(event.target.value)} required /></label>}
      {["backups", "logs"].includes(category) && <label className="storage-field"><span>Service (platform super admins only)</span><input value={service} onChange={(event) => setService(event.target.value)} required /></label>}
      {["datasets", "models"].includes(category) && <><label className="storage-field"><span>Project label</span><input value={project} onChange={(event) => setProject(event.target.value)} required /></label><label className="storage-field"><span>{category === "models" ? "Model" : "Dataset"}</span><input value={category === "models" ? model : dataset} onChange={(event) => category === "models" ? setModel(event.target.value) : setDataset(event.target.value)} required /></label><label className="storage-field"><span>Version</span><input value={version} onChange={(event) => setVersion(event.target.value)} required /></label></>}
      <button className="storage-primary-button" disabled={busy || !file} type="submit">{busy ? "Working…" : "Upload object"}</button>
    </form>
    <div className="storage-table-scroll"><table className="storage-table"><thead><tr><th>File</th><th>Category</th><th>Size</th><th>Created</th><th>Actions</th></tr></thead><tbody>
      {objects.map((obj) => <tr key={obj.id}><td>{obj.original_filename}</td><td>{obj.category}</td><td>{obj.size_bytes.toLocaleString()} bytes</td><td>{new Date(obj.created_at).toLocaleString()}</td><td><button type="button" className="storage-secondary-button" onClick={() => void download(obj.id)}>Download</button>{" "}<button type="button" className="storage-delete-button" disabled={busy} onClick={() => void remove(obj.id)}>Delete</button></td></tr>)}
    </tbody></table></div>
    {objects.length === 0 && <p>No objects on this page.</p>}
    <div><button type="button" className="storage-secondary-button" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - 50))}>Previous</button>{" "}<button type="button" className="storage-secondary-button" disabled={objects.length < 50} onClick={() => setOffset(offset + 50)}>Next</button></div>
  </section>;
}
