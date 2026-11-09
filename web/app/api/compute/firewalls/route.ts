import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";
const maxBodyBytes = 16 * 1024;

async function readBoundedBody(request: Request): Promise<ArrayBuffer | null> {
  const declaredLength = request.headers.get("content-length");
  if (declaredLength && /^\d+$/.test(declaredLength) && Number(declaredLength) > maxBodyBytes) {
    return null;
  }
  if (!request.body) return new ArrayBuffer(0);
  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.byteLength;
    if (total > maxBodyBytes) {
      await reader.cancel().catch(() => undefined);
      return null;
    }
    chunks.push(value);
  }
  const body = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    body.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return body.buffer;
}

async function proxy(request: Request): Promise<NextResponse> {
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" }, { status: 401 });
  }
  if (request.method !== "GET" && !isSameOriginRequest(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected.", code: "CSRF_VALIDATION_FAILED" }, { status: 403 });
  }
  const apiURL = process.env.API_URL;
  if (!apiURL) {
    return NextResponse.json({ error: "Security groups are unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
  const incoming = new URL(request.url);
  const tenantID = incoming.searchParams.get("tenant_id") ?? "";
  const projectID = incoming.searchParams.get("project_id") ?? "";
  if (!/^[1-9]\d*$/.test(tenantID) || !/^[1-9]\d*$/.test(projectID)) {
    return NextResponse.json({ error: "A workspace and project are required.", code: "INVALID_SCOPE" }, { status: 400 });
  }
  const headers = new Headers({ Authorization: "Bearer " + token });
  let body: ArrayBuffer | undefined;
  if (request.method === "POST") {
    const bytes = await readBoundedBody(request);
    if (bytes === null) {
      return NextResponse.json({ error: "Request body is too large.", code: "REQUEST_TOO_LARGE" }, { status: 413 });
    }
    body = bytes;
    headers.set("Content-Type", "application/json");
  }
  try {
    const response = await fetch(
      `${apiURL}/api/v1/cloud/security-groups?tenant_id=${encodeURIComponent(tenantID)}&project_id=${encodeURIComponent(projectID)}`,
      {
        method: request.method,
        headers,
        ...(body ? { body } : {}),
        cache: "no-store",
        signal: AbortSignal.timeout(15000),
      },
    );
    return new NextResponse(await response.arrayBuffer(), {
      status: response.status,
      headers: {
        "Content-Type": response.headers.get("content-type") || "application/json",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return NextResponse.json({ error: "Security groups are temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
