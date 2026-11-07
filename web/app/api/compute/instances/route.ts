import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";

async function proxy(request: Request): Promise<NextResponse> {
  const apiURL = process.env.API_URL;
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" }, { status: 401 });
  }
  if (request.method !== "GET" && !isSameOriginRequest(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected.", code: "CSRF_VALIDATION_FAILED" }, { status: 403 });
  }
  if (!apiURL) {
    return NextResponse.json({ error: "Compute API is unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }

  const url = new URL(request.url);
  if (request.method === "GET") {
    const tenantID = url.searchParams.get("tenant_id") ?? "";
    const projectID = url.searchParams.get("project_id") ?? "";
    if (!/^[1-9]\d*$/.test(tenantID) || !/^[1-9]\d*$/.test(projectID)) {
      return NextResponse.json({ error: "A tenant and project are required.", code: "INVALID_SCOPE" }, { status: 400 });
    }
  }
  const headers = new Headers({ Authorization: "Bearer " + token });
  for (const name of ["content-type", "idempotency-key", "x-request-id"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  try {
    const response = await fetch(`${apiURL}/api/v1/cloud/instances${url.search}`, {
      method: request.method,
      headers,
      ...(request.method === "GET" ? {} : { body: await request.arrayBuffer() }),
      cache: "no-store",
      signal: AbortSignal.timeout(60000),
    });
    return new NextResponse(await response.arrayBuffer(), {
      status: response.status,
      headers: {
        "Content-Type": response.headers.get("content-type") || "application/json",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return NextResponse.json({ error: "Compute API is temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
