import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";
type RouteContext = { params: Promise<{ path: string[] }> };
const idPattern = /^[a-f0-9-]{36}$/;

async function proxy(request: Request, context: RouteContext): Promise<NextResponse> {
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) return NextResponse.json({ error: "Authentication required." }, { status: 401 });
  if (request.method !== "GET" && !isSameOriginRequest(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected." }, { status: 403 });
  }
  const apiURL = process.env.API_URL;
  if (!apiURL) return NextResponse.json({ error: "Storage API is unavailable." }, { status: 503 });
  const { path } = await context.params;
  const allowed =
    request.method === "GET" && path.length === 1 && path[0] === "objects" ||
    request.method === "POST" && path.length === 1 && path[0] === "direct-uploads" ||
    request.method === "POST" && path.length === 3 && path[0] === "direct-uploads" && idPattern.test(path[1]) && path[2] === "complete" ||
    request.method === "GET" && path.length === 2 && idPattern.test(path[0]) && path[1] === "download-url" ||
    request.method === "DELETE" && path.length === 1 && idPattern.test(path[0]);
  if (!allowed) return NextResponse.json({ error: "Invalid file API path." }, { status: 404 });
  const headers = new Headers({ Authorization: `Bearer ${token}` });
  if (request.headers.has("content-type")) headers.set("Content-Type", request.headers.get("content-type")!);
  try {
    const response = await fetch(`${apiURL}/api/v1/files/${path.map(encodeURIComponent).join("/")}${new URL(request.url).search}`, {
      method: request.method,
      headers,
      ...(request.method === "POST" ? { body: await request.text() } : {}),
      cache: "no-store",
      signal: AbortSignal.timeout(600000),
    });
    const responseHeaders = { "Content-Type": response.headers.get("content-type") || "application/json", "Cache-Control": "no-store" };
    if (response.status === 204) return new NextResponse(null, { status: 204, headers: responseHeaders });
    return new NextResponse(response.body, { status: response.status, headers: responseHeaders });
  } catch {
    return NextResponse.json({ error: "Storage API is temporarily unavailable." }, { status: 503 });
  }
}

export const GET = proxy;
export const POST = proxy;
export const DELETE = proxy;
