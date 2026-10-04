import { proxyAgentRequest } from "../../../../lib/agent-api";

const MAX_REQUEST_BYTES = 1024 * 1024;

export async function POST(request: Request): Promise<Response> {
  const contentType = request.headers.get("content-type")?.split(";", 1)[0].trim();
  if (contentType !== "application/json") {
    return Response.json(
      { error: "Content-Type must be application/json." },
      { status: 415, headers: { "Cache-Control": "no-store" } },
    );
  }

  const contentLength = Number(request.headers.get("content-length") ?? 0);
  if (contentLength > MAX_REQUEST_BYTES) {
    return Response.json(
      { error: "Request body exceeds the 1 MiB limit." },
      { status: 413, headers: { "Cache-Control": "no-store" } },
    );
  }

  const body = await request.text();
  if (new TextEncoder().encode(body).byteLength > MAX_REQUEST_BYTES) {
    return Response.json(
      { error: "Request body exceeds the 1 MiB limit." },
      { status: 413, headers: { "Cache-Control": "no-store" } },
    );
  }
  try {
    JSON.parse(body);
  } catch {
    return Response.json(
      { error: "Request body must contain valid JSON." },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }

  return proxyAgentRequest("/v1/agent/predict", { method: "POST", body });
}
