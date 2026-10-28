import { cookies } from "next/headers";
import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ path: string[] }> };

function isAllowedPath(path: string[]): boolean {
  const scopedRoot = path.length >= 5 &&
    path[0] === "tenants" && /^\d+$/.test(path[1]) &&
    path[2] === "projects" && /^\d+$/.test(path[3]);
  if (!scopedRoot) return false;
  if (path.length === 5 && [
    "overview", "resources", "metrics", "logs", "events", "alerts", "dashboards", "integrations",
  ].includes(path[4])) return true;
  if (path.length === 7 && path[4] === "resources" &&
      /^[a-zA-Z0-9._:-]+$/.test(path[5]) &&
      /^[a-zA-Z0-9._:-]+$/.test(path[6])) return true;
  return path.length === 8 && path[4] === "resources" &&
    /^[a-zA-Z0-9._:-]+$/.test(path[5]) &&
    /^[a-zA-Z0-9._:-]+$/.test(path[6]) &&
    path[7] === "metrics";
}

export async function GET(request: Request, context: RouteContext): Promise<NextResponse> {
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json(
      { error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" },
      { status: 401, headers: { "Cache-Control": "no-store" } },
    );
  }
  const apiURL = process.env.API_URL;
  if (!apiURL) {
    return NextResponse.json(
      { error: "Observability API is unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
  const { path } = await context.params;
  if (!isAllowedPath(path)) {
    return NextResponse.json(
      { error: "Invalid observability API path.", code: "INVALID_PATH" },
      { status: 404, headers: { "Cache-Control": "no-store" } },
    );
  }
  const incomingURL = new URL(request.url);
  try {
    const response = await fetch(
      `${apiURL}/api/v1/observability/${path.map(encodeURIComponent).join("/")}${incomingURL.search}`,
      {
        method: "GET",
        headers: { Authorization: `Bearer ${token}` },
        cache: "no-store",
        signal: AbortSignal.timeout(10000),
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
      { error: "Observability API is temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}
