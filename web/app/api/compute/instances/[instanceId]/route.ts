import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ instanceId: string }> };

async function proxy(request: Request, context: RouteContext): Promise<NextResponse> {
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

  const { instanceId } = await context.params;
  if (!/^[a-f0-9]{32}$/.test(instanceId)) {
    return NextResponse.json({ error: "Invalid instance ID.", code: "INVALID_ID" }, { status: 400 });
  }
  const incomingURL = new URL(request.url);
  const action = incomingURL.searchParams.get("action");
  if (request.method === "POST" && !["start", "stop", "reboot"].includes(action ?? "")) {
    return NextResponse.json({ error: "Invalid instance action.", code: "INVALID_ACTION" }, { status: 400 });
  }

  const path = `/api/v1/cloud/instances/${encodeURIComponent(instanceId)}` +
    (request.method === "POST" ? `/${action}` : "");
  const headers = new Headers({ Authorization: "Bearer " + token });
  for (const name of ["content-type", "idempotency-key", "x-request-id"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  try {
    const response = await fetch(`${apiURL}${path}`, {
      method: request.method,
      headers,
      cache: "no-store",
      signal: AbortSignal.timeout(60000),
    });
    const body = response.status === 204 ? null : await response.arrayBuffer();
    return new NextResponse(body, {
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
export const DELETE = proxy;
