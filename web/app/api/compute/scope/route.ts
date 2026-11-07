import { NextResponse } from "next/server";
import { cookies } from "next/headers";

export const dynamic = "force-dynamic";

export async function GET(request: Request): Promise<NextResponse> {
  const apiURL = process.env.API_URL;
  const token = (await cookies()).get("atonixcorp_session")?.value;
  if (!token) {
    return NextResponse.json({ error: "Authentication required.", code: "AUTHENTICATION_REQUIRED" }, { status: 401 });
  }
  if (!apiURL) {
    return NextResponse.json({ error: "Compute API is unavailable.", code: "DEPENDENCY_UNAVAILABLE" }, { status: 503 });
  }
  const tenantID = new URL(request.url).searchParams.get("tenant_id");
  if (tenantID && !/^[1-9]\d*$/.test(tenantID)) {
    return NextResponse.json({ error: "Invalid workspace.", code: "INVALID_SCOPE" }, { status: 400 });
  }
  const path = tenantID
    ? `/api/v1/tenants/${encodeURIComponent(tenantID)}/projects`
    : "/api/v1/tenants";
  try {
    const response = await fetch(`${apiURL}${path}`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
      signal: AbortSignal.timeout(15000),
    });
    return new NextResponse(await response.arrayBuffer(), {
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
