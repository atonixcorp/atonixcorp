import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ path: string[] }> };

function isAllowedPath(path: string[], method: string): boolean {
  if (path.length === 1 && path[0] === "capabilities") {
    return method === "GET";
  }
  if (
    path.length === 5 &&
    path[0] === "tenants" &&
    /^\d+$/.test(path[1]) &&
    path[2] === "projects" &&
    /^\d+$/.test(path[3]) &&
    path[4] === "networks"
  ) {
    return method === "GET" || method === "POST";
  }
  return (
    path.length === 7 &&
    path[0] === "tenants" &&
    /^\d+$/.test(path[1]) &&
    path[2] === "projects" &&
    /^\d+$/.test(path[3]) &&
    path[4] === "networks" &&
    /^[a-f0-9-]{36}$/i.test(path[5]) &&
    path[6] === "subnets" &&
    method === "POST"
  );
}

async function proxy(request: Request, context: RouteContext): Promise<NextResponse> {
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json(
      { error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" },
      { status: 401 },
    );
  }
  if (request.method !== "GET" && request.method !== "HEAD" && !isSameOriginRequest(request)) {
    return NextResponse.json(
      { error: "Cross-origin request rejected.", code: "CSRF_VALIDATION_FAILED" },
      { status: 403 },
    );
  }
  const apiURL = process.env.NETWORK_API_URL;
  if (!apiURL) {
    return NextResponse.json(
      { error: "Networking service is unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503 },
    );
  }

  const { path } = await context.params;
  if (
    !isAllowedPath(path, request.method) ||
    path.some((part) => !/^[a-zA-Z0-9._-]+$/.test(part))
  ) {
    return NextResponse.json(
      { error: "Invalid networking API path.", code: "INVALID_PATH" },
      { status: 404 },
    );
  }

  const incomingURL = new URL(request.url);
  const headers = new Headers({ Authorization: `Bearer ${token}` });
  for (const name of ["content-type", "idempotency-key", "x-request-id"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  try {
    const response = await fetch(
      `${apiURL}/api/v1/networking/${path.map(encodeURIComponent).join("/")}${incomingURL.search}`,
      {
        method: request.method,
        headers,
        ...(request.method === "GET" || request.method === "HEAD"
          ? {}
          : { body: await request.arrayBuffer() }),
        cache: "no-store",
        signal: AbortSignal.timeout(60000),
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
    return NextResponse.json(
      { error: "Networking service is temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503 },
    );
  }
}

export const GET = proxy;
export const POST = proxy;
