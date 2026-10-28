import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ path: string[] }> };

function isAllowedPath(path: string[]): boolean {
  if (path.length === 1) {
    return path[0] === "database-engines" || path[0] === "database-plans";
  }
  const scopedDatabaseRoute = path.length >= 5 &&
    path.length <= 6 &&
    path[0] === "tenants" &&
    /^\d+$/.test(path[1]) &&
    path[2] === "projects" &&
    /^\d+$/.test(path[3]) &&
    path[4] === "databases" &&
    (path.length === 5 || /^[a-zA-Z0-9._-]+$/.test(path[5]));
  const scopedOperationRoute = (path.length === 6 || path.length === 7) &&
    path[0] === "tenants" &&
    /^\d+$/.test(path[1]) &&
    path[2] === "projects" &&
    /^\d+$/.test(path[3]) &&
    path[4] === "operations" &&
    /^[a-zA-Z0-9._-]+$/.test(path[5]) &&
    (path.length === 6 || path[6] === "events");
  return scopedDatabaseRoute || scopedOperationRoute;
}

async function proxy(request: Request, context: RouteContext): Promise<NextResponse> {
  const apiURL = process.env.DATABASE_API_URL;
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required." }, { status: 401 });
  }
  if (request.method !== "GET" && request.method !== "HEAD" && !isSameOriginRequest(request)) {
    return NextResponse.json(
      { error: "Cross-origin request rejected." },
      { status: 403 },
    );
  }
  if (!apiURL) {
    return NextResponse.json({ error: "Database API is unavailable." }, { status: 503 });
  }

  const { path } = await context.params;
  if (!isAllowedPath(path) || path.some((part) => !/^[a-zA-Z0-9._-]+$/.test(part))) {
    return NextResponse.json({ error: "Invalid database API path." }, { status: 404 });
  }
  const incomingURL = new URL(request.url);
  const headers = new Headers({ Authorization: `Bearer ${token}` });
  for (const name of ["content-type", "idempotency-key", "x-request-id"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }
  try {
    const response = await fetch(
      `${apiURL}/api/v1/${path.map(encodeURIComponent).join("/")}${incomingURL.search}`,
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
    const body = response.headers.get("content-type")?.includes("text/event-stream")
      ? response.body
      : await response.arrayBuffer();
    return new NextResponse(body, {
      status: response.status,
      headers: {
        "Content-Type": response.headers.get("content-type") || "application/json",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return NextResponse.json(
      { error: "Database API is temporarily unavailable." },
      { status: 503 },
    );
  }
}

export const GET = proxy;
export const POST = proxy;
