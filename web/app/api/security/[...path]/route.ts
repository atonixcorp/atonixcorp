import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ path: string[] }> };
const maxRequestBodyBytes = 32 * 1024;

class RequestBodyTooLargeError extends Error {}

async function readRequestBody(request: Request): Promise<ArrayBuffer> {
  const declaredLength = request.headers.get("content-length");
  if (declaredLength && /^\d+$/.test(declaredLength) && Number(declaredLength) > maxRequestBodyBytes) {
    throw new RequestBodyTooLargeError();
  }
  if (!request.body) return new ArrayBuffer(0);

  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.byteLength;
    if (total > maxRequestBodyBytes) {
      await reader.cancel().catch(() => undefined);
      throw new RequestBodyTooLargeError();
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

function isAllowedPath(path: string[], method: string): boolean {
  const scopedRoot = path.length >= 5 &&
    path[0] === "tenants" && /^\d+$/.test(path[1]) &&
    path[2] === "projects" && /^\d+$/.test(path[3]);
  if (!scopedRoot) return false;
  if (path.length === 5 && ["overview", "policies", "audit"].includes(path[4])) {
    return path[4] === "policies" ? method === "GET" || method === "POST" : method === "GET";
  }
  if (path.length === 6 && path[4] === "policies" && /^\d+$/.test(path[5])) {
    return method === "GET";
  }
  return path.length === 7 &&
    path[4] === "policies" &&
    /^\d+$/.test(path[5]) &&
    path[6] === "simulate" &&
    method === "POST";
}

async function proxy(request: Request, context: RouteContext): Promise<NextResponse> {
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json(
      { error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" },
      { status: 401, headers: { "Cache-Control": "no-store" } },
    );
  }
  if (request.method !== "GET" && !isSameOriginRequest(request)) {
    return NextResponse.json(
      { error: "Cross-origin request rejected.", code: "CSRF_VALIDATION_FAILED" },
      { status: 403, headers: { "Cache-Control": "no-store" } },
    );
  }
  const apiURL = process.env.API_URL;
  if (!apiURL) {
    return NextResponse.json(
      { error: "Security API is unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }

  const { path } = await context.params;
  if (!isAllowedPath(path, request.method)) {
    return NextResponse.json(
      { error: "Invalid security API path.", code: "INVALID_PATH" },
      { status: 404, headers: { "Cache-Control": "no-store" } },
    );
  }

  const incomingURL = new URL(request.url);
  const headers = new Headers({ Authorization: `Bearer ${token}` });
  for (const name of ["content-type", "idempotency-key", "x-request-id", "x-correlation-id"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }

  let body: ArrayBuffer | undefined;
  if (request.method !== "GET") {
    try {
      body = await readRequestBody(request);
    } catch (error) {
      if (error instanceof RequestBodyTooLargeError) {
        return NextResponse.json(
          { error: "The security request exceeds the allowed size.", code: "REQUEST_TOO_LARGE" },
          { status: 413, headers: { "Cache-Control": "no-store" } },
        );
      }
      throw error;
    }
  }

  try {
    const response = await fetch(
      `${apiURL}/api/v1/security/${path.map(encodeURIComponent).join("/")}${incomingURL.search}`,
      {
        method: request.method,
        headers,
        ...(body === undefined ? {} : { body }),
        cache: "no-store",
        signal: AbortSignal.timeout(10000),
      },
    );
    const responseHeaders = new Headers({
      "Content-Type": response.headers.get("content-type") || "application/json",
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
    });
    for (const name of ["x-request-id", "retry-after"]) {
      const value = response.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }
    return new NextResponse(await response.arrayBuffer(), {
      status: response.status,
      headers: responseHeaders,
    });
  } catch {
    return NextResponse.json(
      { error: "Security API is temporarily unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}

export const GET = proxy;
export const POST = proxy;
