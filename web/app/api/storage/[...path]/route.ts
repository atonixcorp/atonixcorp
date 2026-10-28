import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ path: string[] }> };

async function proxy(request: Request, context: RouteContext): Promise<NextResponse> {
  const apiURL = process.env.API_URL;
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" }, { status: 401 });
  }
  if (request.method !== "GET" && request.method !== "HEAD" && !isSameOriginRequest(request)) {
    return NextResponse.json(
      { error: "Cross-origin request rejected.", code: "CSRF_VALIDATION_FAILED" },
      { status: 403 },
    );
  }
  if (!apiURL) {
    return NextResponse.json({ error: "Storage API is unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
  const { path } = await context.params;
  const isTenantRoute = path[0] === "tenants" &&
    (path.length === 1 || (path.length === 3 && path[2] === "projects"));
  const isStorageRoute =
    (path[0] === "volume-types" || path[0] === "overview") && path.length === 1 ||
    (path[0] === "volumes" || path[0] === "operations") && path.length <= 2;
  if (!path.length || path.some((part) => !/^[a-zA-Z0-9._-]+$/.test(part)) ||
      (!isTenantRoute && !isStorageRoute)) {
    return NextResponse.json({ error: "Invalid storage API path." }, { status: 404 });
  }

  const incomingURL = new URL(request.url);
  const headers = new Headers();
  headers.set("Authorization", "Bearer " + token);
  for (const name of ["content-type", "idempotency-key", "x-request-id"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  try {
    const apiPath = path[0] === "tenants" ? path : ["storage", ...path];
    const response = await fetch(
      `${apiURL}/api/v1/${apiPath.map(encodeURIComponent).join("/")}${incomingURL.search}`,
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
    return NextResponse.json({ error: "Storage API is temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
export const DELETE = proxy;
