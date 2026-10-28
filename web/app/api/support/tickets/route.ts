import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isSameOriginRequest } from "@/app/shared/lib/request-security";

export const dynamic = "force-dynamic";
const maxRequestBodyBytes = 16 * 1024;
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

export async function GET(): Promise<NextResponse> {
  return proxyTickets("GET");
}

export async function POST(request: Request): Promise<NextResponse> {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json(
      { error: "Cross-origin request rejected.", code: "CSRF_VALIDATION_FAILED" },
      { status: 403, headers: { "Cache-Control": "no-store" } },
    );
  }

  let body: ArrayBuffer;
  try {
    body = await readRequestBody(request);
  } catch (error) {
    if (error instanceof RequestBodyTooLargeError) {
      return NextResponse.json(
        { error: "The support request exceeds the allowed size.", code: "REQUEST_TOO_LARGE" },
        { status: 413, headers: { "Cache-Control": "no-store" } },
      );
    }
    return NextResponse.json(
      { error: "The support request body could not be read.", code: "INVALID_REQUEST_BODY" },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }
  return proxyTickets("POST", body, request.headers.get("content-type") ?? "application/json");
}

async function proxyTickets(method: "GET" | "POST", body?: ArrayBuffer, contentType?: string): Promise<NextResponse> {
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json(
      { error: "Sign in to manage support tickets.", code: "AUTHENTICATION_REQUIRED" },
      { status: 401, headers: { "Cache-Control": "no-store" } },
    );
  }

  const apiURL = process.env.API_URL;
  if (!apiURL) {
    return NextResponse.json(
      { error: "The support service is unavailable.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }

  try {
    const response = await fetch(`${apiURL}/api/v1/support/tickets`, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        ...(contentType ? { "Content-Type": contentType } : {}),
      },
      ...(body ? { body } : {}),
      cache: "no-store",
      signal: AbortSignal.timeout(10000),
    });
    const text = await response.text();
    let payload: unknown;
    try {
      payload = JSON.parse(text);
    } catch {
      return NextResponse.json(
        { error: "The support service returned an invalid response.", code: "INVALID_UPSTREAM_RESPONSE" },
        { status: 502, headers: { "Cache-Control": "no-store" } },
      );
    }
    return NextResponse.json(payload, {
      status: response.status,
      headers: { "Cache-Control": "no-store" },
    });
  } catch {
    return NextResponse.json(
      { error: "The support service could not be reached.", code: "DEPENDENCY_UNAVAILABLE" },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}
