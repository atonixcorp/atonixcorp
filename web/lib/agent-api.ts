const MAX_AGENT_RESPONSE_BYTES = 1024 * 1024;

export async function proxyAgentRequest(
  path: "/v1/agent/registry" | "/v1/agent/predict",
  init: RequestInit = {},
): Promise<Response> {
  const baseUrl = process.env.ATONIX_AGENT_API_URL?.replace(/\/+$/, "");
  const token = process.env.ATONIX_AGENT_API_TOKEN;
  if (!baseUrl || !token) {
    return Response.json(
      { error: "AtonixCorp Agent API is not configured." },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }

  try {
    const upstream = await fetch(`${baseUrl}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
        ...(init.body ? { "Content-Type": "application/json" } : {}),
        ...init.headers,
      },
      cache: "no-store",
      signal: AbortSignal.timeout(10_000),
    });
    const body = await upstream.arrayBuffer();
    if (body.byteLength > MAX_AGENT_RESPONSE_BYTES) {
      return Response.json(
        { error: "Agent API response exceeds the 1 MiB limit." },
        { status: 502, headers: { "Cache-Control": "no-store" } },
      );
    }
    return new Response(body, {
      status: upstream.status,
      headers: {
        "Content-Type": upstream.headers.get("Content-Type") ?? "application/json",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch (error) {
    console.error("AtonixCorp Agent API request failed", error);
    return Response.json(
      { error: "AtonixCorp Agent API is unreachable." },
      { status: 502, headers: { "Cache-Control": "no-store" } },
    );
  }
}
